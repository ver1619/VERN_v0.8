package iterators

import "vern_kv0.8/internal"

// VersionFilterIterator hides records with sequence numbers higher than readSeq.
type VersionFilterIterator struct {
	child   InternalIterator
	readSeq uint64

	valid bool
	key   []byte
	value []byte
}

// NewVersionFilterIterator initializes a version filter.
func NewVersionFilterIterator(
	child InternalIterator,
	readSeq uint64,
) *VersionFilterIterator {
	return &VersionFilterIterator{
		child:   child,
		readSeq: readSeq,
	}
}

func (it *VersionFilterIterator) SeekToFirst() {
	it.child.SeekToFirst()
	it.advance()
}

func (it *VersionFilterIterator) Next() {
	it.advance()
}

func (it *VersionFilterIterator) Valid() bool {
	return it.valid
}

func (it *VersionFilterIterator) Key() []byte {
	return it.key
}

func (it *VersionFilterIterator) Value() []byte {
	return it.value
}

// advance skips records not visible at the current sequence.
func (it *VersionFilterIterator) advance() {
	it.valid = false

	for it.child.Valid() {
		k := it.child.Key()
		seq, _, err := internal.ExtractTrailer(k)
		if err != nil {
			// Terminate on key corruption.
			return
		}

		if seq <= it.readSeq {
			it.key = k
			it.value = it.child.Value()
			it.valid = true
			it.child.Next()
			return
		}

		// Skip newer versions.
		it.child.Next()
	}
}
