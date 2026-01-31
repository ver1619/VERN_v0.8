package wal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSegmentAppendAndSync(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wal_000001.log")

	seg, err := OpenSegment(path)
	if err != nil {
		t.Fatal(err)
	}

	data := []byte("hello wal")

	if err := seg.Append(data); err != nil {
		t.Fatal(err)
	}

	if seg.Size() != int64(len(data)) {
		t.Fatalf("unexpected segment size")
	}

	if err := seg.Sync(); err != nil {
		t.Fatal(err)
	}

	if err := seg.Close(); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if string(raw) != "hello wal" {
		t.Fatalf("unexpected file contents")
	}
}

func TestSegmentReopenAppend(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wal_000001.log")

	seg1, _ := OpenSegment(path)
	seg1.Append([]byte("first"))
	seg1.Sync()
	seg1.Close()

	seg2, _ := OpenSegment(path)
	seg2.Append([]byte("second"))
	seg2.Sync()
	seg2.Close()

	raw, _ := os.ReadFile(path)
	if string(raw) != "firstsecond" {
		t.Fatalf("append on reopen failed")
	}
}

func TestAppendAfterCloseFails(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wal_000001.log")

	seg, _ := OpenSegment(path)
	seg.Close()

	err := seg.Append([]byte("x"))
	if err == nil {
		t.Fatalf("expected error after close")
	}
}
