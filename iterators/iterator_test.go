package iterators

import (
	"path/filepath"
	"testing"

	"vern_kv0.8/internal"
	"vern_kv0.8/memtable"
	"vern_kv0.8/sstable"
)

func TestMergeMemtableAndSSTable(t *testing.T) {
	dir := t.TempDir()
	sstPath := filepath.Join(dir, "test.sst")

	// ---- Build SSTable ----
	b, err := sstable.NewBuilder(sstPath)
	if err != nil {
		t.Fatal(err)
	}

	b.Add(internal.EncodeInternalKey([]byte("a"), 1, internal.RecordTypeValue), []byte("old"))
	b.Add(internal.EncodeInternalKey([]byte("b"), 1, internal.RecordTypeValue), []byte("b1"))

	if err := b.Close(); err != nil {
		t.Fatal(err)
	}

	sstIt, err := sstable.NewIterator(sstPath, nil)
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

func TestVersionFilterIterator(t *testing.T) {
	mt := memtable.New()

	mt.Insert(
		internal.EncodeInternalKey([]byte("a"), 3, internal.RecordTypeValue),
		[]byte("v3"),
	)
	mt.Insert(
		internal.EncodeInternalKey([]byte("a"), 2, internal.RecordTypeValue),
		[]byte("v2"),
	)
	mt.Insert(
		internal.EncodeInternalKey([]byte("a"), 1, internal.RecordTypeValue),
		[]byte("v1"),
	)

	mtIt := NewMemtableIterator(mt)
	filtered := NewVersionFilterIterator(mtIt, 2)
	merge := NewMergeIterator([]InternalIterator{filtered})

	merge.SeekToFirst()
	if !merge.Valid() {
		t.Fatalf("expected visible version")
	}

	if string(merge.Value()) != "v2" {
		t.Fatalf("expected v2, got %s", merge.Value())
	}
}

func TestVersionFilterHidesFutureWrites(t *testing.T) {
	mt := memtable.New()

	mt.Insert(
		internal.EncodeInternalKey([]byte("x"), 1, internal.RecordTypeValue),
		[]byte("old"),
	)
	mt.Insert(
		internal.EncodeInternalKey([]byte("x"), 5, internal.RecordTypeValue),
		[]byte("new"),
	)

	mtIt := NewMemtableIterator(mt)
	filtered := NewVersionFilterIterator(mtIt, 3)
	merge := NewMergeIterator([]InternalIterator{filtered})

	merge.SeekToFirst()
	if !merge.Valid() {
		t.Fatalf("expected visible entry")
	}

	if string(merge.Value()) != "old" {
		t.Fatalf("expected old value")
	}
}
