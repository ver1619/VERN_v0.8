package sstable

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"os"
)

var errCorruptSSTable = errors.New("corrupt sstable")

// Record represents one SSTable entry.
type Record struct {
	Key   []byte
	Value []byte
}

// WriteRecord is used ONLY for tests.
func WriteRecord(w io.Writer, key, value []byte) error {
	var payload bytes.Buffer

	binary.Write(&payload, binary.LittleEndian, uint32(len(key)))
	binary.Write(&payload, binary.LittleEndian, uint32(len(value)))
	payload.Write(key)
	payload.Write(value)

	length := uint32(payload.Len())

	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(0)) // CRC placeholder
	binary.Write(&buf, binary.LittleEndian, length)
	buf.Write(payload.Bytes())

	raw := buf.Bytes()
	crc := crc32.ChecksumIEEE(raw[4:])
	binary.LittleEndian.PutUint32(raw[0:], crc)

	_, err := w.Write(raw)
	return err
}

// Open opens an SSTable for reading.
func Open(path string) (*os.File, error) {
	return os.Open(path)
}

// ReadRecord reads one record at offset.
func ReadRecord(data []byte, offset int) (Record, int, error) {
	if offset+8 > len(data) {
		return Record{}, 0, io.EOF
	}

	wantCRC := binary.LittleEndian.Uint32(data[offset:])
	length := binary.LittleEndian.Uint32(data[offset+4:])

	total := int(8 + length)
	if offset+total > len(data) {
		return Record{}, 0, errCorruptSSTable
	}

	if crc32.ChecksumIEEE(data[offset+4:offset+total]) != wantCRC {
		return Record{}, 0, errCorruptSSTable
	}

	payload := data[offset+8 : offset+total]
	rd := bytes.NewReader(payload)

	var klen, vlen uint32
	binary.Read(rd, binary.LittleEndian, &klen)
	binary.Read(rd, binary.LittleEndian, &vlen)

	key := make([]byte, klen)
	value := make([]byte, vlen)

	rd.Read(key)
	rd.Read(value)

	return Record{Key: key, Value: value}, total, nil
}
