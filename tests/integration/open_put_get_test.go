package integration

import (
	"testing"

	"vern_kv0.5/engine"
)

func TestOpenPutGet(t *testing.T) {
	dir := t.TempDir()

	db, err := engine.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	if err := db.Put([]byte("k1"), []byte("v1")); err != nil {
		t.Fatal(err)
	}

	val, err := db.Get([]byte("k1"))
	if err != nil {
		t.Fatal(err)
	}
	if string(val) != "v1" {
		t.Fatalf("unexpected value")
	}

	// Restart
	db2, err := engine.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	val2, err := db2.Get([]byte("k1"))
	if err != nil {
		t.Fatal(err)
	}
	if string(val2) != "v1" {
		t.Fatalf("value lost after restart")
	}
}
