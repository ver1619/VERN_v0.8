package engine

import (
	"fmt"
	"testing"
	"time"
)

func TestCompactionL0toL1(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	// Create 5 L0 SSTables by flushing memtables.
	for i := 0; i < 5; i++ {
		key := []byte(fmt.Sprintf("key%d", i))
		val := []byte(fmt.Sprintf("val%d-v1", i))
		db.Put(key, val)

		// Overwrite key0 to verify version precedence.
		if i == 4 {
			db.Put([]byte("key0"), []byte("val0-updated"))
		}

		db.freezeMemtable()
	}

	// Verify background compaction reduces L0 and populates L1.
	var l0, l1 int
	for i := 0; i < 50; i++ {
		db.mu.RLock()
		l0 = len(db.version.Levels[0])
		l1 = len(db.version.Levels[1])
		db.mu.RUnlock()
		if l0 == 1 && l1 == 1 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if l0 != 1 || l1 != 1 {
		t.Errorf("Timeout waiting for compaction: L0=%d, L1=%d (want 1, 1)", l0, l1)
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
