package memtable

import (
	"bytes"
	"sync"
)

//
// Internal representation
//

//
// Public API
//

// Memtable is an in-memory state
type Memtable struct {
	mu       sync.RWMutex
	skiplist *Skiplist
	size     int64
}

// Entry represents a key-value pair.
type Entry struct {
	Key   []byte
	Value []byte
}

//
// Construction
//

// New creates a fresh Memtable.
func New() *Memtable {
	return &Memtable{
		skiplist: NewSkiplist(),
		size:     0,
	}
}

//
// Write path
//

// Insert adds a key-value pair.
func (m *Memtable) Insert(key []byte, value []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Improve approximation logic if needed.
	estimatedSize := int64(len(key) + len(value) + 16) // Node overhead.
	m.size += estimatedSize
	m.skiplist.Insert(key, value)
}

//
// Read-only access (for iterators & recovery)
//

// Get looks up a key.
func (m *Memtable) Get(key []byte) ([]byte, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	it := m.skiplist.NewIterator()
	it.Seek(key)
	if it.Valid() && bytes.Equal(it.Key(), key) {
		return it.Value(), true
	}
	return nil, false
}

// Size returns the number of entries.
func (m *Memtable) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.skiplist.Size()
}

// ApproximateSize returns estimated memory usage.
func (m *Memtable) ApproximateSize() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return int(m.size)
}

// Iterator returns an iterator over the memtable.
func (m *Memtable) Iterator() *Iterator {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.skiplist.NewIterator()
}
