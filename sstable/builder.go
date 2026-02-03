package sstable

import (
	"io"
	"os"
)

// Builder constructs an SSTable.
type Builder struct {
	file        *os.File
	writer      io.Writer
	dataBlock   *BlockBuilder
	indexBlock  *BlockBuilder
	filterBlock *BlockBuilder // Not used as BlockBuilder, but we need to write it. Actually filter is just raw bytes.

	metaIndexBlock *BlockBuilder

	pendingIndexEntry bool
	pendingHandle     BlockHandle
	lastKey           []byte

	offset uint64
	err    error

	// Filter support
	filterPolicy FilterPolicy
	keys         [][]byte // Accumulate all keys for the table-wide filter
}

const (
	// Target block size before flushing (4KB)
	blockSize = 4 * 1024
)

func NewBuilder(filename string) (*Builder, error) {
	f, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	return &Builder{
		file:           f,
		writer:         f,
		dataBlock:      NewBlockBuilder(),
		indexBlock:     NewBlockBuilder(),
		metaIndexBlock: NewBlockBuilder(),
		filterPolicy:   NewBloomFilter(10), // Default 10 bits per key
		keys:           make([][]byte, 0, 1024),
	}, nil
}

func (b *Builder) Add(key, value []byte) error {
	if b.err != nil {
		return b.err
	}

	// If the previous block was flushed, we need to add an index entry for it.
	if b.pendingIndexEntry {
		k := make([]byte, len(b.lastKey))
		copy(k, b.lastKey)

		encodedHandle := make([]byte, 16)
		b.pendingHandle.EncodeTo(encodedHandle)

		b.indexBlock.Add(k, encodedHandle)
		b.pendingIndexEntry = false
	}

	b.dataBlock.Add(key, value)

	// Save key for filter
	kCopy := make([]byte, len(key))
	copy(kCopy, key)
	b.keys = append(b.keys, kCopy)

	b.lastKey = key
	// lastKey copy handled in Flush().

	if b.dataBlock.CurrentSize() >= blockSize {
		if err := b.Flush(); err != nil {
			b.err = err
			return err
		}
	}

	return nil
}

// Flush forces the current data block to be written to file.
func (b *Builder) Flush() error {
	if b.dataBlock.Empty() {
		return nil
	}

	// 1. Finish block
	content := b.dataBlock.Finish()

	// 2. Write to file
	n, err := b.writer.Write(content)
	if err != nil {
		return err
	}

	// 3. Record handle
	handle := BlockHandle{
		Offset: b.offset,
		Length: uint64(n),
	}

	// 4. Update state
	b.offset += uint64(n)
	b.dataBlock.Reset()

	// 5. Manage Index Entry
	lk := make([]byte, len(b.lastKey))
	copy(lk, b.lastKey)
	b.lastKey = lk

	b.pendingHandle = handle
	b.pendingIndexEntry = true

	return nil
}

func (b *Builder) Close() error {
	if b.err != nil {
		return b.err
	}

	// Flush any remaining data
	if err := b.Flush(); err != nil {
		return err
	}

	// Add pending index entry for the last block
	if b.pendingIndexEntry {
		b.indexBlock.Add(b.lastKey, func() []byte {
			buf := make([]byte, 16)
			b.pendingHandle.EncodeTo(buf)
			return buf
		}())
		b.pendingIndexEntry = false
	}

	// --- Write Filter Block ---
	var filterHandle BlockHandle
	if b.filterPolicy != nil && len(b.keys) > 0 {
		filterData := b.filterPolicy.CreateFilter(b.keys)
		n, err := b.writer.Write(filterData)
		if err != nil {
			return err
		}
		filterHandle = BlockHandle{Offset: b.offset, Length: uint64(n)}
		b.offset += uint64(n)

		// Add to MetaIndex
		encodedFilterHandle := make([]byte, 16)
		filterHandle.EncodeTo(encodedFilterHandle)
		b.metaIndexBlock.Add([]byte(b.filterPolicy.Name()), encodedFilterHandle)
	}

	// --- Write MetaIndex Block ---
	metaIndexContent := b.metaIndexBlock.Finish()
	n, err := b.writer.Write(metaIndexContent)
	if err != nil {
		return err
	}
	metaIndexHandle := BlockHandle{Offset: b.offset, Length: uint64(n)}
	b.offset += uint64(n)

	// --- Write Index Block ---
	indexContent := b.indexBlock.Finish()
	n, err = b.writer.Write(indexContent)
	if err != nil {
		return err
	}

	indexHandle := BlockHandle{
		Offset: b.offset,
		Length: uint64(n),
	}
	b.offset += uint64(n)

	// --- Write Footer ---
	footer := Footer{
		MetaindexHandle: metaIndexHandle,
		IndexHandle:     indexHandle,
	}
	var footerBuf [FooterSize]byte
	footer.EncodeTo(footerBuf[:])

	if _, err := b.writer.Write(footerBuf[:]); err != nil {
		return err
	}

	return b.file.Close()
}
