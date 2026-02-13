package sstable

import (
	"path/filepath"
	"testing"

	"vern_kv0.8/internal"
)

func TestBloomFilterLogic(t *testing.T) {
	filter := NewBloomFilter(10)
	keys := [][]byte{
		[]byte("hello"),
		[]byte("world"),
		[]byte("foo"),
		[]byte("bar"),
	}

	data := filter.CreateFilter(keys)

	// Check present keys
	for _, k := range keys {
		if !filter.KeyMayMatch(k, data) {
			t.Errorf("Key %s should be present", k)
		}
	}

	// Check missing keys.
	missing := [][]byte{
		[]byte("missing"),
		[]byte("baz"),
		[]byte("12345"),
	}

	for _, k := range missing {
		if filter.KeyMayMatch(k, data) {
			t.Logf("False positive for key %s", k)
		}
	}
}

func TestFilterIntegration(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "filter_test.sst")

	b, err := NewBuilder(file)
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}

	keys := []string{"key1", "key2", "key3", "ultra_long_key_name"}
	for _, k := range keys {
		// Builder expects Internal Keys to extract User Key for filter.
		ikey := internal.EncodeInternalKey([]byte(k), 100, internal.RecordTypeValue)
		b.Add(ikey, []byte("val"))
	}

	if err := b.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	r, err := NewReader(file, nil)
	if err != nil {
		t.Fatalf("NewReader: %v", err)
	}
	defer r.Close()

	if r.filterData == nil {
		t.Fatal("Expected filter data to be loaded")
	}

	// Test matches
	for _, k := range keys {
		if !r.MayContain([]byte(k)) {
			// Reader.MayContain expects User Key.
			t.Errorf("MayContain(%s) = false, want true", k)
		}
	}

	// Test non-matches
	if r.MayContain([]byte("missing_key")) {
		t.Log("False positive for 'missing_key'")
	}
}
