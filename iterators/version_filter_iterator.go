package iterators

import "vern_kv0.8/internal"

// VersionFilterIterator filters InternalKeys based on snapshot visibility.
//
// It wraps another InternalIterator and hides versions with
// sequence numbers greater than readSeq.
type VersionFilterIterator struct {
	child   InternalIterator
	readSeq uint64

	valid bool
	key   []byte
	value []byte
}

// NewVersionFilterIterator creates a new filtering iterator.
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

// advance moves to the next visible version.
func (it *VersionFilterIterator) advance() {
	it.valid = false

	for it.child.Valid() {
		k := it.child.Key()
		seq, _, err := internal.ExtractTrailer(k)
		if err != nil {
			// Corruption: stop iteration safely
			return
		}

		if seq <= it.readSeq {
			it.key = k
			it.value = it.child.Value()
			it.valid = true
			it.child.Next()
			return
		}

		// Version too new — skip
		it.child.Next()
	}
}
