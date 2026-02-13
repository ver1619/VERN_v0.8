package iterators

import "vern_kv0.8/memtable"

// MemtableIterator provides an iterator interface for the memtable.
type MemtableIterator struct {
	iter *memtable.Iterator
}

func NewMemtableIterator(mt *memtable.Memtable) *MemtableIterator {
	return &MemtableIterator{
		iter: mt.Iterator(),
	}
}

func (it *MemtableIterator) SeekToFirst() {
	it.iter.SeekToFirst()
}

func (it *MemtableIterator) Seek(key []byte) {
	it.iter.Seek(key)
}

func (it *MemtableIterator) Next() {
	it.iter.Next()
}

func (it *MemtableIterator) Valid() bool {
	return it.iter.Valid()
}

func (it *MemtableIterator) Key() []byte {
	return it.iter.Key()
}

func (it *MemtableIterator) Value() []byte {
	return it.iter.Value()
}
