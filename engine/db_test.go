package engine

import (
	"testing"
)

func TestDBPutGetDelete(t *testing.T) {
	dir := t.TempDir()

	db, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Put
	if err := db.Put([]byte("a"), []byte("1")); err != nil {
		t.Fatal(err)
	}

	// Get
	val, err := db.Get([]byte("a"))
	if err != nil {
		t.Fatal(err)
	}
	if string(val) != "1" {
		t.Fatalf("unexpected value")
	}

	// Delete
	if err := db.Delete([]byte("a")); err != nil {
		t.Fatal(err)
	}

	// Get after delete
	_, err = db.Get([]byte("a"))
	if err != ErrNotFound {
		t.Fatalf("expected not found")
	}
}

func TestDBRecovery(t *testing.T) {
	dir := t.TempDir()

	{
		db, _ := Open(dir)
		db.Put([]byte("x"), []byte("y"))
	}

	db2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	val, err := db2.Get([]byte("x"))
	if err != nil {
		t.Fatal(err)
	}
	if string(val) != "y" {
		t.Fatalf("recovery failed")
	}
}

func TestGetWithSnapshot(t *testing.T) {
	dir := t.TempDir()

	db, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	db.Put([]byte("a"), []byte("v1"))

	snap := db.GetSnapshot()

	db.Put([]byte("a"), []byte("v2"))

	// Snapshot read must see old value
	val, err := db.GetWithOptions([]byte("a"), &ReadOptions{Snapshot: snap})
	if err != nil {
		t.Fatal(err)
	}
	if string(val) != "v1" {
		t.Fatalf("expected v1, got %s", val)
	}

	// Latest read must see new value
	val2, err := db.GetWithOptions([]byte("a"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(val2) != "v2" {
		t.Fatalf("expected v2, got %s", val2)
	}
}

func TestSnapshotDeleteVisibility(t *testing.T) {
	dir := t.TempDir()

	db, _ := Open(dir)
	db.Put([]byte("x"), []byte("1"))

	snap := db.GetSnapshot()

	db.Delete([]byte("x"))

	// Snapshot must still see value
	val, err := db.GetWithOptions([]byte("x"), &ReadOptions{Snapshot: snap})
	if err != nil {
		t.Fatal(err)
	}
	if string(val) != "1" {
		t.Fatalf("expected value before delete")
	}

	// Latest read must not see it
	_, err = db.GetWithOptions([]byte("x"), nil)
	if err != ErrNotFound {
		t.Fatalf("expected not found after delete")
	}
}

func TestImmutableMemtableReadVisibility(t *testing.T) {
	dir := t.TempDir()
	db, _ := Open(dir)

	db.Put([]byte("a"), []byte("1"))

	// Freeze active memtable
	db.freezeMemtable()

	// New writes go to new memtable
	db.Put([]byte("b"), []byte("2"))

	v1, err := db.Get([]byte("a"))
	if err != nil || string(v1) != "1" {
		t.Fatalf("expected to read from immutable memtable")
	}

	v2, err := db.Get([]byte("b"))
	if err != nil || string(v2) != "2" {
		t.Fatalf("expected to read from active memtable")
	}
}
