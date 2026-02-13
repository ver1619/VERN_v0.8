package sstable

import (
	"bytes"
	"encoding/binary"
)

// BlockBuilder constructs a KV block.
type BlockBuilder struct {
	buf           bytes.Buffer
	restarts      []uint32
	restartCount  int
	counter       int // Items since last restart.
	lastUnsafeKey []byte
	finished      bool
}

const (
	restartInterval = 16
)

func NewBlockBuilder() *BlockBuilder {
	return &BlockBuilder{
		restarts: []uint32{0}, // First restart point.
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

	// Calculate shared prefix
	shared := 0
	if b.counter < restartInterval && b.lastUnsafeKey != nil {
		minLen := len(b.lastUnsafeKey)
		if len(key) < minLen {
			minLen = len(key)
		}
		for shared < minLen && b.lastUnsafeKey[shared] == key[shared] {
			shared++
		}
	} else {
		// Restart point: shared prefix is 0
		if b.counter >= restartInterval {
			b.restarts = append(b.restarts, uint32(b.buf.Len()))
			b.counter = 0
		}
	}

	unshared := len(key) - shared

	// Write KV with prefix compression
	putVarint(&b.buf, uint64(shared))
	putVarint(&b.buf, uint64(unshared))
	putVarint(&b.buf, uint64(len(value)))
	b.buf.Write(key[shared:])
	b.buf.Write(value)

	b.lastUnsafeKey = key // Keep reference for next key.
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

// BlockIterator parses a block.
type BlockIterator struct {
	data        []byte
	restarts    uint32 // Start of restart points.
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
	// Binary search restart points.
	left := uint32(0)
	right := it.numRestarts - 1

	for left < right {
		mid := (left + right + 1) / 2
		regionOffset := it.GetRestartPoint(mid)

		// Parse key
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

	// Scan linearly for target.
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
	it.valid = false        // Validated after ParseNext.
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
	current := 0

	// Read shared key length
	shared, n := binary.Uvarint(src[current:])
	if n <= 0 {
		return nil, nil, 0, false
	}
	current += n

	// Read unshared key length
	unshared, n := binary.Uvarint(src[current:])
	if n <= 0 {
		return nil, nil, 0, false
	}
	current += n

	// Read value length
	vlen, n := binary.Uvarint(src[current:])
	if n <= 0 {
		return nil, nil, 0, false
	}
	current += n

	// Check bounds
	if len(src) < current+int(unshared)+int(vlen) {
		return nil, nil, 0, false
	}

	// Reconstruct key
	fullKeyLength := int(shared) + int(unshared)
	fullKey := make([]byte, fullKeyLength)

	if shared > 0 {
		if len(it.key) < int(shared) {
			return nil, nil, 0, false
		}
		copy(fullKey, it.key[:shared])
	}

	copy(fullKey[shared:], src[current:current+int(unshared)])
	current += int(unshared)

	value = src[current : current+int(vlen)]
	current += int(vlen)

	return fullKey, value, current, true
}

func putVarint(buf *bytes.Buffer, x uint64) {
	var b [10]byte
	n := binary.PutUvarint(b[:], x)
	buf.Write(b[:n])
}
