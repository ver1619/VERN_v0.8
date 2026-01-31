package wal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWALTruncation(t *testing.T) {
	dir := t.TempDir()

	// Create two WAL segments
	w1, _ := OpenSegment(filepath.Join(dir, "wal_000001.log"))
	w2, _ := OpenSegment(filepath.Join(dir, "wal_000002.log"))

	w1.Append(mustEncode(1))
	w1.Append(mustEncode(2))
	w1.Sync()
	w1.Close()

	w2.Append(mustEncode(3))
	w2.Sync()
	w2.Close()

	// Truncate with cutoff = 2
	if err := Truncate(dir, 2); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, "wal_000001.log")); err == nil {
		t.Fatalf("expected first segment to be deleted")
	}

	if _, err := os.Stat(filepath.Join(dir, "wal_000002.log")); err != nil {
		t.Fatalf("expected second segment to remain")
	}
}

// helper
func mustEncode(seq uint64) []byte {
	b, err := EncodeRecord(Batch{
		SeqStart: seq,
		Records: []LogicalRecord{
			{Key: []byte("k"), Value: []byte("v"), Type: LogicalTypePut},
		},
	})
	if err != nil {
		panic(err)
	}
	return b
}
