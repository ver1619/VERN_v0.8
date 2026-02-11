package engine

import (
	"vern_kv0.8/internal"
	"vern_kv0.8/iterators"
)

// Iterator is a user-facing iterator.
type Iterator interface {
	SeekToFirst()
	Next()
	Valid() bool
	Key() []byte
	Value() []byte
}

type dbIterator struct {
	inner iterators.InternalIterator
}

func (it *dbIterator) SeekToFirst() {
	it.inner.SeekToFirst()
	it.skipTombstones()
}

func (it *dbIterator) Next() {
	it.inner.Next()
	it.skipTombstones()
}

func (it *dbIterator) skipTombstones() {
	for it.inner.Valid() {
		_, typ, _ := internal.ExtractTrailer(it.inner.Key())
		if typ == internal.RecordTypeTombstone {
			it.inner.Next()
			continue
		}
		break
	}
}

func (it *dbIterator) Valid() bool {
	return it.inner.Valid()
}

// Key returns user key.
func (it *dbIterator) Key() []byte {
	return internal.ExtractUserKey(it.inner.Key())
}

// Value returns the associated value.
func (it *dbIterator) Value() []byte {
	return it.inner.Value()
}
