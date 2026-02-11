package memtable

import (
	"math/rand"

	"vern_kv0.8/internal"
)

const (
	maxLevel    = 12
	probability = 0.5
)

type Node struct {
	key   []byte
	value []byte
	next  []*Node
}

// Skiplist is a probabilistic sorted list.
type Skiplist struct {
	head      *Node
	level     int
	count     int
	sizeBytes int
	cmp       internal.Comparator
}

func NewSkiplist() *Skiplist {
	return &Skiplist{
		head: &Node{
			next: make([]*Node, maxLevel),
		},
		level: 1,
		cmp:   internal.Comparator{},
	}
}

// randomLevel generates a level.
func randomLevel() int {
	lvl := 1
	for rand.Float64() < probability && lvl < maxLevel {
		lvl++
	}
	return lvl
}

func (s *Skiplist) Insert(key, value []byte) {
	update := make([]*Node, maxLevel)
	current := s.head

	for i := s.level - 1; i >= 0; i-- {
		for current.next[i] != nil && s.cmp.Compare(current.next[i].key, key) < 0 {
			current = current.next[i]
		}
		update[i] = current
	}

	current = current.next[0]

	if current != nil && s.cmp.Compare(current.key, key) == 0 {
		// Update existing
		s.sizeBytes -= (len(current.value) - len(value))
		current.value = value
		return
	}

	lvl := randomLevel()
	if lvl > s.level {
		for i := s.level; i < lvl; i++ {
			update[i] = s.head
		}
		s.level = lvl
	}

	newNode := &Node{
		key:   key,
		value: value,
		next:  make([]*Node, lvl),
	}

	for i := 0; i < lvl; i++ {
		newNode.next[i] = update[i].next[i]
		update[i].next[i] = newNode
	}

	s.count++
	s.sizeBytes += len(key) + len(value)
}

func (s *Skiplist) Size() int {
	return s.count
}

func (s *Skiplist) ApproximateSize() int {
	return s.sizeBytes
}

type iterEntry struct {
	key   []byte
	value []byte
}

// Iterator traverses the Skiplist.
type Iterator struct {
	entries []iterEntry
	pos     int
}

// NewIterator creates an iterator.
func (s *Skiplist) NewIterator() *Iterator {
	entries := make([]iterEntry, 0, s.count)
	node := s.head.next[0]
	for node != nil {
		entries = append(entries, iterEntry{key: node.key, value: node.value})
		node = node.next[0]
	}
	return &Iterator{
		entries: entries,
		pos:     -1,
	}
}

// SeekToFirst resets iterator.
func (it *Iterator) SeekToFirst() {
	if len(it.entries) > 0 {
		it.pos = 0
	} else {
		it.pos = -1
	}
}

// Next advances iterator.
func (it *Iterator) Next() {
	if it.pos >= 0 && it.pos < len(it.entries) {
		it.pos++
	}
}

// Valid checks if iterator is valid.
func (it *Iterator) Valid() bool {
	return it.pos >= 0 && it.pos < len(it.entries)
}

// Key returns the current key.
func (it *Iterator) Key() []byte {
	return it.entries[it.pos].key
}

// Value returns the current value.
func (it *Iterator) Value() []byte {
	return it.entries[it.pos].value
}
