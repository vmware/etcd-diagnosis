// Copyright (c) 2025 Broadcom. All Rights Reserved.
// Broadcom Confidential. The term "Broadcom" refers to Broadcom Inc.
// and/or its subsidiaries.

package commands_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"

	"github.com/vmware/etcd-diagnosis/commands"
)

func TestCommandHash(t *testing.T) {
	data := map[string][][]string{
		"bucket1": {
			{"key01", "value01"},
			{"key02", "value02"},
			{"key03", "value03"},
		},
		"bucket2": {
			{"key10", "value10"},
			{"key11", "value11"},
			{"key12", "value12"},
		},
		"bucket3": {
			{"key20", "value20"},
			{"key21", "value21"},
			{"key22", "value22"},
		},
	}

	results := make([]string, 2)
	for i := 0; i < 2; i++ {
		t.Logf("Round %d", i)

		db := commands.MustCreateDB(t)
		t.Log("Populate data")
		err := db.Update(func(tx *bolt.Tx) error {
			for bucket, kv := range data {
				b, berr := tx.CreateBucket([]byte(bucket))
				if berr != nil {
					return berr
				}

				for _, kv := range kv {
					if pErr := b.Put([]byte(kv[0]), []byte(kv[1])); pErr != nil {
						return pErr
					}
				}
			}
			return nil
		})
		require.NoError(t, err)
		db.MustClose()

		t.Log("Execute hash command")
		rootCmd := commands.RootCmd()
		outputBuf := bytes.NewBuffer(nil)
		rootCmd.SetOut(outputBuf)
		rootCmd.SetArgs([]string{"hash", db.Path()})
		err = rootCmd.Execute()
		require.NoError(t, err)

		t.Log("Check output")
		actualOutput := outputBuf.String()
		require.Contains(t, actualOutput, fmt.Sprintf("db path: %s\nHash:", db.Path()))

		idx := strings.Index(actualOutput, "Hash:")
		results[i] = actualOutput[idx:]
	}

	t.Log("Verify the two results are identical")
	require.Equal(t, results[0], results[1])
}
