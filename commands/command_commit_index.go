// Copyright (c) 2025 Broadcom. All Rights Reserved.
// Broadcom Confidential. The term "Broadcom" refers to Broadcom Inc.
// and/or its subsidiaries.

package commands

import (
	"errors"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"go.etcd.io/etcd/server/v3/etcdserver/api/snap"
	"go.etcd.io/etcd/server/v3/storage/datadir"
	"go.etcd.io/etcd/server/v3/storage/wal"
	"go.etcd.io/etcd/server/v3/storage/wal/walpb"
	"go.etcd.io/raft/v3/raftpb"
	"go.uber.org/zap"
)

func NewCommandCommitIndex() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commit-index [data dir]",
		Short: "commit-index reads the commit index from data directory.",
		Args:  cobra.ExactArgs(1),
		Run:   commitIndexCommandFunc,
	}

	cmd.Flags().StringVar(&walDir, "wal-dir", "", "path to the dedicated wal directory, defaults to '${data_dir}/member/wal/' if not set")

	return cmd
}

func commitIndexCommandFunc(cmd *cobra.Command, args []string) {
	dp := args[0]
	mustVerifyDir(dp)

	if walDir == "" {
		walDir = datadir.ToWALDir(dp)
	}
	mustVerifyDir(walDir)

	commitIndex, err := getCommitIndex(dp, walDir)
	if err != nil {
		log.Fatalf("Error reading commit index: %v", err)
	}
	consistentIndex, err := getConsistentIndex(datadir.ToBackendFileName(dp))
	if err != nil {
		log.Fatalf("Error reading consistent index: %v", err)
	}

	outWriter := cmd.OutOrStdout()
	fmt.Fprintf(outWriter, "%d\n", max(commitIndex, consistentIndex))
}

func getCommitIndex(dp string, walDir string) (uint64, error) {
	var (
		walsnap   walpb.Snapshot
		snapshot  *raftpb.Snapshot
		hardState raftpb.HardState
		err       error
	)

	ss := snap.New(zap.NewExample(), datadir.ToSnapDir(dp))
	snapshot, err = ss.Load()
	if err != nil {
		if !errors.Is(err, snap.ErrNoSnapshot) {
			return 0, fmt.Errorf("error loading snapshot: %w", err)
		}
	} else {
		walsnap.Index, walsnap.Term = snapshot.Metadata.Index, snapshot.Metadata.Term
	}

	w, err := wal.OpenForRead(zap.NewExample(), walDir, walsnap)
	if err != nil {
		return 0, fmt.Errorf("error opening WAL: %w", err)
	}
	// ignore the error for now. There is a minor bug upstream,
	// it always returns "invalid argument".
	_ = w.Close()

	_, hardState, _, err = w.ReadAll()
	if err != nil {
		if !errors.Is(err, wal.ErrSliceOutOfRange) {
			return 0, fmt.Errorf("error reading WAL: %w", err)
		}
	}

	return hardState.Commit, nil
}
