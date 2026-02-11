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

	// Writer routine.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			key := []byte(fmt.Sprintf("key-%d", i))
			val := []byte(fmt.Sprintf("val-%d", i))
			if err := db.Put(key, val); err != nil {
				t.Errorf("Put failed: %v", err)
			}

			// Flush memtable periodically.
			if i%100 == 0 {
				db.freezeMemtable()
			}
		}
	}()

	// Reader routines.
	for r := 0; r < 5; r++ {
		wg.Add(1)
		go func(rid int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				// Random delay
				time.Sleep(1 * time.Millisecond)

				// Verify concurrent snapshot iteration.
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
