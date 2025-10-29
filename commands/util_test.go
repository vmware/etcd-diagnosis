// Copyright (c) 2025 Broadcom. All Rights Reserved.
// Broadcom Confidential. The term "Broadcom" refers to Broadcom Inc.
// and/or its subsidiaries.

package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	bolt "go.etcd.io/bbolt"
)

const (
	// TestFreelistType is used as an env variable for test to indicate the backend type.
	TestFreelistType = "TEST_FREELIST_TYPE"
)

// DB is a test wrapper for bolt.DB.
type DB struct {
	*bolt.DB
	f  string
	o  *bolt.Options
	tb testing.TB
}

// MustCreateDB returns a new, open DB at a temporary location.
func MustCreateDB(tb testing.TB) *DB {
	return MustCreateDBWithOption(tb, nil)
}

// MustCreateDBWithOption returns a new, open DB at a temporary location with given options.
func MustCreateDBWithOption(tb testing.TB, o *bolt.Options) *DB {
	f := filepath.Join(tb.TempDir(), "db")
	return MustOpenDBWithOption(tb, f, o)
}

func MustOpenDBWithOption(tb testing.TB, f string, o *bolt.Options) *DB {
	db, err := OpenDBWithOption(tb, f, o)
	require.NoError(tb, err)
	require.NotNil(tb, db)
	return db
}

func OpenDBWithOption(tb testing.TB, f string, o *bolt.Options) (*DB, error) {
	tb.Logf("Opening bbolt DB at: %s", f)
	if o == nil {
		o = bolt.DefaultOptions
	}

	freelistType := bolt.FreelistArrayType
	if env := os.Getenv(TestFreelistType); env == string(bolt.FreelistMapType) {
		freelistType = bolt.FreelistMapType
	}

	o.FreelistType = freelistType

	db, err := bolt.Open(f, 0o600, o)
	if err != nil {
		return nil, err
	}
	resDB := &DB{
		DB: db,
		f:  f,
		o:  o,
		tb: tb,
	}

	tb.Cleanup(resDB.PostTestCleanup)
	return resDB, nil
}

func (db *DB) PostTestCleanup() {
	if db.DB != nil {
		db.MustClose()
	}
}

// Close closes the database but does NOT delete the underlying file.
func (db *DB) Close() error {
	if db.DB != nil {
		db.tb.Logf("Closing bbolt DB at: %s", db.f)
		err := db.DB.Close()
		if err != nil {
			return err
		}
		db.DB = nil
	}
	return nil
}

// MustClose closes the database but does NOT delete the underlying file.
func (db *DB) MustClose() {
	err := db.Close()
	require.NoError(db.tb, err)
}

func (db *DB) SetOptions(o *bolt.Options) {
	db.o = o
}

// MustReopen reopen the database. Panic on error.
func (db *DB) MustReopen() {
	if db.DB != nil {
		panic("Please call Close() before MustReopen()")
	}
	db.tb.Logf("Reopening bbolt DB at: %s", db.f)
	indb, err := bolt.Open(db.Path(), 0o600, db.o)
	require.NoError(db.tb, err)
	db.DB = indb
}

func (db *DB) Path() string {
	return db.f
}
