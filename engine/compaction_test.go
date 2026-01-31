package engine

import (
	"fmt"
	"testing"
)

func TestCompactionL0toL1(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	// Create 5 SSTables in L0
	// By flushing 5 memtables
	for i := 0; i < 5; i++ {
		key := []byte(fmt.Sprintf("key%d", i))
		val := []byte(fmt.Sprintf("val%d-v1", i))
		db.Put(key, val)

		// Add an overwrite for key0 in the last file to check precedence
		if i == 4 {
			db.Put([]byte("key0"), []byte("val0-updated"))
		}

		db.freezeMemtable() // Force flush -> new L0 file
	}

	// Check L0/L1 state
	// Threshold = 4 (len >= 4).
	// 4th flush triggers compaction. L0 becomes 0, L1 becomes 1.
	// 5th flush adds one more file to L0.
	// So L0 should be 1 and L1 should be 1.
	if len(db.version.Levels[0]) != 1 {
		t.Errorf("Expected 1 L0 file (after auto-compaction + 1 new), got %d", len(db.version.Levels[0]))
	}
	if len(db.version.Levels[1]) != 1 {
		t.Errorf("Expected 1 L1 file, got %d", len(db.version.Levels[1]))
	}

	// Data should still be valid
	v, err := db.Get([]byte("key0"))
	if err != nil {
		t.Errorf("Get key0 failed: %v", err)
	} else if string(v) != "val0-updated" {
		t.Errorf("key0: want val0-updated, got %s", string(v))
	}

	// key1 should be present (val1-v1)
	v, err = db.Get([]byte("key1"))
	if err != nil {
		t.Errorf("Get key1 failed: %v", err)
	} else if string(v) != "val1-v1" {
		t.Errorf("key1: want val1-v1, got %s", string(v))
	}
}
