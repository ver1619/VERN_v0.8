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
