package crash

import (
	"testing"

	"vern_kv0.5/engine"
)

func TestRecoveryIsDeterministic(t *testing.T) {
	dir := t.TempDir()

	db, _ := engine.Open(dir)
	db.Put([]byte("x"), []byte("1"))
	db.Put([]byte("y"), []byte("2"))

	db1, _ := engine.Open(dir)
	db2, _ := engine.Open(dir)

	v1, _ := db1.Get([]byte("x"))
	v2, _ := db2.Get([]byte("x"))

	if string(v1) != string(v2) {
		t.Fatalf("non-deterministic recovery")
	}
}
