package crash

import (
	"os"
	"os/exec"
	"testing"

	"vern_kv0.5/engine"
)

func TestCrashBeforeWALFsync(t *testing.T) {
	dir := t.TempDir()

	cmd := exec.Command(
		"go", "run", "./helpers/crash_main.go", dir,
	)
	cmd.Env = append(os.Environ(),
		"VERN_CRASH_POINT=before_wal_fsync",
	)
	_ = cmd.Run() // expected SIGKILL

	// Restart engine — MUST NOT crash
	db, err := engine.Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Key MAY or MAY NOT exist.
	// If it exists, it must be correct.
	val, err := db.Get([]byte("a"))
	if err == nil && string(val) != "1" {
		t.Fatalf("corrupted value after crash before fsync")
	}
}
