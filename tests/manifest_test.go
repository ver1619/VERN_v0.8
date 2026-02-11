package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"vern_kv0.8/engine"
)

func TestManifestCompaction(t *testing.T) {
	dir, err := os.MkdirTemp("", "vern-manifest-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	opts := engine.DefaultConfig()
	// Set small limits to trigger flushing.
	opts.MemtableSizeLimit = 1024

	db, err := engine.Open(dir, opts)
	if err != nil {
		t.Fatal(err)
	}

	// Create a lot of manifest entries by flushing many small tables
	for i := 0; i < 5; i++ {
		key := []byte(fmt.Sprintf("key-%04d", i))
		value := []byte(fmt.Sprintf("val-%04d", i))
		if err := db.Put(key, value); err != nil {
			t.Fatal(err)
		}
	}

	for i := 0; i < 10; i++ {
		key := []byte(fmt.Sprintf("key-%04d", i))
		val := make([]byte, 100)
		if err := db.Put(key, val); err != nil {
			t.Fatal(err)
		}
	}

	// Check manifest size
	manifestPath := filepath.Join(dir, "MANIFEST")
	info, err := os.Stat(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	initialSize := info.Size()

	// Compact Manifest
	if err := db.CompactManifest(); err != nil {
		t.Fatal(err)
	}

	// Check size again
	info, err = os.Stat(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	finalSize := info.Size()

	t.Logf("Initial Manifest Size: %d, Final: %d", initialSize, finalSize)

	db.Close()

	// Reopen verify
	db2, err := engine.Open(dir, opts)
	if err != nil {
		t.Fatal(err)
	}

	val, err := db2.Get([]byte("key-0005"))
	if err != nil {
		t.Fatal(err)
	}
	if len(val) != 100 {
		t.Fatalf("expected value length 100, got %d", len(val))
	}

	db2.Close()
}
