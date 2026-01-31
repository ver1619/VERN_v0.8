package sstable

import (
	"os"
	"path/filepath"
	"testing"

	"vern_kv0.5/internal"
)

func TestSSTableIterator(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.sst")

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}

	WriteRecord(f,
		internal.EncodeInternalKey([]byte("a"), 1, internal.RecordTypeValue),
		[]byte("1"),
	)
	WriteRecord(f,
		internal.EncodeInternalKey([]byte("b"), 1, internal.RecordTypeValue),
		[]byte("2"),
	)
	f.Close()

	it, err := NewIterator(path)
	if err != nil {
		t.Fatal(err)
	}

	it.SeekToFirst()
	if !it.Valid() {
		t.Fatalf("iterator should be valid")
	}

	if string(it.Value()) != "1" {
		t.Fatalf("unexpected value")
	}
}
