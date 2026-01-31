package memtable

import (
	"testing"

	"vern_kv0.5/internal"
)

func TestMemtableInsertAndSize(t *testing.T) {
	mt := New()

	if mt.Size() != 0 {
		t.Fatalf("expected empty memtable")
	}

	k1 := internal.EncodeInternalKey([]byte("a"), 1, internal.RecordTypeValue)
	mt.Insert(k1, []byte("1"))

	if mt.Size() != 1 {
		t.Fatalf("expected size 1")
	}
}

func TestMemtableOrderingByUserKey(t *testing.T) {
	mt := New()

	kb := internal.EncodeInternalKey([]byte("b"), 1, internal.RecordTypeValue)
	ka := internal.EncodeInternalKey([]byte("a"), 1, internal.RecordTypeValue)

	mt.Insert(kb, []byte("b"))
	mt.Insert(ka, []byte("a"))

	if string(mt.entries[0].key[:1]) != "a" {
		t.Fatalf("expected key 'a' first")
	}
}

func TestMemtableOrderingBySequenceDesc(t *testing.T) {
	mt := New()

	k1 := internal.EncodeInternalKey([]byte("a"), 1, internal.RecordTypeValue)
	k2 := internal.EncodeInternalKey([]byte("a"), 2, internal.RecordTypeValue)

	mt.Insert(k1, []byte("old"))
	mt.Insert(k2, []byte("new"))

	seq, _, _ := internal.ExtractTrailer(mt.entries[0].key)
	if seq != 2 {
		t.Fatalf("expected newer sequence first")
	}
}

func TestMemtableStoresTombstone(t *testing.T) {
	mt := New()

	put := internal.EncodeInternalKey([]byte("x"), 1, internal.RecordTypeValue)
	del := internal.EncodeInternalKey([]byte("x"), 2, internal.RecordTypeTombstone)

	mt.Insert(put, []byte("v"))
	mt.Insert(del, nil)

	if mt.Size() != 2 {
		t.Fatalf("expected both value and tombstone stored")
	}
}
