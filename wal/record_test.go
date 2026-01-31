package wal

import (
	"bytes"
	"testing"
)

func TestEncodeDecodeRecord(t *testing.T) {
	batch := Batch{
		SeqStart: 100,
		Records: []LogicalRecord{
			{Key: []byte("a"), Value: []byte("1"), Type: logicalTypePut},
			{Key: []byte("b"), Type: logicalTypeDelete},
		},
	}

	raw, err := EncodeRecord(batch)
	if err != nil {
		t.Fatal(err)
	}

	decoded, n, err := DecodeRecord(raw)
	if err != nil {
		t.Fatal(err)
	}
	if n != len(raw) {
		t.Fatalf("length mismatch")
	}

	if decoded.SeqStart != batch.SeqStart {
		t.Fatalf("seq start mismatch")
	}
	if len(decoded.Records) != 2 {
		t.Fatalf("record count mismatch")
	}

	if !bytes.Equal(decoded.Records[0].Value, []byte("1")) {
		t.Fatalf("value mismatch")
	}
}

func TestCRCFailureStopsDecode(t *testing.T) {
	batch := Batch{
		SeqStart: 1,
		Records: []LogicalRecord{
			{Key: []byte("x"), Value: []byte("y"), Type: logicalTypePut},
		},
	}

	raw, _ := EncodeRecord(batch)
	raw[len(raw)-1] ^= 0xFF // corrupt

	_, _, err := DecodeRecord(raw)
	if err == nil {
		t.Fatalf("expected CRC failure")
	}
}

func TestPartialRecordFails(t *testing.T) {
	batch := Batch{
		SeqStart: 1,
		Records: []LogicalRecord{
			{Key: []byte("x"), Value: []byte("y"), Type: logicalTypePut},
		},
	}

	raw, _ := EncodeRecord(batch)
	raw = raw[:len(raw)-5]

	_, _, err := DecodeRecord(raw)
	if err == nil {
		t.Fatalf("expected failure on partial record")
	}
}
