// Copyright (c) 2025 Broadcom. All Rights Reserved.
// Broadcom Confidential. The term "Broadcom" refers to Broadcom Inc.
// and/or its subsidiaries.

package commands

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"

	"github.com/spf13/cobra"
	bolt "go.etcd.io/bbolt"
	"go.etcd.io/etcd/api/v3/authpb"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"go.etcd.io/etcd/server/v3/lease/leasepb"
	"go.etcd.io/etcd/server/v3/storage/mvcc"
	"go.etcd.io/etcd/server/v3/storage/schema"
)

var (
	iterateBucketLimit  uint64
	iterateBucketDecode bool
)

func NewCommandListBucket() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-bucket [data dir or db file path]",
		Short: "list-bucket lists all buckets.",
		Args:  cobra.ExactArgs(1),
		Run:   listBucketCommandFunc,
	}

	return cmd
}

func NewCommandIterateBucket() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "iterate-bucket [data dir or db file path] [bucket name]",
		Short: "iterate-bucket lists key-value pairs in reverse order.",
		Args:  cobra.ExactArgs(2),
		Run:   iterateBucketCommandFunc,
	}

	cmd.Flags().Uint64Var(&iterateBucketLimit, "limit", 0, "max number of key-value pairs to iterate (0 to iterate all)")
	cmd.Flags().BoolVar(&iterateBucketDecode, "decode", false, "true to decode Protocol Buffer encoded data")

	return cmd
}

func listBucketCommandFunc(cmd *cobra.Command, args []string) {
	dp := args[0]
	dp = mustVerifyAndConvertDBPath(dp)

	bts, err := getBuckets(dp)
	if err != nil {
		log.Fatalf("Failed to get buckets: %v", err)
	}

	outWriter := cmd.OutOrStdout()
	for _, b := range bts {
		fmt.Fprintln(outWriter, b)
	}
}

func getBuckets(dbPath string) (buckets []string, err error) {
	db, derr := bolt.Open(dbPath, 0o600, &bolt.Options{Timeout: flockTimeout})
	if derr != nil {
		return nil, fmt.Errorf("failed to open bolt DB, %w", derr)
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(b []byte, _ *bolt.Bucket) error {
			buckets = append(buckets, string(b))
			return nil
		})
	})
	return buckets, err
}

func iterateBucketCommandFunc(cmd *cobra.Command, args []string) {
	dp := args[0]
	dp = mustVerifyAndConvertDBPath(dp)

	bucket := args[1]
	outWriter := cmd.OutOrStdout()
	err := iterateBucket(dp, bucket, iterateBucketLimit, iterateBucketDecode, outWriter)
	if err != nil {
		log.Fatalf("Failed to iterate bucket: %v", err)
	}
}

type decoder func(k, v []byte, writer io.Writer)

// key is the bucket name, and value is the function to decode K/V in the bucket.
var decoders = map[string]decoder{
	"key":       keyDecoder,
	"lease":     leaseDecoder,
	"auth":      authDecoder,
	"authRoles": authRolesDecoder,
	"authUsers": authUsersDecoder,
	"meta":      metaDecoder,
}

func defaultDecoder(k, v []byte, writer io.Writer) {
	fmt.Fprintf(writer, "key=%q, value=%q\n", k, v)
}

func keyDecoder(k, v []byte, writer io.Writer) {
	rev := mvcc.BytesToBucketKey(k)
	var kv mvccpb.KeyValue
	if err := kv.Unmarshal(v); err != nil {
		panic(err)
	}
	fmt.Fprintf(writer, "rev=%+v, value=[key %q | val %q | created %d | mod %d | ver %d]\n", rev, string(kv.Key), string(kv.Value), kv.CreateRevision, kv.ModRevision, kv.Version)
}

func bytesToLeaseID(bytes []byte) int64 {
	if len(bytes) != 8 {
		panic(fmt.Errorf("lease ID must be 8-byte"))
	}
	return int64(binary.BigEndian.Uint64(bytes))
}

func leaseDecoder(k, v []byte, writer io.Writer) {
	leaseID := bytesToLeaseID(k)
	var lpb leasepb.Lease
	if err := lpb.Unmarshal(v); err != nil {
		panic(err)
	}
	fmt.Fprintf(writer, "lease ID=%016x, TTL=%ds, remaining TTL=%ds\n", leaseID, lpb.TTL, lpb.RemainingTTL)
}

func authDecoder(k, v []byte, writer io.Writer) {
	if string(k) == "authRevision" {
		rev := binary.BigEndian.Uint64(v)
		fmt.Fprintf(writer, "key=%q, value=%v\n", k, rev)
	} else {
		fmt.Fprintf(writer, "key=%q, value=%v\n", k, v)
	}
}

func authRolesDecoder(_, v []byte, writer io.Writer) {
	role := &authpb.Role{}
	err := role.Unmarshal(v)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(writer, "role=%q, keyPermission=%v\n", string(role.Name), role.KeyPermission)
}

func authUsersDecoder(_, v []byte, writer io.Writer) {
	user := &authpb.User{}
	err := user.Unmarshal(v)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(writer, "user=%q, roles=%q, option=%v\n", user.Name, user.Roles, user.Options)
}

func metaDecoder(k, v []byte, writer io.Writer) {
	if string(k) == string(schema.MetaConsistentIndexKeyName) || string(k) == string(schema.MetaTermKeyName) {
		fmt.Fprintf(writer, "key=%q, value=%v\n", k, binary.BigEndian.Uint64(v))
	} else if string(k) == string(schema.ScheduledCompactKeyName) || string(k) == string(schema.FinishedCompactKeyName) {
		rev := mvcc.BytesToRev(v)
		fmt.Fprintf(writer, "key=%q, value=%v\n", k, rev)
	} else {
		defaultDecoder(k, v, writer)
	}
}

func iterateBucket(dbPath, bucket string, limit uint64, decode bool, writer io.Writer) (err error) {
	db, err := bolt.Open(dbPath, 0o600, &bolt.Options{Timeout: flockTimeout})
	if err != nil {
		return fmt.Errorf("failed to open bolt DB %w", err)
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucket))
		if b == nil {
			return fmt.Errorf("got nil bucket for %s", bucket)
		}

		c := b.Cursor()

		// iterate in reverse order (use First() and Next() for ascending order)
		for k, v := c.Last(); k != nil; k, v = c.Prev() {
			if dec, ok := decoders[bucket]; decode && ok {
				dec(k, v, writer)
			} else {
				defaultDecoder(k, v, writer)
			}

			limit--
			if limit == 0 {
				break
			}
		}

		return nil
	})
	return err
}
