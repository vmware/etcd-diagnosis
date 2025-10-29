// Copyright (c) 2025 Broadcom. All Rights Reserved.
// Broadcom Confidential. The term "Broadcom" refers to Broadcom Inc.
// and/or its subsidiaries.

package commands

import (
	"time"

	"github.com/spf13/cobra"
	bbolt_cmd "go.etcd.io/bbolt/cmd/bbolt/command"

	"github.com/vmware/etcd-diagnosis/commands/report"
)

const (
	cliName        = "etcd-diagnosis"
	cliDescription = "A comprehensive etcd diagnosis tool"
)

var (
	flockTimeout time.Duration

	rootCmd = &cobra.Command{
		Use:   cliName,
		Short: cliDescription,
	}
)

func init() {
	rootCmd.PersistentFlags().DurationVar(&flockTimeout, "timeout", 10*time.Second, "time to wait to obtain a file lock on db file, 0 to block indefinitely")

	rootCmd.AddCommand(
		NewCommandVersion(),
		NewCommandListBucket(),
		NewCommandIterateBucket(),
		NewCommandHash(),
		NewCommandLog(),
		NewCommandConsistentIndex(),
		NewCommandCommitIndex(),
		report.NewCommandReport(),
		bbolt_cmd.NewRootCommand(),
	)
}

func RootCmd() *cobra.Command {
	return rootCmd
}
