package memtable

import (
	"sort"

	"vern_kv0.5/internal"
)

//
// Internal representation
//

// entry represents one memtable entry.
// It is intentionally unexported.
type entry struct {
	key   []byte
	value []byte
}

//
// Public API
//

// Memtable is an in-memory sorted structure of InternalKey -> value.
//
// Invariants:
// - Entries are sorted strictly by internal.Comparator
// - Insert-only (no mutation, no deletion)
// - Tombstones are stored as values (interpreted later)
type Memtable struct {
	cmp     internal.Comparator
	entries []entry
}

// Entry is a read-only view of a memtable entry.
// Safe to expose to internal iterators.
type Entry struct {
	Key   []byte
	Value []byte
}

//
// Construction
//

// New creates an empty memtable.
func New() *Memtable {
	return &Memtable{
		cmp:     internal.Comparator{},
		entries: make([]entry, 0),
	}
}

//
// Write path
//

// Insert inserts an InternalKey -> value into the memtable.
// The key MUST already be an encoded InternalKey.
//
// Ordering is preserved using binary search.
func (m *Memtable) Insert(key []byte, value []byte) {
	// Find insertion point
	i := sort.Search(len(m.entries), func(i int) bool {
		return m.cmp.Compare(m.entries[i].key, key) >= 0
	})

	// Make room
	m.entries = append(m.entries, entry{})
	copy(m.entries[i+1:], m.entries[i:])

	// Insert (defensive copies)
	m.entries[i] = entry{
		key:   clone(key),
		value: clone(value),
	}
}

//
// Read-only access (for iterators & recovery)
//

// Size returns the number of entries in the memtable.
func (m *Memtable) Size() int {
	return len(m.entries)
}

// Entry returns the i-th entry in sorted order.
// Caller MUST ensure: 0 <= i < Size().
func (m *Memtable) Entry(i int) Entry {
	e := m.entries[i]
	return Entry{
		Key:   e.key,
		Value: e.value,
	}
}

//
// Utilities
//

// clone defensively copies a byte slice.
// Prevents external mutation of memtable state.
func clone(b []byte) []byte {
	if b == nil {
		return nil
	}
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
