package iterators

import (
	"bytes"

	"vern_kv0.5/internal"
)

// MergeIterator merges multiple sorted InternalIterators into one.
type MergeIterator struct {
	iters []InternalIterator
	valid []bool

	currKey   []byte
	currValue []byte

	initialized bool
	cmp         internal.Comparator
}

// NewMergeIterator creates a merge iterator over child iterators.
func NewMergeIterator(children []InternalIterator) *MergeIterator {
	return &MergeIterator{
		iters: children,
		valid: make([]bool, len(children)),
		cmp:   internal.Comparator{},
	}
}

func (m *MergeIterator) SeekToFirst() {
	for i, it := range m.iters {
		it.SeekToFirst()
		m.valid[i] = it.Valid()
	}
	m.initialized = true
	m.advance()
}

func (m *MergeIterator) Next() {
	if !m.initialized {
		return
	}
	m.advance()
}

func (m *MergeIterator) Valid() bool {
	return m.currKey != nil
}

func (m *MergeIterator) Key() []byte {
	return m.currKey
}

func (m *MergeIterator) Value() []byte {
	return m.currValue
}

// advance selects the next visible InternalKey.
func (m *MergeIterator) advance() {
	best := -1

	// Step 1: pick smallest InternalKey
	for i, ok := range m.valid {
		if !ok {
			continue
		}
		if best == -1 {
			best = i
			continue
		}
		if m.cmp.Compare(m.iters[i].Key(), m.iters[best].Key()) < 0 {
			best = i
		}
	}

	if best == -1 {
		m.currKey = nil
		m.currValue = nil
		return
	}

	m.currKey = m.iters[best].Key()
	m.currValue = m.iters[best].Value()

	// Step 2: advance all iterators pointing to same InternalKey
	for i, it := range m.iters {
		if m.valid[i] && bytes.Equal(it.Key(), m.currKey) {
			it.Next()
			m.valid[i] = it.Valid()
		}
	}

	// Step 3: version collapse (skip older versions of same user key)
	userKey := internal.ExtractUserKey(m.currKey)
	for i, it := range m.iters {
		for m.valid[i] {
			if !bytes.Equal(internal.ExtractUserKey(it.Key()), userKey) {
				break
			}
			it.Next()
			m.valid[i] = it.Valid()
		}
	}
}
