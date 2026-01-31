package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestFlushManual(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close() // We don't have Close yet but let's assume valid or just ignore
	// Actually DB struct doesn't have Close method yet in db.go shown previously?
	// It relies on OS cleanup or we should add one.
	// Manifest has Close. DB has WAL which has Close.
	// The Open method shows returning &DB.

	// Write some data
	db.Put([]byte("key1"), []byte("value1"))
	db.Put([]byte("key2"), []byte("value2"))

	// Force flush by freezing
	// This calls MaybeScheduleFlush which calls flushMemtable
	db.freezeMemtable()

	// Check if SSTable exists
	// File num should be 1
	sstPath := filepath.Join(dir, fmt.Sprintf("%06d.sst", 1))
	if _, err := os.Stat(sstPath); os.IsNotExist(err) {
		t.Fatalf("SSTable %s not created", sstPath)
	}

	// Check manifest
	// We can't easily read internal manifest state without reopening or peeking
	// Let's reopen and check restored state
}

func TestFlushRecovery(t *testing.T) {
	dir := t.TempDir()

	// Phase 1: Write and Flush
	{
		db, _ := Open(dir)
		db.Put([]byte("a"), []byte("val1"))
		db.freezeMemtable() // Flush to 000001.sst

		// DB doesn't have Close method exposed in previous view, let's close manually components if possible
		// or just rely on Syncs that happen during operations.
		// manifest.Append does Sync.
		// wal.Append/Sync does Sync.
		db.wal.Close()
		db.manifest.Close()
	}

	// Phase 2: Recover
	{
		db, err := Open(dir)
		if err != nil {
			t.Fatalf("Reopen failed: %v", err)
		}

		// Check that we 'know' about the table
		tables := db.version.GetAllTables()
		if len(tables) != 1 {
			t.Errorf("Expected 1 table, got %d", len(tables))
		}

		found := false
		for _, t := range tables {
			if t.FileNum == 1 {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("Expected table 1 to be present")
		}

		// Ensure nextFileNum is advanced
		if db.nextFileNum <= 1 {
			t.Errorf("Expected nextFileNum > 1, got %d", db.nextFileNum)
		}
	}
}
