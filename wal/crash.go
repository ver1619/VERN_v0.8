package wal

import (
	"os"
	"syscall"
)

func maybeCrash(point string) {
	if os.Getenv("VERN_CRASH_POINT") == point {
		// Immediate, uncatchable process death
		syscall.Kill(os.Getpid(), syscall.SIGKILL)
	}
}
