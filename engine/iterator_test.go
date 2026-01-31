package engine

import "testing"

func TestSnapshotIteratorStability(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	db.Put([]byte("a"), []byte("1"))
	db.Put([]byte("b"), []byte("2"))

	snap := db.GetSnapshot()

	db.Put([]byte("c"), []byte("3"))

	it := db.NewIterator(&ReadOptions{Snapshot: snap})
	it.SeekToFirst()

	var keys []string
	for it.Valid() {
		keys = append(keys, string(it.Key()))
		it.Next()
	}

	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
	if keys[0] != "a" || keys[1] != "b" {
		t.Fatalf("unexpected keys: %v", keys)
	}
}

func TestIteratorDeleteVisibility(t *testing.T) {
	dir := t.TempDir()
	db, _ := Open(dir)

	db.Put([]byte("x"), []byte("1"))
	snap := db.GetSnapshot()
	db.Delete([]byte("x"))

	it := db.NewIterator(&ReadOptions{Snapshot: snap})
	it.SeekToFirst()

	if !it.Valid() {
		t.Fatalf("expected x to be visible in snapshot")
	}

	if string(it.Value()) != "1" {
		t.Fatalf("unexpected value")
	}
}
