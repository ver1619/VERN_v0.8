package sstable

import (
	"os"
)

type Iterator struct {
	data   []byte
	offset int
	valid  bool

	key   []byte
	value []byte
}

func NewIterator(path string) (*Iterator, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return &Iterator{
		data:   data,
		offset: 0,
		valid:  false,
	}, nil
}

func (it *Iterator) SeekToFirst() {
	it.offset = 0
	it.advance()
}

func (it *Iterator) Next() {
	it.advance()
}

func (it *Iterator) Valid() bool {
	return it.valid
}

func (it *Iterator) Key() []byte {
	return it.key
}

func (it *Iterator) Value() []byte {
	return it.value
}

func (it *Iterator) advance() {
	rec, n, err := ReadRecord(it.data, it.offset)
	if err != nil {
		it.valid = false
		return
	}

	it.key = rec.Key
	it.value = rec.Value
	it.offset += n
	it.valid = true
}
