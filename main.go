// Copyright (c) 2025 Broadcom. All Rights Reserved.
// Broadcom Confidential. The term "Broadcom" refers to Broadcom Inc.
// and/or its subsidiaries.

package main

import (
	"log"
	"os"

	"github.com/vmware/etcd-diagnosis/commands"
)

const (
	exitError = 1
)

func main() {
	rootCmd := commands.RootCmd()
	if err := rootCmd.Execute(); err != nil {
		if rootCmd.SilenceErrors {
			log.Printf("Error: %v\n", err)
		}
		os.Exit(exitError)
	}
}
