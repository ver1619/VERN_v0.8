package sstable

import (
	"bytes"
	"encoding/binary"
)

// BlockBuilder constructs a block of KV records.
// Format:
// Record 1
// Record 2
// ...
// Restart Points (offsets to records)
// Num Restarts (uint32)
type BlockBuilder struct {
	buf           bytes.Buffer
	restarts      []uint32
	restartCount  int
	counter       int // items since last restart
	lastUnsafeKey []byte
	finished      bool
}

const (
	// restartInterval is the number of keys between restart points.
	restartInterval = 16
)

func NewBlockBuilder() *BlockBuilder {
	return &BlockBuilder{
		restarts: []uint32{0}, // first restart point is at 0
	}
}

func (b *BlockBuilder) Reset() {
	b.buf.Reset()
	b.restarts = b.restarts[:0]
	b.restarts = append(b.restarts, 0)
	b.counter = 0
	b.lastUnsafeKey = nil
	b.finished = false
}

func (b *BlockBuilder) Add(key, value []byte) {
	// Prefix compression could go here, for now using simple encoding
	// Shared (0) | Unshared (len) | Value Len | Key | Value

	// shared := 0
	// TODO: Implement prefix compression if needed. For now simplified:
	// Format without prefix compression:
	// Key Len (varint) | Value Len (varint) | Key | Value

	// If we reached restart interval, add a new restart point
	if b.counter >= restartInterval {
		b.restarts = append(b.restarts, uint32(b.buf.Len()))
		b.counter = 0
	}

	// Write simple length-prefixed key/value for now
	putVarint(&b.buf, uint64(len(key)))
	putVarint(&b.buf, uint64(len(value)))
	b.buf.Write(key)
	b.buf.Write(value)

	b.lastUnsafeKey = key // Keep reference, be careful if caller reuses buffer
	b.counter++
}

func (b *BlockBuilder) Finish() []byte {
	// Append restarts
	for _, r := range b.restarts {
		binary.Write(&b.buf, binary.LittleEndian, r)
	}
	// Append number of restarts
	binary.Write(&b.buf, binary.LittleEndian, uint32(len(b.restarts)))
	b.finished = true
	return b.buf.Bytes()
}

func (b *BlockBuilder) CurrentSize() int {
	return b.buf.Len()
}

func (b *BlockBuilder) Empty() bool {
	return b.buf.Len() == 0
}

// BlockIterator iterates over a block.
type BlockIterator struct {
	data        []byte
	restarts    uint32 // Offset where restarts begin
	numRestarts uint32

	offset     int // Current offset in data
	nextOffset int // Next offset

	key   []byte
	value []byte
	err   error
	valid bool
}

func NewBlockIterator(data []byte) *BlockIterator {
	n := uint32(len(data))
	if n < 4 {
		return &BlockIterator{err: ErrBlockCorrupt}
	}

	numRestarts := binary.LittleEndian.Uint32(data[n-4:])
	restartsOffset := n - 4 - (numRestarts * 4)

	if restartsOffset > n-4 {
		return &BlockIterator{err: ErrBlockCorrupt}
	}

	return &BlockIterator{
		data:        data,
		restarts:    restartsOffset,
		numRestarts: numRestarts,
	}
}

func (it *BlockIterator) Valid() bool {
	return it.valid && it.err == nil
}

func (it *BlockIterator) Key() []byte {
	return it.key
}

func (it *BlockIterator) Value() []byte {
	return it.value
}

func (it *BlockIterator) SeekToFirst() {
	it.SeekToRestartPoint(0)
	it.ParseNext()
}

func (it *BlockIterator) Seek(target []byte) {
	// Binary search in restarts
	left := uint32(0)
	right := it.numRestarts - 1

	for left < right {
		mid := (left + right + 1) / 2
		regionOffset := it.GetRestartPoint(mid)

		// Parse key at this restart point
		// Note: Restart points always point to full keys if we had compression
		// Since we don't have compression yet, just read it.
		key, _, _, ok := it.ParseEntry(int(regionOffset))
		if !ok {
			it.err = ErrBlockCorrupt
			return
		}

		if bytes.Compare(key, target) < 0 {
			left = mid
		} else {
			right = mid - 1
		}
	}

	// Linear scan from the found restart point
	it.SeekToRestartPoint(left)

	for {
		if !it.ParseNext() {
			return
		}
		if bytes.Compare(it.key, target) >= 0 {
			return
		}
	}
}

func (it *BlockIterator) Next() {
	it.offset = it.nextOffset
	if it.offset >= int(it.restarts) {
		it.valid = false
		return
	}
	it.ParseNext()
}

func (it *BlockIterator) SeekToRestartPoint(index uint32) {
	if index >= it.numRestarts {
		index = 0
	}
	offset := it.GetRestartPoint(index)
	it.nextOffset = int(offset)
	it.offset = int(offset) // Prepare for Next/ParseNext
	it.valid = false        // valid will be true after ParseNext
}

func (it *BlockIterator) GetRestartPoint(index uint32) uint32 {
	offset := it.restarts + (index * 4)
	return binary.LittleEndian.Uint32(it.data[offset:])
}

func (it *BlockIterator) ParseNext() bool {
	if it.nextOffset >= int(it.restarts) {
		it.valid = false
		return false
	}

	it.offset = it.nextOffset
	key, value, n, ok := it.ParseEntry(it.offset)
	if !ok {
		it.valid = false
		it.err = ErrBlockCorrupt
		return false
	}

	it.key = key
	it.value = value
	it.nextOffset = it.offset + n
	it.valid = true
	return true
}

func (it *BlockIterator) ParseEntry(offset int) (key, value []byte, byteCount int, ok bool) {
	if offset >= int(it.restarts) {
		return nil, nil, 0, false
	}

	src := it.data[offset:]

	klen, n1 := binary.Uvarint(src)
	if n1 <= 0 {
		return nil, nil, 0, false
	}

	vlen, n2 := binary.Uvarint(src[n1:])
	if n2 <= 0 {
		return nil, nil, 0, false
	}

	total := n1 + n2 + int(klen) + int(vlen)
	if len(src) < total {
		return nil, nil, 0, false
	}

	kstart := n1 + n2
	vstart := kstart + int(klen)

	key = src[kstart : kstart+int(klen)]
	value = src[vstart : vstart+int(vlen)]
	return key, value, total, true
}

func putVarint(buf *bytes.Buffer, x uint64) {
	var b [10]byte
	n := binary.PutUvarint(b[:], x)
	buf.Write(b[:n])
}
