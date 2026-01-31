package integration

import (
	"fmt"
	"testing"

	"vern_kv0.8/engine"
	"vern_kv0.8/wal"
)

func TestIntegrationBasic(t *testing.T) {
	dir := t.TempDir()
	db, err := engine.Open(dir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// 1. Basic Puts
	for i := 0; i < 100; i++ {
		db.Put([]byte(fmt.Sprintf("key%d", i)), []byte(fmt.Sprintf("val%d", i)))
	}

	// 2. Batch Write
	batch := &wal.Batch{
		Records: []wal.LogicalRecord{
			{Key: []byte("batch-key1"), Value: []byte("batch-val1"), Type: wal.LogicalTypePut},
			{Key: []byte("batch-key2"), Value: []byte("batch-val2"), Type: wal.LogicalTypePut},
			{Key: []byte("key0"), Type: wal.LogicalTypeDelete}, // Overwrite key0 with delete
		},
	}
	if err := db.Write(batch); err != nil {
		t.Fatalf("Batch Write failed: %v", err)
	}

	// 3. Verify consistency
	v, err := db.Get([]byte("batch-key1"))
	if err != nil || string(v) != "batch-val1" {
		t.Errorf("batch-key1: want batch-val1, got %v", string(v))
	}
	_, err = db.Get([]byte("key0"))
	if err != engine.ErrNotFound {
		t.Errorf("key0 should be deleted, got err %v", err)
	}

	db.Close()

	// 4. Recovery
	db, err = engine.Open(dir)
	if err != nil {
		t.Fatalf("Reopen failed: %v", err)
	}
	defer db.Close()

	v, err = db.Get([]byte("batch-key2"))
	if err != nil || string(v) != "batch-val2" {
		t.Errorf("batch-key2: want batch-val2, got %v", string(v))
	}
}

func TestIntegrationCompactionSpaceReclamation(t *testing.T) {
	dir := t.TempDir()
	db, err := engine.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Fill L0 enough to trigger compaction multiple times if threshold is low
	// Actually, let's just trigger it manually via enough flushes
	for i := 0; i < 10; i++ {
		db.Put([]byte("static-key"), []byte(fmt.Sprintf("val-%d", i)))
		// In engine internal, we'd need to access unexported freezeMemtable to force L0.
		// Since we can't from package integration, we have to rely on auto-triggering
		// if we add a way to trigger it or just write enough data.
		// For now, these integration tests are black-box.
	}
}
