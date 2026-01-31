package internal

import (
	"bytes"
	"testing"
)

func TestInternalKeyEncodeDecode(t *testing.T) {
	key := []byte("apple")
	seq := uint64(42)

	ik := EncodeInternalKey(key, seq, RecordTypeValue)
	decoded, err := DecodeInternalKey(ik)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(decoded.UserKey, key) {
		t.Fatalf("user key mismatch")
	}
	if decoded.Seq != seq {
		t.Fatalf("sequence mismatch")
	}
	if decoded.Type != RecordTypeValue {
		t.Fatalf("type mismatch")
	}
}

func TestComparatorOrdering(t *testing.T) {
	cmp := Comparator{}

	a1 := EncodeInternalKey([]byte("a"), 10, RecordTypeValue)
	a2 := EncodeInternalKey([]byte("a"), 9, RecordTypeValue)
	a3 := EncodeInternalKey([]byte("a"), 9, RecordTypeTombstone)
	b1 := EncodeInternalKey([]byte("b"), 1, RecordTypeValue)

	if cmp.Compare(a1, a2) >= 0 {
		t.Fatalf("newer sequence must come first")
	}
	if cmp.Compare(a2, a3) >= 0 {
		t.Fatalf("value must sort before tombstone at same seq")
	}
	if cmp.Compare(a1, b1) >= 0 {
		t.Fatalf("user key ordering violated")
	}
}

func TestExtractHelpers(t *testing.T) {
	key := EncodeInternalKey([]byte("x"), 7, RecordTypeTombstone)

	uk := ExtractUserKey(key)
	if !bytes.Equal(uk, []byte("x")) {
		t.Fatalf("extract user key failed")
	}

	seq, typ, err := ExtractTrailer(key)
	if err != nil {
		t.Fatal(err)
	}

	if seq != 7 || typ != RecordTypeTombstone {
		t.Fatalf("extract trailer failed")
	}
}
