package sstable

// NewIterator returns a new iterator for the given SSTable file.
func NewIterator(path string) (*TableIterator, error) {
	// Note: Ownership of the reader is returned to the iterator.
	// We might need a way to close the reader when the iterator is done.
	// For now, let's assume the iterator owns the reader.

	r, err := NewReader(path)
	if err != nil {
		return nil, err
	}

	return r.NewIterator()
}
