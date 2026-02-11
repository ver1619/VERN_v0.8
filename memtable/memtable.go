package memtable

import (
	"sync"
)

//
// Internal representation
//

//
// Public API
//

// Memtable stores data in memory.
type Memtable struct {
	mu  sync.RWMutex
	skl *Skiplist
}

// Entry represents a key-value pair.
type Entry struct {
	Key   []byte
	Value []byte
}

//
// Construction
//

// New creates a Memtable.
func New() *Memtable {
	return &Memtable{
		skl: NewSkiplist(),
	}
}

//
// Write path
//

// Insert adds a record.
func (m *Memtable) Insert(key []byte, value []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.skl.Insert(key, value)
}

//
// Read-only access (for iterators & recovery)
//

func (m *Memtable) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.skl.Size()
}

func (m *Memtable) ApproximateSize() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.skl.ApproximateSize()
}

// Iterator returns a Memtable iterator.
func (m *Memtable) Iterator() *Iterator {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.skl.NewIterator()
}

//
// Utilities
//
