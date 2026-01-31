package sstable

import (
	"bytes"
	"errors"
	"os"
)

// Reader reads an SSTable.
type Reader struct {
	file *os.File
	size int64

	filterPolicy FilterPolicy
	filterData   []byte
}

func NewReader(path string) (*Reader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	r := &Reader{
		file:         f,
		size:         info.Size(),
		filterPolicy: NewBloomFilter(10), // Expect compatible filter
	}

	if err := r.loadFilter(); err != nil {
		// If filter loading fails, just ignore it?
		// For robustness, maybe log but proceed.
		// For now, let's just proceed without filter if error, or fail?
		// LevelDB ignores errors here.
		// We'll proceed with nil filterData.
	}

	return r, nil
}

func (r *Reader) loadFilter() error {
	footer, err := r.ReadFooter()
	if err != nil {
		return err
	}

	// Read MetaIndex
	metaIndexBlock, err := r.ReadBlock(footer.MetaindexHandle)
	if err != nil {
		return err
	}

	// Seek for filter
	filterName := "vern.filter.bloom"
	metaIndexBlock.Seek([]byte(filterName))

	if metaIndexBlock.Valid() && string(metaIndexBlock.Key()) == filterName {
		filterHandle := DecodeBlockHandle(metaIndexBlock.Value())

		// Read the raw filter data
		data := make([]byte, filterHandle.Length)
		n, err := r.file.ReadAt(data, int64(filterHandle.Offset))
		if err != nil {
			return err
		}
		if uint64(n) != filterHandle.Length {
			return errors.New("incomplete filter read")
		}

		r.filterData = data
	}

	return nil
}

func (r *Reader) Close() error {
	return r.file.Close()
}

func (r *Reader) MayContain(key []byte) bool {
	if r.filterData == nil || r.filterPolicy == nil {
		return true // Assume yes if no filter
	}
	return r.filterPolicy.KeyMayMatch(key, r.filterData)
}

func (r *Reader) ReadBlock(handle BlockHandle) (*BlockIterator, error) {
	data := make([]byte, handle.Length)
	n, err := r.file.ReadAt(data, int64(handle.Offset))
	if err != nil {
		return nil, err
	}
	if uint64(n) != handle.Length {
		return nil, errors.New("incomplete block read")
	}
	return NewBlockIterator(data), nil
}

func (r *Reader) ReadFooter() (Footer, error) {
	if r.size < int64(FooterSize) {
		return Footer{}, ErrCorruptSSTable
	}

	data := make([]byte, FooterSize)
	_, err := r.file.ReadAt(data, r.size-int64(FooterSize))
	if err != nil {
		return Footer{}, err
	}

	return DecodeFooter(data)
}

func (r *Reader) NewIterator() (*TableIterator, error) {
	footer, err := r.ReadFooter()
	if err != nil {
		return nil, err
	}

	indexBlock, err := r.ReadBlock(footer.IndexHandle)
	if err != nil {
		return nil, err
	}

	return &TableIterator{
		reader: r,
		index:  indexBlock,
	}, nil
}

// TableIterator iterates over an SSTable.
type TableIterator struct {
	reader *Reader
	index  *BlockIterator
	data   *BlockIterator

	valid bool
	err   error
}

func (it *TableIterator) Valid() bool {
	return it.valid && it.err == nil
}

func (it *TableIterator) Key() []byte {
	return it.data.Key()
}

func (it *TableIterator) Value() []byte {
	return it.data.Value()
}

func (it *TableIterator) SeekToFirst() {
	it.index.SeekToFirst()
	it.loadDataBlock()
	if it.data != nil {
		it.data.SeekToFirst()
		it.valid = it.data.Valid()
	} else {
		it.valid = false
	}
}

func (it *TableIterator) Seek(key []byte) {
	it.index.Seek(key)
	it.loadDataBlock()
	if it.data != nil {
		it.data.Seek(key)
		it.valid = it.data.Valid()
	} else {
		it.valid = false
	}
}

func (it *TableIterator) Next() {
	if it.data == nil {
		return
	}
	it.data.Next()
	if !it.data.Valid() {
		// End of data block, move to next
		it.index.Next()
		it.loadDataBlock()
		if it.data != nil {
			it.data.SeekToFirst()
			it.valid = it.data.Valid()
		} else {
			it.valid = false
		}
	} else {
		it.valid = true
	}
}

func (it *TableIterator) loadDataBlock() {
	if !it.index.Valid() {
		it.data = nil
		it.valid = false
		return
	}

	// Decode block handle from index value
	handle := DecodeBlockHandle(it.index.Value())

	block, err := it.reader.ReadBlock(handle)
	if err != nil {
		it.err = err
		it.valid = false
		return
	}
	it.data = block
}

// Helper to compare keys
func compare(a, b []byte) int {
	return bytes.Compare(a, b)
}
