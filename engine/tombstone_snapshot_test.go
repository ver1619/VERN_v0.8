package engine

import (
	"fmt"
	"testing"
	"time"
)

func TestTombstoneGC_SnapshotSafety(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir, &Config{
		MemtableSizeLimit:   1024 * 1024,
		L0CompactionTrigger: 2,
		L1MaxBytes:          1024 * 1024,
	})
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	key := []byte("key1")
	val := []byte("value1")

	// 1. Put K=V1 (Seq=1)
	if err := db.Put(key, val); err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	// 2. Snapshot S1 (Seq=1). Should see V1.
	s1 := db.GetSnapshot()
	defer db.ReleaseSnapshot(s1)

	// 3. Delete K (Tombstone, Seq=2)
	if err := db.Delete(key); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// 4. Snapshot S2 (Seq=2). Should see Not Found.
	s2 := db.GetSnapshot()
	defer db.ReleaseSnapshot(s2)

	// 5. Force Flush (Close/Reopen easiest to ensure data is on disk)
	// But we lose s1/s2 if we close DB.
	// We need to trigger flush without closing.
	// We can fill memtable.
	// But `MaybeScheduleFlush` is background.
	// We can manually call `CompactLevel` if we assume data is there?
	// But Delete is in Memtable.
	// We need to rotate memtable.
	// `db.freezeMemtable()` is private.
	// We can use reflection or just write enough data.
	// Let's write dummy data to fill memtable.

	// Write 1MB of dummy data
	dummyVal := make([]byte, 1024) // 1KB
	for i := 0; i < 1100; i++ {    // > 1MB
		db.Put([]byte(fmt.Sprintf("dummy%d", i)), dummyVal)
	}

	// Wait a bit for flush
	time.Sleep(500 * time.Millisecond)

	// Trigger Compaction on Level 0
	db.CompactLevel(0)

	// 6. Verify S1 sees V1
	v, err := db.GetWithOptions(key, &ReadOptions{Snapshot: s1})
	if err != nil {
		t.Fatalf("S1 should find key, got error: %v", err)
	}
	if string(v) != string(val) {
		t.Fatalf("S1 should see %s, got %s", val, v)
	}

	// 7. Verify S2 sees Not Found (proving T shows deleted)
	_, err = db.GetWithOptions(key, &ReadOptions{Snapshot: s2})
	if err != ErrNotFound {
		t.Fatalf("S2 should not find key, got: %v", err)
	}

	// If Tombstone was dropped, S2 might see V1 (Data Resurrection)
	// Because V1 (Seq=1) < S2 (Seq=2).
	// If T (Seq=2) is dropped, S2 reads V1.
	// The fact we see ErrNotFound implies T was KEPT (or V1 was also dropped).
	// But V1 must be kept for S1. So T must be kept for S2.
}
