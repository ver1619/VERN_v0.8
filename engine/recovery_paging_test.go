package engine

import (
	"path/filepath"
	"testing"

	"vern_kv0.8/manifest"
	"vern_kv0.8/wal"
)

func TestRecoveryPaging(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "MANIFEST")
	walDir := filepath.Join(dir, "wal")

	// Create manifest
	f, _ := manifest.OpenManifest(manifestPath)
	f.Append(manifest.Record{
		Type: manifest.RecordTypeSetWALCutoff,
		Data: manifest.SetWALCutoff{Seq: 0},
	})
	f.Close()

	// Write large WAL (relative to small paging limit)
	w, err := wal.OpenWAL(walDir, 1024*1024)
	if err != nil {
		t.Fatal(err)
	}

	// 10 entries of 100 bytes ~ 1KB
	// Limit will be 200 bytes => flush every 2-3 entries
	for i := 0; i < 10; i++ {
		key := []byte{byte(i)}
		val := make([]byte, 100)
		w.Append(wal.Batch{
			SeqStart: uint64(i + 1),
			Records:  []wal.LogicalRecord{{Key: key, Value: val, Type: wal.LogicalTypePut}},
		})
	}
	w.Sync()
	w.Close()

	// Recover with 200 byte limit
	state, err := Recover(dir, walDir, 200)
	if err != nil {
		t.Fatalf("Recover: %v", err)
	}

	// Should have flushed at least once
	files := state.VersionSet.GetAllTables()
	if len(files) == 0 {
		t.Errorf("Expected flushed tables, got 0")
	}

	// Check final memtable
	if state.Memtable.Size() == 0 {
		// If exact flush, memtable might be empty or small.
	}

	t.Logf("Recovered %d L0 files", len(files))
}
