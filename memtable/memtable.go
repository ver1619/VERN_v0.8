package memtable

import (
	"sort"
	"sync"

	"vern_kv0.8/internal"
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
// Sorted (Insert-only) store.
type Memtable struct {
	mu        sync.RWMutex
	cmp       internal.Comparator
	entries   []entry
	sizeBytes int
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

// New returns a new Memtable.
func New() *Memtable {
	return &Memtable{
		cmp:     internal.Comparator{},
		entries: make([]entry, 0),
	}
}

//
// Write path
//

// Insert inserts an encoded InternalKey -> value.
// Thread-safe.
func (m *Memtable) Insert(key []byte, value []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

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

	m.sizeBytes += len(key) + len(value)
}

//
// Read-only access (for iterators & recovery)
//

// Size returns the number of entries in the memtable.
func (m *Memtable) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.entries)
}

// ApproximateSize returns the approximate memory usage in bytes.
func (m *Memtable) ApproximateSize() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sizeBytes
}

// Entry returns the i-th entry in sorted order.
// Caller MUST ensure: 0 <= i < Size().
func (m *Memtable) Entry(i int) Entry {
	m.mu.RLock()
	defer m.mu.RUnlock()
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
