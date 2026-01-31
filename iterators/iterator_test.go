package iterators

import (
	"os"
	"path/filepath"
	"testing"

	"vern_kv0.5/internal"
	"vern_kv0.5/memtable"
	"vern_kv0.5/sstable"
)

func TestMergeMemtableAndSSTable(t *testing.T) {
	dir := t.TempDir()
	sstPath := filepath.Join(dir, "test.sst")

	// ---- Build SSTable ----
	f, err := os.Create(sstPath)
	if err != nil {
		t.Fatal(err)
	}

	sstable.WriteRecord(f,
		internal.EncodeInternalKey([]byte("a"), 1, internal.RecordTypeValue),
		[]byte("old"),
	)
	sstable.WriteRecord(f,
		internal.EncodeInternalKey([]byte("b"), 1, internal.RecordTypeValue),
		[]byte("b1"),
	)
	f.Close()

	sstIt, err := sstable.NewIterator(sstPath)
	if err != nil {
		t.Fatal(err)
	}

	// ---- Build Memtable ----
	mt := memtable.New()
	mt.Insert(
		internal.EncodeInternalKey([]byte("a"), 2, internal.RecordTypeValue),
		[]byte("new"),
	)

	mtIt := NewMemtableIterator(mt)

	// ---- Merge ----
	merge := NewMergeIterator([]InternalIterator{
		mtIt,
		sstIt,
	})
	merge.SeekToFirst()

	if !merge.Valid() {
		t.Fatalf("merge iterator should be valid")
	}

	if string(merge.Value()) != "new" {
		t.Fatalf("expected memtable version to win")
	}

	merge.Next()
	if !merge.Valid() {
		t.Fatalf("expected second key")
	}

	if string(merge.Value()) != "b1" {
		t.Fatalf("unexpected value for key b")
	}

	merge.Next()
	if merge.Valid() {
		t.Fatalf("expected iterator exhaustion")
	}
}
