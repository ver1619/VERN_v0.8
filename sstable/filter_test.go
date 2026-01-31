package sstable

import (
	"path/filepath"
	"testing"
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

	// Check missing keys (probability of false positive is low but non-zero)
	missing := [][]byte{
		[]byte("missing"),
		[]byte("baz"),
		[]byte("12345"),
	}

	for _, k := range missing {
		// We can't strict assert false, but mostly likely it is false.
		// If it's true, it's a false positive.
		// With 4 keys and 10 bits/key, FP rate is ~1%.
		// So assume false for test, but be aware.
		if filter.KeyMayMatch(k, data) {
			t.Logf("False positive for key %s (expected)", k)
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
		b.Add([]byte(k), []byte("val"))
	}

	if err := b.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	r, err := NewReader(file)
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
			t.Errorf("MayContain(%s) = false, want true", k)
		}
	}

	// Test non-matches
	if r.MayContain([]byte("missing_key")) {
		// FP possible but unlikely for this small set
		t.Log("Potential false positive for 'missing_key'")
	}
}
