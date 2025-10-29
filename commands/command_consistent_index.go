// Copyright (c) 2025 Broadcom. All Rights Reserved.
// Broadcom Confidential. The term "Broadcom" refers to Broadcom Inc.
// and/or its subsidiaries.

package commands

import (
	"encoding/binary"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	bolt "go.etcd.io/bbolt"
	"go.etcd.io/etcd/server/v3/storage/schema"
)

func NewCommandConsistentIndex() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "consistent-index [data dir or db file path]",
		Short: "consistent-index reads consistent_index from the meta bucket in the db file.",
		Args:  cobra.ExactArgs(1),
		Run:   consistentIndexCommandFunc,
	}

	return cmd
}

func consistentIndexCommandFunc(cmd *cobra.Command, args []string) {
	dp := args[0]
	dp = mustVerifyAndConvertDBPath(dp)

	consistentIndex, err := getConsistentIndex(dp)
	if err != nil {
		log.Fatalf("Failed to read consistent index from (%s): %v", dp, err)
	}

	outWriter := cmd.OutOrStdout()
	fmt.Fprintf(outWriter, "%d\n", consistentIndex)
}

func getConsistentIndex(dbPath string) (uint64, error) {
	db, err := bolt.Open(dbPath, 0o600, &bolt.Options{ReadOnly: true})
	if err != nil {
		return 0, fmt.Errorf("failed to open database (%s): %w", dbPath, err)
	}

	var consistentIndex uint64
	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(schema.Meta.Name())
		if b == nil {
			return fmt.Errorf("bucket (%s) not found", schema.Meta.Name())
		}
		v := b.Get(schema.MetaConsistentIndexKeyName)
		if v == nil {
			return fmt.Errorf("key (%s) not found", schema.MetaConsistentIndexKeyName)
		}
		consistentIndex = binary.BigEndian.Uint64(v)
		return nil
	})
	return consistentIndex, err
}
