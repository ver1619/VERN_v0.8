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

	db.Put([]byte("key1"), []byte("val1"))
	db.freezeMemtable()

	db.Put([]byte("key1"), []byte("val1-updated"))
	db.Put([]byte("key2"), []byte("val2"))
	db.freezeMemtable()

	db.Put([]byte("key3"), []byte("val3"))

	// Verify reads from SSTables and Memtable.
	v, err := db.Get([]byte("key1"))
	if err != nil || string(v) != "val1-updated" {
		t.Errorf("key1: want val1-updated, got %s (err: %v)", string(v), err)
	}

	v, err = db.Get([]byte("key2"))
	if err != nil || string(v) != "val2" {
		t.Errorf("key2: want val2, got %s (err: %v)", string(v), err)
	}

	v, err = db.Get([]byte("key3"))
	if err != nil || string(v) != "val3" {
		t.Errorf("key3: want val3, got %s (err: %v)", string(v), err)
	}

	db.Close()

	// Verify recovery.
	db2, err := Open(dir)
	if err != nil {
		t.Fatalf("Reopen: %v", err)
	}
	defer db2.Close()

	v, err = db2.Get([]byte("key1"))
	if err != nil || string(v) != "val1-updated" {
		t.Errorf("Recovered key1: want val1-updated, got %s (err: %v)", string(v), err)
	}

	v, err = db2.Get([]byte("key3"))
	if err != nil || string(v) != "val3" {
		t.Errorf("Recovered key3: want val3, got %s (err: %v)", string(v), err)
	}
}
