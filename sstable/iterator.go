package sstable

import (
	"vern_kv0.8/internal/cache"
)

// NewIterator returns a new iterator for the given SSTable file.
func NewIterator(path string, cache cache.Cache) (*TableIterator, error) {
	// Note: Ownership of the reader is returned to the iterator.
	// We might need a way to close the reader when the iterator is done.
	// For now, let's assume the iterator owns the reader.

	r, err := NewReader(path, cache)
	if err != nil {
		return nil, err
	}

	return r.NewIterator()
}
