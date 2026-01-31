package iterators

import "vern_kv0.5/memtable"

// MemtableIterator iterates over a memtable in sorted order.
type MemtableIterator struct {
	mt    *memtable.Memtable
	index int
}

func NewMemtableIterator(mt *memtable.Memtable) *MemtableIterator {
	return &MemtableIterator{
		mt:    mt,
		index: -1,
	}
}

func (it *MemtableIterator) SeekToFirst() {
	it.index = 0
}

func (it *MemtableIterator) Next() {
	it.index++
}

func (it *MemtableIterator) Valid() bool {
	return it.index >= 0 && it.index < it.mt.Size()
}

func (it *MemtableIterator) Key() []byte {
	return it.mt.Entry(it.index).Key
}

func (it *MemtableIterator) Value() []byte {
	return it.mt.Entry(it.index).Value
}
