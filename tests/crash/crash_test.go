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
