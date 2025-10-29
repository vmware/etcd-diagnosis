// Copyright (c) 2025 Broadcom. All Rights Reserved.
// Broadcom Confidential. The term "Broadcom" refers to Broadcom Inc.
// and/or its subsidiaries.

package commands

import (
	"log"
	"os"

	"go.etcd.io/etcd/server/v3/storage/datadir"
)

func mustVerifyAndConvertDBPath(dp string) string {
	if ok, err := exist(dp); !ok || err != nil {
		log.Fatalf("%s does not exist, error: %v", dp, err)
	}

	if isDir(dp) {
		dp = datadir.ToBackendFileName(dp)
	}

	if ok, err := exist(dp); !ok || err != nil {
		log.Fatalf("%s does not exist, error: %v", dp, err)
	}

	return dp
}

func mustVerifyDir(dp string) {
	if ok, err := exist(dp); !ok || err != nil {
		log.Fatalf("%s does not exist, error: %v", dp, err)
	}

	if !isDir(dp) {
		log.Fatalf("%s is not a directory", dp)
	}
}

// exist returns true if a file or directory exists.
func exist(name string) (bool, error) {
	_, err := os.Stat(name)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func isDir(name string) bool {
	info, err := os.Stat(name)
	return err == nil && info.IsDir()
}
