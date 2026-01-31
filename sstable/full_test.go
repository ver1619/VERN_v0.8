package sstable

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestFullSSTable(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.sst")

	// Build SSTable
	b, err := NewBuilder(file)
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}

	// Write enough data to create multiple blocks (blockSize=4KB)
	const numKeys = 1000
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("k%04d", i)
		value := fmt.Sprintf("v%04d", i)
		// Approx 10 bytes key + 10 bytes value + overhead ~ 25 bytes
		// 1000 * 25 = 25KB, should generate ~6-7 blocks
		if err := b.Add([]byte(key), []byte(value)); err != nil {
			t.Fatalf("Add: %v", err)
		}
	}

	if err := b.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Read SSTable
	r, err := NewReader(file)
	if err != nil {
		t.Fatalf("NewReader: %v", err)
	}
	defer r.Close()

	it, err := r.NewIterator()
	if err != nil {
		t.Fatalf("NewIterator: %v", err)
	}

	// Verify Scan
	it.SeekToFirst()
	count := 0
	for it.Valid() {
		key := string(it.Key())
		want := fmt.Sprintf("k%04d", count)
		if key != want {
			t.Errorf("Scan mismatch at %d: want %s, got %s", count, want, key)
		}
		count++
		it.Next()
	}

	if count != numKeys {
		t.Errorf("Expected %d keys, got %d", numKeys, count)
	}

	// Verify Seek
	seekTests := []struct {
		target string
		want   string
	}{
		{"k0000", "k0000"},
		{"k0500", "k0500"},
		{"k0999", "k0999"},
		{"k0100", "k0100"},
		{"k0100a", "k0101"},
	}

	for _, tt := range seekTests {
		it.Seek([]byte(tt.target))
		if !it.Valid() {
			t.Errorf("Seek(%s) invalid", tt.target)
			continue
		}
		if string(it.Key()) != tt.want {
			t.Errorf("Seek(%s) want %s, got %s", tt.target, tt.want, it.Key())
		}
	}
}
