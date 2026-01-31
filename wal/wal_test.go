package wal

import (
	"os"
	"testing"
)

func TestWALAppendAndRotate(t *testing.T) {
	dir := t.TempDir()

	w, err := OpenWAL(dir, 64)
	if err != nil {
		t.Fatal(err)
	}

	batch := Batch{
		SeqStart: 1,
		Records: []LogicalRecord{
			{Key: []byte("a"), Value: []byte("1"), Type: logicalTypePut},
		},
	}

	for i := 0; i < 10; i++ {
		if err := w.Append(batch); err != nil {
			t.Fatal(err)
		}
	}

	if err := w.Sync(); err != nil {
		t.Fatal(err)
	}

	segments := w.Segments()
	if len(segments) < 2 {
		t.Fatalf("expected rotation, got %d segments", len(segments))
	}

	for _, p := range segments {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("missing segment %s", p)
		}
	}
}

func TestWALReopen(t *testing.T) {
	dir := t.TempDir()

	w1, _ := OpenWAL(dir, 64)
	w1.Append(Batch{
		SeqStart: 1,
		Records: []LogicalRecord{
			{Key: []byte("x"), Value: []byte("y"), Type: logicalTypePut},
		},
	})
	w1.Sync()
	w1.Close()

	w2, err := OpenWAL(dir, 64)
	if err != nil {
		t.Fatal(err)
	}

	if len(w2.Segments()) == 0 {
		t.Fatalf("expected existing WAL segments")
	}
}

func TestWALCloseClosesAllSegments(t *testing.T) {
	dir := t.TempDir()

	w, _ := OpenWAL(dir, 32)
	w.Append(Batch{
		SeqStart: 1,
		Records: []LogicalRecord{
			{Key: []byte("k"), Value: []byte("v"), Type: logicalTypePut},
		},
	})
	w.Close()

	files, _ := os.ReadDir(dir)
	if len(files) == 0 {
		t.Fatalf("expected WAL files")
	}
}
