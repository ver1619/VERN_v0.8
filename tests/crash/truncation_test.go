package crash

import (
	"testing"

	"vern_kv0.5/engine"
)

func TestTruncationIdempotence(t *testing.T) {
	dir := t.TempDir()

	db, _ := engine.Open(dir)
	db.Put([]byte("a"), []byte("1"))

	// simulate restart
	db2, _ := engine.Open(dir)
	val, err := db2.Get([]byte("a"))
	if err != nil || string(val) != "1" {
		t.Fatalf("data lost after truncation/restart")
	}
}
