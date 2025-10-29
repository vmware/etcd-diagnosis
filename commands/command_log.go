// Copyright (c) 2025 Broadcom. All Rights Reserved.
// Broadcom Confidential. The term "Broadcom" refers to Broadcom Inc.
// and/or its subsidiaries.

package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/client/pkg/v3/fileutil"
	"go.etcd.io/etcd/client/pkg/v3/types"
	"go.etcd.io/etcd/pkg/v3/pbutil"
	"go.etcd.io/etcd/server/v3/etcdserver/api/snap"
	"go.etcd.io/etcd/server/v3/storage/datadir"
	"go.etcd.io/etcd/server/v3/storage/wal"
	"go.etcd.io/etcd/server/v3/storage/wal/walpb"
	"go.etcd.io/raft/v3/raftpb"
	"go.uber.org/zap"
)

const (
	defaultEntryTypes string = "Normal,ConfigChange"
	methodSync        string = "SYNC"
	methodQGet        string = "QGET"
	methodDelete      string = "DELETE"
)

var (
	snapFile   string
	walDir     string
	startIndex uint64
	endIndex   uint64
	entryType  string
	raw        bool
)

func NewCommandLog() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "log [data dir]",
		Short: "log dumps the log from data directory.",
		Args:  cobra.ExactArgs(1),
		Run:   logCommandFunc,
	}

	cmd.Flags().StringVar(&snapFile, "start-snap", "", "the base name of snapshot file to start dumping")
	cmd.Flags().StringVar(&walDir, "wal-dir", "", "path to the dedicated wal directory, defaults to '${data_dir}/member/wal/' if not set")
	cmd.Flags().Uint64Var(&startIndex, "start-index", 0, "the index to start dumping (inclusive). If unspecified, dumps from the index of the last snapshot")
	cmd.Flags().Uint64Var(&endIndex, "end-index", math.MaxUint64, "the index to stop dumping (exclusive)")
	cmd.Flags().StringVar(&entryType, "entry-type", defaultEntryTypes, `If set, filters output by entry type. Must be one or more than one of:
ConfigChange, Normal, Request, InternalRaftRequest,
IRRRange, IRRPut, IRRDeleteRange, IRRTxn,
IRRCompaction, IRRLeaseGrant, IRRLeaseRevoke, IRRLeaseCheckpoint`)
	cmd.Flags().BoolVar(&raw, "raw", false, "read the logs in the low-level form")

	return cmd
}

func logCommandFunc(cmd *cobra.Command, args []string) {
	dp := args[0]
	mustVerifyDir(dp)

	if snapFile != "" && startIndex != 0 {
		log.Fatal("--start-snap and --start-index are mutually exclusive.")
	}

	startFromIndex := cmd.Flags().Changed("start-index")
	if !raw {
		ents := readUsingReadAll(startFromIndex, startIndex, endIndex, snapFile, dp, walDir)

		fmt.Printf("WAL entries: %d\n", len(ents))
		if len(ents) > 0 {
			fmt.Printf("lastIndex=%d\n", ents[len(ents)-1].Index)
		}

		fmt.Printf("%4s\t%10s\ttype\tdata", "term", "index")
		fmt.Println()

		listEntriesType(entryType, ents)
	} else {
		if snapFile != "" || entryType != defaultEntryTypes {
			log.Fatalf("Flags --start-snap and --entry-type not supported in the RAW mode.")
		}

		if walDir == "" {
			walDir = datadir.ToWALDir(dp)
		}
		readRaw(startIndex, walDir, os.Stdout)
	}
}

func readUsingReadAll(startFromIndex bool, startIndex uint64, endIndex uint64, snapfile string, dataDir string, waldir string) []raftpb.Entry {
	var (
		walsnap  walpb.Snapshot
		snapshot *raftpb.Snapshot
		err      error
	)

	lg := zap.NewExample()

	endAtIndex := endIndex < math.MaxUint64
	if startFromIndex {
		fmt.Printf("Start dumping log entries from index %d.\n", startIndex)
		// ReadAll() reads entries from the index after walsnap.Index, so we need to move walsnap.Index back one.
		if startIndex > 0 {
			startIndex--
		}
		walsnap.Index = startIndex
	} else {
		if snapfile == "" {
			ss := snap.New(lg, datadir.ToSnapDir(dataDir))
			snapshot, err = ss.Load()
		} else {
			snapshot, err = snap.Read(lg, filepath.Join(datadir.ToSnapDir(dataDir), snapfile))
		}

		switch {
		case err == nil:
			walsnap.Index, walsnap.Term = snapshot.Metadata.Index, snapshot.Metadata.Term
			nodes := genIDSlice(snapshot.Metadata.ConfState.Voters)

			confStateJSON, merr := json.Marshal(snapshot.Metadata.ConfState)
			if merr != nil {
				confStateJSON = []byte(fmt.Sprintf("confstate err: %v", merr))
			}
			fmt.Printf("Snapshot:\nterm=%d index=%d nodes=%s confstate=%s\n",
				walsnap.Term, walsnap.Index, nodes, confStateJSON)
		case errors.Is(err, snap.ErrNoSnapshot):
			fmt.Print("Snapshot:\nempty\n")
		default:
			log.Fatalf("Failed loading snapshot: %v", err)
		}
		fmt.Println("Start dumping log entries from snapshot.")
	}

	wd := waldir
	if wd == "" {
		wd = datadir.ToWALDir(dataDir)
	}

	w, err := wal.OpenForRead(zap.NewExample(), wd, walsnap)
	if err != nil {
		log.Fatalf("Failed opening WAL: %v", err)
	}
	wmetadata, state, ents, err := w.ReadAll()
	w.Close()
	if err != nil && (!startFromIndex || !errors.Is(err, wal.ErrSnapshotNotFound)) {
		// ReadAll might return ErrSliceOutOfRange and the first series of entries if the server is offline for a while and receives a snapshot from leader.
		// It is ok to ignore ErrSliceOutOfRange if just requesting a specific range of entries
		if !endAtIndex || !errors.Is(err, wal.ErrSliceOutOfRange) {
			log.Fatalf("Failed reading WAL: %v", err)
		}
		log.Printf("Failed reading all WAL: %v", err)
	}
	id, cid := parseWALMetadata(wmetadata)
	vid := types.ID(state.Vote)
	fmt.Printf("WAL metadata:\nnodeID=%s clusterID=%s term=%d commitIndex=%d vote=%s\n",
		id, cid, state.Term, state.Commit, vid)
	if endAtIndex {
		entries := make([]raftpb.Entry, 0)
		for _, e := range ents {
			// WAL might contain entries with e.Index >= *endIndex from prev term, then e.Index < *endIndex in the next term.
			// We cannot break when e.Index >= *endIndex.
			if e.Index >= endIndex {
				continue
			}
			entries = append(entries, e)
		}
		return entries
	}
	return ents
}

func parseWALMetadata(b []byte) (id, cid types.ID) {
	var metadata etcdserverpb.Metadata
	pbutil.MustUnmarshal(&metadata, b)
	id = types.ID(metadata.NodeID)
	cid = types.ID(metadata.ClusterID)
	return id, cid
}

func genIDSlice(a []uint64) []types.ID {
	ids := make([]types.ID, len(a))
	for i, id := range a {
		ids[i] = types.ID(id)
	}
	return ids
}

// excerpt replaces middle part with ellipsis and returns a double-quoted
// string safely escaped with Go syntax.
func excerpt(str string, pre, suf int) string {
	if pre+suf > len(str) {
		return fmt.Sprintf("%q", str)
	}
	return fmt.Sprintf("%q...%q", str[:pre], str[len(str)-suf:])
}

type EntryFilter func(e raftpb.Entry) (bool, string)

// The 9 pass functions below takes the raftpb.Entry and return if the entry should be printed and the type of entry,
// the type of the entry will used in the following print function
func passConfChange(entry raftpb.Entry) (bool, string) {
	return entry.Type == raftpb.EntryConfChange, "ConfigChange"
}

func passInternalRaftRequest(entry raftpb.Entry) (bool, string) {
	var rr etcdserverpb.InternalRaftRequest
	return entry.Type == raftpb.EntryNormal && rr.Unmarshal(entry.Data) == nil, "InternalRaftRequest"
}

func passUnknownNormal(entry raftpb.Entry) (bool, string) {
	var rr1 etcdserverpb.Request
	var rr2 etcdserverpb.InternalRaftRequest
	return (entry.Type == raftpb.EntryNormal) && (rr1.Unmarshal(entry.Data) != nil) && (rr2.Unmarshal(entry.Data) != nil), "UnknownNormal"
}

func passIRRRange(entry raftpb.Entry) (bool, string) {
	var rr etcdserverpb.InternalRaftRequest
	return entry.Type == raftpb.EntryNormal && rr.Unmarshal(entry.Data) == nil && rr.Range != nil, "InternalRaftRequest"
}

func passIRRPut(entry raftpb.Entry) (bool, string) {
	var rr etcdserverpb.InternalRaftRequest
	return entry.Type == raftpb.EntryNormal && rr.Unmarshal(entry.Data) == nil && rr.Put != nil, "InternalRaftRequest"
}

func passIRRDeleteRange(entry raftpb.Entry) (bool, string) {
	var rr etcdserverpb.InternalRaftRequest
	return entry.Type == raftpb.EntryNormal && rr.Unmarshal(entry.Data) == nil && rr.DeleteRange != nil, "InternalRaftRequest"
}

func passIRRTxn(entry raftpb.Entry) (bool, string) {
	var rr etcdserverpb.InternalRaftRequest
	return entry.Type == raftpb.EntryNormal && rr.Unmarshal(entry.Data) == nil && rr.Txn != nil, "InternalRaftRequest"
}

func passIRRCompaction(entry raftpb.Entry) (bool, string) {
	var rr etcdserverpb.InternalRaftRequest
	return entry.Type == raftpb.EntryNormal && rr.Unmarshal(entry.Data) == nil && rr.Compaction != nil, "InternalRaftRequest"
}

func passIRRLeaseGrant(entry raftpb.Entry) (bool, string) {
	var rr etcdserverpb.InternalRaftRequest
	return entry.Type == raftpb.EntryNormal && rr.Unmarshal(entry.Data) == nil && rr.LeaseGrant != nil, "InternalRaftRequest"
}

func passIRRLeaseRevoke(entry raftpb.Entry) (bool, string) {
	var rr etcdserverpb.InternalRaftRequest
	return entry.Type == raftpb.EntryNormal && rr.Unmarshal(entry.Data) == nil && rr.LeaseRevoke != nil, "InternalRaftRequest"
}

func passIRRLeaseCheckpoint(entry raftpb.Entry) (bool, string) {
	var rr etcdserverpb.InternalRaftRequest
	return entry.Type == raftpb.EntryNormal && rr.Unmarshal(entry.Data) == nil && rr.LeaseCheckpoint != nil, "InternalRaftRequest"
}

func passRequest(entry raftpb.Entry) (bool, string) {
	var rr1 etcdserverpb.Request
	var rr2 etcdserverpb.InternalRaftRequest
	return entry.Type == raftpb.EntryNormal && rr1.Unmarshal(entry.Data) == nil && rr2.Unmarshal(entry.Data) != nil, "Request"
}

type EntryPrinter func(e raftpb.Entry)

// The 4 print functions below print the entry format based on there types

// printInternalRaftRequest is used to print entry information for IRRRange, IRRPut,
// IRRDeleteRange and IRRTxn entries
func printInternalRaftRequest(entry raftpb.Entry) {
	var rr etcdserverpb.InternalRaftRequest
	if err := rr.Unmarshal(entry.Data); err == nil {
		// Ensure we don't log user password
		if rr.AuthUserChangePassword != nil && rr.AuthUserChangePassword.Password != "" {
			rr.AuthUserChangePassword.Password = "<value removed>"
		}
		fmt.Printf("%4d\t%10d\tnorm\t%s", entry.Term, entry.Index, rr.String())
	}
}

func printUnknownNormal(entry raftpb.Entry) {
	fmt.Printf("%4d\t%10d\tnorm\t???", entry.Term, entry.Index)
}

func printConfChange(entry raftpb.Entry) {
	fmt.Printf("%4d\t%10d", entry.Term, entry.Index)
	fmt.Print("\tconf")
	var r raftpb.ConfChange
	if err := r.Unmarshal(entry.Data); err != nil {
		fmt.Print("\t???")
	} else {
		fmt.Printf("\tmethod=%s id=%s", r.Type, types.ID(r.NodeID))
	}
}

func printRequest(entry raftpb.Entry) {
	var r etcdserverpb.Request
	if err := r.Unmarshal(entry.Data); err == nil {
		fmt.Printf("%4d\t%10d\tnorm", entry.Term, entry.Index)
		switch r.Method {
		case "":
			fmt.Print("\tnoop")
		case methodSync:
			fmt.Printf("\tmethod=SYNC time=%q", time.Unix(0, r.Time).UTC())
		case methodQGet, methodDelete:
			fmt.Printf("\tmethod=%s path=%s", r.Method, excerpt(r.Path, 64, 64))
		default:
			fmt.Printf("\tmethod=%s path=%s val=%s", r.Method, excerpt(r.Path, 64, 64), excerpt(r.Val, 128, 0))
		}
	}
}

// evaluateEntrytypeFlag evaluates entry-type flag and choose proper filter/filters to filter entries
func evaluateEntrytypeFlag(entrytype string) []EntryFilter {
	var entrytypelist []string
	if entrytype != "" {
		entrytypelist = strings.Split(entrytype, ",")
	}

	validRequest := map[string][]EntryFilter{
		"ConfigChange":        {passConfChange},
		"Normal":              {passInternalRaftRequest, passRequest, passUnknownNormal},
		"Request":             {passRequest},
		"InternalRaftRequest": {passInternalRaftRequest},
		"IRRRange":            {passIRRRange},
		"IRRPut":              {passIRRPut},
		"IRRDeleteRange":      {passIRRDeleteRange},
		"IRRTxn":              {passIRRTxn},
		"IRRCompaction":       {passIRRCompaction},
		"IRRLeaseGrant":       {passIRRLeaseGrant},
		"IRRLeaseRevoke":      {passIRRLeaseRevoke},
		"IRRLeaseCheckpoint":  {passIRRLeaseCheckpoint},
	}
	filters := make([]EntryFilter, 0)
	for _, et := range entrytypelist {
		if f, ok := validRequest[et]; ok {
			filters = append(filters, f...)
		} else {
			log.Printf(`[%+v] is not a valid entry-type, ignored.
Please set entry-type to one or more of the following:
ConfigChange, Normal, Request, InternalRaftRequest,
IRRRange, IRRPut, IRRDeleteRange, IRRTxn,
IRRCompaction, IRRLeaseGrant, IRRLeaseRevoke, IRRLeaseCheckpoint`, et)
		}
	}

	return filters
}

// listEntriesType filters and prints entries based on the entry-type flag,
func listEntriesType(entrytype string, ents []raftpb.Entry) {
	entryFilters := evaluateEntrytypeFlag(entrytype)
	printerMap := map[string]EntryPrinter{
		"InternalRaftRequest": printInternalRaftRequest,
		"Request":             printRequest,
		"ConfigChange":        printConfChange,
		"UnknownNormal":       printUnknownNormal,
	}

	cnt := 0

	for _, e := range ents {
		passed := false
		currtype := ""
		for _, filter := range entryFilters {
			passed, currtype = filter(e)
			if passed {
				cnt++
				break
			}
		}
		if passed {
			printer := printerMap[currtype]
			printer(e)
			fmt.Println()
		}
	}

	fmt.Printf("\nEntry types (%s) count is : %d\n", entrytype, cnt)
}

func readRaw(fromIndex uint64, waldir string, out io.Writer) {
	var walReaders []fileutil.FileReader
	dirEntry, err := os.ReadDir(waldir)
	if err != nil {
		log.Fatalf("Error: Failed to read directory '%s' error:%v", waldir, err)
	}
	for _, e := range dirEntry {
		finfo, err := e.Info()
		if err != nil {
			log.Fatalf("Error: failed to get fileInfo of file: %s, error: %v", e.Name(), err)
		}
		if filepath.Ext(finfo.Name()) != ".wal" {
			log.Printf("Warning: Ignoring not .wal file: %s", finfo.Name())
			continue
		}
		f, err := os.Open(filepath.Join(waldir, finfo.Name()))
		if err != nil {
			log.Printf("Error: Failed to read file: %s . error:%v", finfo.Name(), err)
		}
		walReaders = append(walReaders, fileutil.NewFileReader(f))
	}
	decoder := wal.NewDecoderAdvanced(true, walReaders...)
	// The variable is used to not pollute log with multiple continuous crc errors.
	crcDesync := false
	for {
		rec := walpb.Record{}
		err := decoder.Decode(&rec)
		if err == nil || errors.Is(err, walpb.ErrCRCMismatch) {
			if err != nil && !crcDesync {
				log.Printf("Error: Reading entry failed with CRC error: %c", err)
				crcDesync = true
			}
			printRec(&rec, fromIndex, out)
			if rec.Type == wal.CrcType {
				decoder.UpdateCRC(rec.Crc)
				crcDesync = false
			}
			continue
		}
		if errors.Is(err, io.EOF) {
			fmt.Fprintf(out, "EOF: All entries were processed.\n")
			break
		} else if errors.Is(err, io.ErrUnexpectedEOF) {
			fmt.Fprintf(out, "ErrUnexpectedEOF: The last record might be corrupted, error: %v.\n", err)
			break
		}
		log.Printf("Error: Reading failed: %v", err)
		break
	}
}

func printRec(rec *walpb.Record, fromIndex uint64, out io.Writer) {
	switch rec.Type {
	case wal.MetadataType:
		var metadata etcdserverpb.Metadata
		pbutil.MustUnmarshal(&metadata, rec.Data)
		fmt.Fprintf(out, "Metadata: %s\n", metadata.String())
	case wal.CrcType:
		fmt.Fprintf(out, "CRC: %d\n", rec.Crc)
	case wal.EntryType:
		e := wal.MustUnmarshalEntry(rec.Data)
		if e.Index >= fromIndex {
			fmt.Fprintf(out, "Entry: %s\n", e.String())
		}
	case wal.SnapshotType:
		var snap walpb.Snapshot
		pbutil.MustUnmarshal(&snap, rec.Data)
		if snap.Index >= fromIndex {
			fmt.Fprintf(out, "Snapshot: %s\n", snap.String())
		}
	case wal.StateType:
		var state raftpb.HardState
		pbutil.MustUnmarshal(&state, rec.Data)
		if state.Commit >= fromIndex {
			fmt.Fprintf(out, "HardState: %s\n", state.String())
		}
	default:
		log.Printf("Unexpected WAL log type: %d", rec.Type)
	}
}
