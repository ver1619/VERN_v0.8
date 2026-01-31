package engine

import (
	"path/filepath"
	"testing"

	"vern_kv0.5/manifest"
	"vern_kv0.5/wal"
)

func TestFullRecovery(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "MANIFEST")
	walDir := filepath.Join(dir, "wal")

	// ---- Write manifest ----
	m, _ := manifest.OpenManifest(manifestPath)
	m.Append(manifest.Record{
		Type: manifest.RecordTypeSetWALCutoff,
		Data: manifest.SetWALCutoff{Seq: 0},
	})
	m.Close()

	// ---- Write WAL ----
	w, _ := wal.OpenWAL(walDir, 1024)
	w.Append(wal.Batch{
		SeqStart: 1,
		Records: []wal.LogicalRecord{
			{Key: []byte("a"), Value: []byte("1"), Type: wal.LogicalTypePut},
			{Key: []byte("b"), Value: []byte("2"), Type: wal.LogicalTypePut},
		},
	})
	w.Sync()
	w.Close()

	// ---- Recover ----
	state, err := Recover(manifestPath, walDir)
	if err != nil {
		t.Fatal(err)
	}

	if state.NextSeq != 3 {
		t.Fatalf("expected next seq 3, got %d", state.NextSeq)
	}

	if state.Memtable.Size() != 2 {
		t.Fatalf("expected 2 entries in memtable")
	}
}
