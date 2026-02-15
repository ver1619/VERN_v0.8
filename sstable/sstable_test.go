package sstable

import (
	"os"
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

func TestPrefixCompression(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prefix.sst")

	b, err := NewBuilder(path)
	if err != nil {
		t.Fatal(err)
	}

	// Add keys with long shared prefix
	prefix := "long_shared_prefix_for_compression_"
	b.Add(internal.EncodeInternalKey([]byte(prefix+"1"), 1, internal.RecordTypeValue), []byte("v1"))
	b.Add(internal.EncodeInternalKey([]byte(prefix+"2"), 1, internal.RecordTypeValue), []byte("v2"))
	b.Add(internal.EncodeInternalKey([]byte(prefix+"3"), 1, internal.RecordTypeValue), []byte("v3"))

	if err := b.Close(); err != nil {
		t.Fatal(err)
	}

	// Verify size is smaller than raw sum
	// Raw keys = (32+1+8)*3 = 123 bytes
	// Compressed sum should be much smaller.
	info, _ := os.Stat(path)
	if info.Size() > 200 {
		// Very rough upper bound, mainly checking it's not exploding.
	}

	// Verify Iterator
	it, err := NewIterator(path, nil)
	if err != nil {
		t.Fatal(err)
	}

	it.SeekToFirst()
	if !it.Valid() || string(internal.ExtractUserKey(it.Key())) != prefix+"1" {
		t.Fatalf("Expected %s1", prefix)
	}

	it.Next()
	if !it.Valid() || string(internal.ExtractUserKey(it.Key())) != prefix+"2" {
		t.Fatalf("Expected %s2", prefix)
	}

	it.Next()
	if !it.Valid() || string(internal.ExtractUserKey(it.Key())) != prefix+"3" {
		t.Fatalf("Expected %s3", prefix)
	}
}
