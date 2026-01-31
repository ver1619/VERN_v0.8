package determinism

import (
	"testing"

	"vern_kv0.5/engine"
)

func TestReplayRepeatability(t *testing.T) {
	dir := t.TempDir()

	db, err := engine.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	db.Put([]byte("a"), []byte("1"))
	db.Put([]byte("b"), []byte("2"))
	db.Put([]byte("c"), []byte("3"))

	// Replay multiple times
	for i := 0; i < 5; i++ {
		dbi, err := engine.Open(dir)
		if err != nil {
			t.Fatal(err)
		}

		v, err := dbi.Get([]byte("b"))
		if err != nil || string(v) != "2" {
			t.Fatalf("non-deterministic replay on iteration %d", i)
		}
	}
}
