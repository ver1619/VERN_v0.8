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
	defer db.Close()

	// Write some data and flush.
	db.Put([]byte("key1"), []byte("value1"))
	db.Put([]byte("key2"), []byte("value2"))
	db.freezeMemtable()

	// Verify SSTable creation.
	sstPath := filepath.Join(dir, fmt.Sprintf("%06d.sst", 1))
	if _, err := os.Stat(sstPath); os.IsNotExist(err) {
		t.Fatalf("SSTable %s not created", sstPath)
	}
}

func TestFlushRecovery(t *testing.T) {
	dir := t.TempDir()

	{
		db, _ := Open(dir)
		db.Put([]byte("a"), []byte("val1"))
		db.freezeMemtable()

		db.wal.Close()
		db.manifest.Close()
	}

	{
		db, err := Open(dir)
		if err != nil {
			t.Fatalf("Reopen failed: %v", err)
		}

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

		if db.nextFileNum <= 1 {
			t.Errorf("Expected nextFileNum > 1, got %d", db.nextFileNum)
		}
	}
}
