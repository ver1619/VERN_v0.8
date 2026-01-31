package engine

import "testing"

func TestSnapshotCapturesSequence(t *testing.T) {
	dir := t.TempDir()

	db, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	db.Put([]byte("a"), []byte("1"))
	db.Put([]byte("b"), []byte("2"))

	snap := db.GetSnapshot()
	if snap == nil {
		t.Fatalf("snapshot should not be nil")
	}

	expected := db.nextSeq - 1
	if snap.ReadSeq != expected {
		t.Fatalf("expected snapshot seq %d, got %d", expected, snap.ReadSeq)
	}
}

func TestSnapshotIsImmutable(t *testing.T) {
	dir := t.TempDir()

	db, _ := Open(dir)
	db.Put([]byte("x"), []byte("1"))

	snap := db.GetSnapshot()
	oldSeq := snap.ReadSeq

	db.Put([]byte("y"), []byte("2"))

	if snap.ReadSeq != oldSeq {
		t.Fatalf("snapshot must be immutable")
	}
}
