// Copyright (c) 2025 Broadcom. All Rights Reserved.
// Broadcom Confidential. The term "Broadcom" refers to Broadcom Inc.
// and/or its subsidiaries.

package commands

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/vmware/etcd-diagnosis/version"
)

// NewCommandVersion prints out the version of etcd-diagnosis.
func NewCommandVersion() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Prints the version of etcd-diagnosis",
		Run:   versionCommandFunc,
	}
}

func versionCommandFunc(cmd *cobra.Command, args []string) {
	fmt.Printf("etcd-diagnosis version: %s\n", version.Version)
	fmt.Printf("Git SHA: %s\n", version.GitSHA)
	fmt.Printf("Go Version: %s\n", runtime.Version())
	fmt.Printf("Go OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}
