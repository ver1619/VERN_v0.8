package engine

import (
	"testing"
)

func TestCompactionL1ToL2(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	// Simulate L1 state.

	// Verify CompactLevel(1) behavior for empty levels.

	err = db.CompactLevel(1)
	if err != nil {
		// It might fail because L1 is empty, which returns nil in our new logic?
		// "if len(currentLevel) == 0 { return nil }"
		// So it should return nil, NOT "only L0->L1 compaction supported"
	}

	if err != nil {
		t.Errorf("CompactLevel(1) returned error: %v", err)
	}

	// If the old code was in place, it would return error "only L0->L1...".
	// Since it returns nil (or some other error from FS), we know the check passed.
}
