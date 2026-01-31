package engine

import (
	"testing"
)

func TestFullCycleFlushAndRead(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	// 1. Write Memtable 1
	db.Put([]byte("key1"), []byte("val1"))
	db.freezeMemtable() // Flush to SST 1

	// 2. Write Memtable 2
	db.Put([]byte("key1"), []byte("val1-updated")) // Overwrite
	db.Put([]byte("key2"), []byte("val2"))
	db.freezeMemtable() // Flush to SST 2

	// 3. Write Active Memtable (not flushed)
	db.Put([]byte("key3"), []byte("val3"))

	// 4. Read check
	// key1 should be val1-updated (from SST, newer one)
	v, err := db.Get([]byte("key1"))
	if err != nil {
		t.Fatalf("Get key1: %v", err)
	}
	if string(v) != "val1-updated" {
		t.Errorf("key1: want val1-updated, got %s", string(v))
	}

	// key2 should be val2 (from SST)
	v, err = db.Get([]byte("key2"))
	if err != nil {
		t.Fatalf("Get key2: %v", err)
	}
	if string(v) != "val2" {
		t.Errorf("key2: want val2, got %s", string(v))
	}

	// key3 should be val3 (from Memtable)
	v, err = db.Get([]byte("key3"))
	if err != nil {
		t.Fatalf("Get key3: %v", err)
	}
	if string(v) != "val3" {
		t.Errorf("key3: want val3, got %s", string(v))
	}

	db.Close()

	// 5. Recover and Read
	db2, err := Open(dir)
	if err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	defer db2.Close()

	v, err = db2.Get([]byte("key1"))
	if err != nil {
		t.Errorf("Recovered Get key1 error: %v", err)
	} else if string(v) != "val1-updated" {
		t.Errorf("Recovered key1: want val1-updated, got %s", string(v))
	}

	v, err = db2.Get([]byte("key3"))
	if err != nil {
		// key3 was in memtable, so should have been recovered from WAL
		t.Errorf("Recovered Get key3 error: %v", err)
	} else if string(v) != "val3" {
		t.Errorf("Recovered key3: want val3, got %s", string(v))
	}
}
