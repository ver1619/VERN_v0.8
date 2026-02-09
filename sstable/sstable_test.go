package sstable

import (
	"path/filepath"
	"testing"

	"vern_kv0.8/internal"
)

func TestSSTableIterator(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.sst")

	// Build SSTable
	b, err := NewBuilder(path)
	if err != nil {
		t.Fatal(err)
	}

	b.Add(internal.EncodeInternalKey([]byte("a"), 1, internal.RecordTypeValue), []byte("1"))
	b.Add(internal.EncodeInternalKey([]byte("b"), 1, internal.RecordTypeValue), []byte("2"))

	if err := b.Close(); err != nil {
		t.Fatal(err)
	}

	it, err := NewIterator(path, nil)
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
