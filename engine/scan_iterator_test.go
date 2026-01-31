package engine

import "testing"

func TestRangeScanSnapshot(t *testing.T) {
	dir := t.TempDir()
	db, _ := Open(dir)

	db.Put([]byte("a"), []byte("1"))
	db.Put([]byte("b"), []byte("2"))
	db.Put([]byte("c"), []byte("3"))

	snap := db.GetSnapshot()

	db.Put([]byte("d"), []byte("4"))

	it := db.NewRangeIterator(
		[]byte("a"),
		[]byte("d"),
		&ReadOptions{Snapshot: snap},
	)

	it.SeekToFirst()

	var keys []string
	for it.Valid() {
		keys = append(keys, string(it.Key()))
		it.Next()
	}

	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %v", keys)
	}
}

func TestPrefixScan(t *testing.T) {
	dir := t.TempDir()
	db, _ := Open(dir)

	db.Put([]byte("user:1"), []byte("a"))
	db.Put([]byte("user:2"), []byte("b"))
	db.Put([]byte("sys:1"), []byte("x"))

	it := db.NewPrefixIterator(
		[]byte("user:"),
		nil,
	)

	it.SeekToFirst()

	var keys []string
	for it.Valid() {
		keys = append(keys, string(it.Key()))
		it.Next()
	}

	if len(keys) != 2 {
		t.Fatalf("unexpected keys: %v", keys)
	}
}
