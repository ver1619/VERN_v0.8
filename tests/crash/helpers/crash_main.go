package main

import (
	"os"
	"syscall"

	"vern_kv0.5/engine"
)

func maybeCrash(point string) {
	if os.Getenv("VERN_CRASH_POINT") == point {
		syscall.Kill(os.Getpid(), syscall.SIGKILL)
	}
}

func main() {
	dir := os.Args[1]

	db, err := engine.Open(dir)
	if err != nil {
		os.Exit(1)
	}

	// Put but crash before fsync
	_ = db.Put([]byte("a"), []byte("1"))
	maybeCrash("before_wal_fsync")

	// Second write (never reached in fsync test)
	_ = db.Put([]byte("b"), []byte("2"))
	maybeCrash("after_put")
}
