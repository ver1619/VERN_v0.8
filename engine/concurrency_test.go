package engine

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestConcurrencySWMR(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	var wg sync.WaitGroup

	// Writer
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			key := []byte(fmt.Sprintf("key-%d", i))
			val := []byte(fmt.Sprintf("val-%d", i))
			if err := db.Put(key, val); err != nil {
				t.Errorf("Put failed: %v", err)
			}

			// Occasionally flush manually to force rotation?
			// But Put triggers flush if memtable full (not implemented yet, we rely on freezeMemtable called manually or by size).
			// v0.8 memtable size trigger is not in `Put` yet (Wait, did I add it? No, plan said Manual/Auto Trigger mostly Manual in tests).
			// If `Put` doesn't auto-flush, memtable grows indefinitely.
			// Let's add explicit flush trigger every 100 writes.
			if i%100 == 0 {
				db.freezeMemtable()
			}
		}
	}()

	// Readers
	for r := 0; r < 5; r++ {
		wg.Add(1)
		go func(rid int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				// Random delay
				time.Sleep(1 * time.Millisecond)

				// Read a key that might exist
				// concurrency is nondeterministic, so we just check for errors/panics
				// or consistency if we know what to expect.
				// Here we just ensure NO RACES (go test -race).

				// We can try to read keys written by writer.
				// Snapshots
				snap := db.GetSnapshot()
				iter := db.NewIterator(&ReadOptions{Snapshot: snap})
				iter.SeekToFirst()
				count := 0
				for iter.Valid() {
					count++
					iter.Next()
				}
				// t.Logf("Reader %d saw %d items", rid, count)
			}
		}(r)
	}

	wg.Wait()
}
