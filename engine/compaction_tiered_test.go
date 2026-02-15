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
		t.Errorf("CompactLevel(1) returned error: %v", err)
	}

}
