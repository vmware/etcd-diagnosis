// Copyright (c) 2025 Broadcom. All Rights Reserved.
// Broadcom Confidential. The term "Broadcom" refers to Broadcom Inc.
// and/or its subsidiaries.

package commands_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"

	"github.com/vmware/etcd-diagnosis/commands"
)

func TestCommandListBucket(t *testing.T) {
	db := commands.MustCreateDB(t)

	t.Log("Prepare data: create a db file with several buckets")
	err := db.Update(func(tx *bolt.Tx) error {
		buckets := []string{"bucket1", "bucket2", "bucket3"}
		for _, bucket := range buckets {
			if _, berr := tx.CreateBucket([]byte(bucket)); berr != nil {
				return berr
			}
		}
		return nil
	})
	require.NoError(t, err)
	db.MustClose()

	t.Log("Execute list bucket command")
	rootCmd := commands.RootCmd()
	outputBuf := bytes.NewBuffer(nil)
	rootCmd.SetOut(outputBuf)
	rootCmd.SetArgs([]string{"list-bucket", db.Path()})
	err = rootCmd.Execute()
	require.NoError(t, err)

	t.Log("Check output")
	actualOutput := outputBuf.String()
	require.Equal(t, "bucket1\nbucket2\nbucket3\n", actualOutput)
}

func TestCommandIterateBucket(t *testing.T) {
	db := commands.MustCreateDB(t)

	kvData := [][]string{
		{"key1", "value1"},
		{"key2", "value2"},
		{"key3", "value3"},
	}

	t.Log("Prepare data: create a db file with a bucket and populate the bucket")
	err := db.Update(func(tx *bolt.Tx) error {
		b, berr := tx.CreateBucket([]byte("data"))
		if berr != nil {
			return berr
		}
		for _, kv := range kvData {
			if pErr := b.Put([]byte(kv[0]), []byte(kv[1])); pErr != nil {
				return pErr
			}
		}
		return nil
	})
	require.NoError(t, err)
	db.MustClose()

	t.Log("Execute iterate bucket command")
	rootCmd := commands.RootCmd()
	outputBuf := bytes.NewBuffer(nil)
	rootCmd.SetOut(outputBuf)
	rootCmd.SetArgs([]string{"iterate-bucket", db.Path(), "data"})
	err = rootCmd.Execute()
	require.NoError(t, err)

	t.Log("Check output")
	actualOutput := outputBuf.String()
	expectedOutput := `key="key3", value="value3"
key="key2", value="value2"
key="key1", value="value1"
`
	require.Equal(t, expectedOutput, actualOutput)
}
