package sstable

import (
	"io"
	"os"
)

// Builder constructs SSTables.
type Builder struct {
	file       *os.File
	writer     io.Writer
	dataBlock  *BlockBuilder
	indexBlock *BlockBuilder

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

	// Pending index entry
	if b.pendingIndexEntry {
		k := make([]byte, len(b.lastKey))
		copy(k, b.lastKey)

		encodedHandle := make([]byte, 16)
		b.pendingHandle.EncodeTo(encodedHandle)

		b.indexBlock.Add(k, encodedHandle)
		b.pendingIndexEntry = false
	}

	b.dataBlock.Add(key, value)

	// Update filter state.
	kCopy := make([]byte, len(key))
	copy(kCopy, key)
	b.keys = append(b.keys, kCopy)

	b.lastKey = key

	if b.dataBlock.CurrentSize() >= blockSize {
		if err := b.Flush(); err != nil {
			b.err = err
			return err
		}
	}

	return nil
}

// Size returns the approximate size of the file being built.
func (b *Builder) Size() uint64 {
	return b.offset + uint64(b.dataBlock.CurrentSize())
}

// Flush writes current block.
func (b *Builder) Flush() error {
	if b.dataBlock.Empty() {
		return nil
	}

	// Write filter block.
	content := b.dataBlock.Finish()

	// Compress block data.
	compressed := compress(content)
	// Use compression if it reduces size.

	var final []byte
	var cType byte

	if len(compressed) < len(content)-2 { // simple heuristic
		final = compressed
		cType = byte(ZlibCompression)
	} else {
		final = content
		cType = byte(NoCompression)
	}

	// Write to file
	n, err := b.writer.Write(final)
	if err != nil {
		return err
	}

	// Write compression type (1 byte)
	if _, err := b.writer.Write([]byte{cType}); err != nil {
		return err
	}
	n++

	// Record handle
	handle := BlockHandle{
		Offset: b.offset,
		Length: uint64(n),
	}

	// Update state
	b.offset += uint64(n)
	b.dataBlock.Reset()

	// Prepare index entry.
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

	// Flush remaining
	if err := b.Flush(); err != nil {
		return err
	}

	// Prepare index entry.
	if b.pendingIndexEntry {
		b.indexBlock.Add(b.lastKey, func() []byte {
			buf := make([]byte, 16)
			b.pendingHandle.EncodeTo(buf)
			return buf
		}())
		b.pendingIndexEntry = false
	}

	// Write filter block.
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

	// Write meta index block.
	metaIndexContent := b.metaIndexBlock.Finish()
	// Compress/Wrap MetaIndex
	compressedMeta := compress(metaIndexContent)
	var finalMeta []byte
	var metaType byte
	if len(compressedMeta) < len(metaIndexContent)-2 {
		finalMeta = compressedMeta
		metaType = byte(ZlibCompression)
	} else {
		finalMeta = metaIndexContent
		metaType = byte(NoCompression)
	}

	n, err := b.writer.Write(finalMeta)
	if err != nil {
		return err
	}
	if _, err := b.writer.Write([]byte{metaType}); err != nil {
		return err
	}
	n++

	metaIndexHandle := BlockHandle{Offset: b.offset, Length: uint64(n)}
	b.offset += uint64(n)

	// Write meta index block.
	indexContent := b.indexBlock.Finish()
	// Compress/Wrap Index
	compressedIndex := compress(indexContent)
	var finalIndex []byte
	var indexType byte
	if len(compressedIndex) < len(indexContent)-2 {
		finalIndex = compressedIndex
		indexType = byte(ZlibCompression)
	} else {
		finalIndex = indexContent
		indexType = byte(NoCompression)
	}

	n, err = b.writer.Write(finalIndex)
	if err != nil {
		return err
	}
	if _, err := b.writer.Write([]byte{indexType}); err != nil {
		return err
	}
	n++

	indexHandle := BlockHandle{
		Offset: b.offset,
		Length: uint64(n),
	}
	b.offset += uint64(n)

	// Footer
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
