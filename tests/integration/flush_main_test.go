package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"vern_kv0.8/engine"
)

func TestAutoFlushKeyTrigger(t *testing.T) {
	dir := t.TempDir()
	db, err := engine.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Memtable limit is 4MB. Write ~4.5MB.
	// 1KB per entry -> 4500 entries.
	val := make([]byte, 1000)
	for i := 0; i < 4500; i++ {
		key := []byte(fmt.Sprintf("k%09d", i))
		if err := db.Put(key, val); err != nil {
			t.Fatal(err)
		}
	}

	// Give async flush time to run
	time.Sleep(2 * time.Second)

	// Check for SST files
	files, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}

	sstCount := 0
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".sst" {
			sstCount++
		}
	}

	if sstCount == 0 {
		t.Fatalf("Expected auto-flush to create SST files, found 0. Memory growth fix failed.")
	}
}
