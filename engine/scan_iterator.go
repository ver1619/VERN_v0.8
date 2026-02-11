package engine

import "bytes"

// scanIterator filters keys by range.
type scanIterator struct {
	inner  Iterator
	start  []byte
	end    []byte
	prefix []byte
	stop   bool
}

func (it *scanIterator) SeekToFirst() {
	it.stop = false
	it.inner.SeekToFirst()
	it.advance()
}

// Next advances to the next key in range.
func (it *scanIterator) Next() {
	it.inner.Next()
	it.advance()
}

func (it *scanIterator) Valid() bool {
	return !it.stop && it.inner.Valid()
}

func (it *scanIterator) Key() []byte {
	return it.inner.Key()
}

func (it *scanIterator) Value() []byte {
	return it.inner.Value()
}

func (it *scanIterator) advance() {
	for it.inner.Valid() {
		k := it.inner.Key()

		if it.prefix != nil {
			if !bytes.HasPrefix(k, it.prefix) {
				it.inner.Next()
				continue
			}
		}

		if it.start != nil && bytes.Compare(k, it.start) < 0 {
			it.inner.Next()
			continue
		}

		if it.end != nil && bytes.Compare(k, it.end) >= 0 {
			// end bound reached — stop iteration
			it.stop = true
			return
		}

		return
	}
}
