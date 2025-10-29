// Copyright (c) 2025 Broadcom. All Rights Reserved.
// Broadcom Confidential. The term "Broadcom" refers to Broadcom Inc.
// and/or its subsidiaries.

package commands_test

import (
	"bytes"
	"encoding/binary"
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
	"go.etcd.io/etcd/server/v3/storage/schema"

	"github.com/vmware/etcd-diagnosis/commands"
)

func TestCommandConsistentIndex(t *testing.T) {
	db := commands.MustCreateDB(t)

	consistentIndex := rand.Uint64()
	t.Logf("Generated a random consistent_index: %d", consistentIndex)
	err := db.Update(func(tx *bolt.Tx) error {
		b, berr := tx.CreateBucket(schema.Meta.Name())
		if berr != nil {
			return berr
		}
		bs := make([]byte, 8)
		binary.BigEndian.PutUint64(bs, consistentIndex)
		return b.Put(schema.MetaConsistentIndexKeyName, bs)
	})
	require.NoError(t, err)
	db.MustClose()

	t.Log("Execute consistent-index command")
	rootCmd := commands.RootCmd()
	outputBuf := bytes.NewBuffer(nil)
	rootCmd.SetOut(outputBuf)
	rootCmd.SetArgs([]string{"consistent-index", db.Path()})
	err = rootCmd.Execute()
	require.NoError(t, err)

	t.Log("Check output")
	actualOutput := strings.TrimSpace(outputBuf.String())
	val, err := strconv.ParseUint(actualOutput, 10, 64)
	require.NoError(t, err)
	require.Equal(t, consistentIndex, val)
}
