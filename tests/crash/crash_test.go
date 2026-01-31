package crash

import (
	"fmt"
	"testing"

	"vern_kv0.8/engine"
)

func TestCrashConsistency(t *testing.T) {
	dir := t.TempDir()

	// 1. Setup and write some data
	db, err := engine.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 50; i++ {
		db.Put([]byte(fmt.Sprintf("p-%d", i)), []byte("data"))
	}

	// Simulate "hard" close (no manifest/WAL finish sequence if we were doing something complex)
	// But current engine.Close is simple.
	// Real crash test would be killing the process, but here we just Close and re-open
	// and ensure all durable Puts (which are sync in v0.8) are back.
	db.Close()

	db, err = engine.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for i := 0; i < 50; i++ {
		_, err := db.Get([]byte(fmt.Sprintf("p-%d", i)))
		if err != nil {
			t.Errorf("missing key p-%d after recovery", i)
		}
	}
}
