package iterators

import (
	"bytes"

	"vern_kv0.8/internal"
)

// MergeIterator merges multiple sorted iterators into one.
type MergeIterator struct {
	iters []InternalIterator
	valid []bool

	currKey   []byte
	currValue []byte

	initialized bool
	cmp         internal.Comparator
	Deduplicate bool // If true, skips older versions of same key.
}

// NewMergeIterator initializes a merging iterator.
func NewMergeIterator(children []InternalIterator, deduplicate bool) *MergeIterator {
	return &MergeIterator{
		iters:       children,
		valid:       make([]bool, len(children)),
		cmp:         internal.Comparator{},
		Deduplicate: deduplicate,
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

// advance selects the next smallest key.
func (m *MergeIterator) advance() {
	best := -1

	// Find the smallest key across iterators.
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

	// Skip duplicate internal keys.
	for i, it := range m.iters {
		if m.valid[i] && bytes.Equal(it.Key(), m.currKey) {
			it.Next()
			m.valid[i] = it.Valid()
		}
	}

	// Advance past older versions of the same user key ONLY if Deduplicate is true.
	if m.Deduplicate {
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
}
