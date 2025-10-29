// Copyright (c) 2025 Broadcom. All Rights Reserved.
// Broadcom Confidential. The term "Broadcom" refers to Broadcom Inc.
// and/or its subsidiaries.

package commands

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"go.etcd.io/etcd/server/v3/storage/backend"
	"go.uber.org/zap"
)

func NewCommandHash() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hash [data dir or db file path]",
		Short: "hash computes the hash of db file.",
		Args:  cobra.ExactArgs(1),
		Run:   hashCommandFunc,
	}

	return cmd
}

func hashCommandFunc(cmd *cobra.Command, args []string) {
	dp := args[0]
	dp = mustVerifyAndConvertDBPath(dp)

	hash, err := getHash(dp)
	if err != nil {
		log.Fatalf("Failed to get hash for (%s): %v", dp, err)
	}

	outWriter := cmd.OutOrStdout()
	fmt.Fprintf(outWriter, "db path: %s\nHash: %d\n", dp, hash)
}

func getHash(dbPath string) (hash uint32, err error) {
	b := backend.NewDefaultBackend(zap.NewNop(), dbPath)
	return b.Hash(nil)
}
