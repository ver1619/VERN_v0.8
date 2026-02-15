package sstable

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"

	"vern_kv0.8/internal"
)

// Builder writes SSTables.
type Builder struct {
	file           *os.File
	writer         *bufio.Writer
	dataBlock      *BlockBuilder
	indexBlock     *BlockBuilder
	metaIndexBlock *BlockBuilder

	offset            uint64
	numEntries        uint64
	closed            bool
	lastKey           []byte
	pendingIndexEntry bool
	pendingHandle     BlockHandle
	err               error

	// Filter support.
	filterPolicy FilterPolicy
	keys         [][]byte
}

const (
	blockSize = 4 * 1024
)

func NewBuilder(filename string) (*Builder, error) {
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return nil, err
	}

	f, err := os.Create(filename)
	if err != nil {
		return nil, err
	}

	return &Builder{
		file:           f,
		writer:         bufio.NewWriter(f),
		dataBlock:      NewBlockBuilder(),
		indexBlock:     NewBlockBuilder(),
		metaIndexBlock: NewBlockBuilder(),
		filterPolicy:   NewBloomFilter(10),
		keys:           make([][]byte, 0, 1024),
	}, nil
}

// Add appends a key-value pair.
func (b *Builder) Add(key, value []byte) error {
	if b.err != nil {
		return b.err
	}
	if b.closed {
		return fmt.Errorf("builder closed")
	}
	// Ordering check removed. Internal key ordering cannot be validated via bytes.Compare.
	// Caller guarantees sorted input.

	// Pending index entry.
	if b.pendingIndexEntry {
		sep := findShortestSeparator(b.lastKey, key)
		encodedHandle := make([]byte, 16)
		b.pendingHandle.EncodeTo(encodedHandle)
		b.indexBlock.Add(sep, encodedHandle)
		b.pendingIndexEntry = false
	}

	b.dataBlock.Add(key, value)

	b.lastKey = make([]byte, len(key))
	copy(b.lastKey, key)

	if b.filterPolicy != nil {
		b.keys = append(b.keys, internal.ExtractUserKey(key))
	}
	b.numEntries++

	if b.dataBlock.CurrentSize() >= blockSize {
		if err := b.flushDataBlock(); err != nil {
			b.err = err
			return err
		}
	}

	return nil
}

func (b *Builder) Size() uint64 {
	return b.offset + uint64(b.dataBlock.CurrentSize())
}

func (b *Builder) flushDataBlock() error {
	if b.dataBlock.Empty() {
		return nil
	}

	content := b.dataBlock.Finish()
	compressed := compress(content)

	var final []byte
	var cType byte

	if len(compressed) < len(content)-2 {
		final = compressed
		cType = byte(ZlibCompression)
	} else {
		final = content
		cType = byte(NoCompression)
	}

	n, err := b.writer.Write(final)
	if err != nil {
		return err
	}

	if _, err := b.writer.Write([]byte{cType}); err != nil {
		return err
	}
	n++

	crc := crc32.ChecksumIEEE(final)
	crc = crc32.Update(crc, crc32.IEEETable, []byte{cType})

	if err := binary.Write(b.writer, binary.LittleEndian, crc); err != nil {
		return err
	}
	n += 4

	handle := BlockHandle{
		Offset: b.offset,
		Length: uint64(n),
	}

	b.offset += uint64(n)
	b.dataBlock.Reset()

	b.pendingHandle = handle
	b.pendingIndexEntry = true

	return nil
}

// Close finishes the SSTable.
func (b *Builder) Close() error {
	if b.closed {
		return nil
	}
	if b.err != nil {
		return b.err
	}

	// Flush remaining.
	if err := b.flushDataBlock(); err != nil {
		return err
	}

	if b.pendingIndexEntry {
		encodedHandle := make([]byte, 16)
		b.pendingHandle.EncodeTo(encodedHandle)
		b.indexBlock.Add(b.lastKey, encodedHandle)
		b.pendingIndexEntry = false
	}

	// Filter Block.
	var filterHandle BlockHandle
	if b.filterPolicy != nil && len(b.keys) > 0 {
		filterData := b.filterPolicy.CreateFilter(b.keys)
		n, err := b.writer.Write(filterData)
		if err != nil {
			return err
		}
		filterHandle = BlockHandle{Offset: b.offset, Length: uint64(n)}
		b.offset += uint64(n)

		encodedFilterHandle := make([]byte, 16)
		filterHandle.EncodeTo(encodedFilterHandle)
		b.metaIndexBlock.Add([]byte(b.filterPolicy.Name()), encodedFilterHandle)
	}

	// Meta Index.
	metaIndexContent := b.metaIndexBlock.Finish()

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

	crcMeta := crc32.ChecksumIEEE(finalMeta)
	crcMeta = crc32.Update(crcMeta, crc32.IEEETable, []byte{metaType})
	if err := binary.Write(b.writer, binary.LittleEndian, crcMeta); err != nil {
		return err
	}
	n += 4

	metaIndexHandle := BlockHandle{Offset: b.offset, Length: uint64(n)}
	b.offset += uint64(n)

	// Index Block.
	indexContent := b.indexBlock.Finish()

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

	crcIndex := crc32.ChecksumIEEE(finalIndex)
	crcIndex = crc32.Update(crcIndex, crc32.IEEETable, []byte{indexType})
	if err := binary.Write(b.writer, binary.LittleEndian, crcIndex); err != nil {
		return err
	}
	n += 4

	indexHandle := BlockHandle{
		Offset: b.offset,
		Length: uint64(n),
	}
	b.offset += uint64(n)

	// Footer.
	footer := Footer{
		MetaindexHandle: metaIndexHandle,
		IndexHandle:     indexHandle,
	}
	var footerBuf [FooterSize]byte
	footer.EncodeTo(footerBuf[:])

	if _, err := b.writer.Write(footerBuf[:]); err != nil {
		return err
	}

	if err := b.writer.Flush(); err != nil {
		return err
	}

	b.closed = true
	return b.file.Close()
}

func findShortestSeparator(a, b []byte) []byte {
	return a
}

func encodeBlockHandle(offset, size uint64) []byte {
	buf := make([]byte, 16)
	binary.BigEndian.PutUint64(buf[0:], offset)
	binary.BigEndian.PutUint64(buf[8:], size)
	return buf
}
