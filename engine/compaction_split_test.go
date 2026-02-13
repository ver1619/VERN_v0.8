package engine

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

func TestCompactionSplit(t *testing.T) {
	dir := t.TempDir()
	opts := DefaultConfig()
	opts.L0CompactionTrigger = 100           // Prevent auto-compaction
	opts.MemtableSizeLimit = 4 * 1024 * 1024 // 4MB flush threshold
	opts.SyncWrites = false                  // Speed up test

	db, err := Open(dir, opts)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	// Generate 30MB of incompressible data
	totalBytes := 30 * 1024 * 1024
	valSize := 1024

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	entries := totalBytes / valSize
	for i := 0; i < entries; i++ {
		key := []byte(fmt.Sprintf("key-%012d", i))
		val := make([]byte, valSize)
		rng.Read(val) // Randomize content to defeat compression
		if err := db.Put(key, val); err != nil {
			t.Fatalf("Put: %v", err)
		}
	}

	// Force final flush
	db.freezeMemtable()

	// Wait explicitly for flushes to complete
	expectedL0 := 6
	var l0Count int

	for i := 0; i < 100; i++ {
		db.mu.RLock()
		l0Count = len(db.version.Levels[0])
		db.mu.RUnlock()
		if l0Count >= expectedL0 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if l0Count < expectedL0 {
		t.Logf("Warning: L0 count %d is less than expected %d, proceeding anyway", l0Count, expectedL0)
	}

	// Now compact
	t.Log("Starting manual compaction of L0...")
	if err := db.CompactLevel(0); err != nil {
		t.Fatalf("CompactLevel(0): %v", err)
	}

	// Verify L1
	db.mu.RLock()
	l0Count = len(db.version.Levels[0])
	l1Count := len(db.version.Levels[1])
	files := db.version.Levels[1]
	db.mu.RUnlock()

	t.Logf("Compaction complete. L0=%d, L1=%d", l0Count, l1Count)

	if l0Count != 0 {
		t.Errorf("Expected L0 to be empty, got %d", l0Count)
	}

	if l1Count < 2 {
		t.Errorf("Expected L1 to have at least 2 files (split), got %d", l1Count)
	}

	// Verify total size roughly matches
	var totalL1Size int64
	for _, f := range files {
		totalL1Size += f.FileSize
		t.Logf("L1 File: Num=%d, Size=%d", f.FileNum, f.FileSize)
	}

	// Should be around 25MB (plus overheads)
	if totalL1Size < 20*1024*1024 {
		t.Errorf("L1 size too small: %d", totalL1Size)
	}
}
