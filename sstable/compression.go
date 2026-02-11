package sstable

import (
	"bytes"
	"compress/zlib"
	"io"
)

const (
	NoCompression     = 0
	ZlibCompression   = 1
	SnappyCompression = 2 // Future
	ZstdCompression   = 3 // Future
)

func compress(src []byte) []byte {
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	w.Write(src)
	w.Close()
	return b.Bytes()
}

func decompress(src []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}
