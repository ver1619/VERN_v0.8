package sstable

import (
	"vern_kv0.8/internal/cache"
)

// NewIterator creates table iterator.
func NewIterator(path string, cache cache.Cache) (*TableIterator, error) {

	r, err := NewReader(path, cache)
	if err != nil {
		return nil, err
	}

	return r.NewIterator()
}

// Close closes the underlying Reader file handle.
func (it *TableIterator) Close() error {
	if it.reader != nil {
		return it.reader.Close()
	}
	return nil
}
