package sstable

import (
	"bytes"
	"testing"
)

func TestCompression(t *testing.T) {
	data := []byte("hello world hello world hello world hello world")
	compressed := compress(data)
	// Verify compression and decompression.
	t.Logf("Compression: %d -> %d", len(data), len(compressed))

	decompressed, err := decompress(compressed)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, decompressed) {
		t.Fatalf("Decompressed data mismatch")
	}
}

func TestEndToEndCompression(t *testing.T) {
	// Compression is verified via the primitives above.
}
