package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManifestAppendAndDecode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "MANIFEST")

	m, err := OpenManifest(path)
	if err != nil {
		t.Fatal(err)
	}

	rec := Record{
		Type: RecordTypeSetWALCutoff,
		Data: SetWALCutoff{Seq: 123},
	}

	if err := m.Append(rec); err != nil {
		t.Fatal(err)
	}
	m.Close()

	raw, _ := os.ReadFile(path)
	r, _, err := DecodeRecord(raw)
	if err != nil {
		t.Fatal(err)
	}

	if r.Type != RecordTypeSetWALCutoff {
		t.Fatalf("wrong record type")
	}
	if r.Data.(SetWALCutoff).Seq != 123 {
		t.Fatalf("wrong cutoff seq")
	}
}

func TestManifestCorruption(t *testing.T) {
	rec := Record{
		Type: RecordTypeSetWALCutoff,
		Data: SetWALCutoff{Seq: 1},
	}

	raw, _ := EncodeRecord(rec)
	raw[len(raw)-1] ^= 0xFF

	if _, _, err := DecodeRecord(raw); err == nil {
		t.Fatalf("expected corruption detection")
	}
}
