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

// Generate random level.
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

	// Find position.
	for i := s.level - 1; i >= 0; i-- {
		for current.next[i] != nil && s.cmp.Compare(current.next[i].key, key) < 0 {
			current = current.next[i]
		}
		update[i] = current
	}

	current = current.next[0]

	if current != nil && s.cmp.Compare(current.key, key) == 0 {
		// Update existing.
		s.sizeBytes -= (len(current.value) - len(value))
		current.value = value
		return
	}

	// Insert new.
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
	list *Skiplist
	node *Node
}

// NewIterator creates an iterator.
func (s *Skiplist) NewIterator() *Iterator {
	return &Iterator{
		list: s,
		node: nil,
	}
}

// Reset to first element.
func (it *Iterator) SeekToFirst() {
	it.node = it.list.head.next[0]
}

// Advance to first node >= target.
func (it *Iterator) Seek(target []byte) {
	current := it.list.head
	for i := it.list.level - 1; i >= 0; i-- {
		for current.next[i] != nil && it.list.cmp.Compare(current.next[i].key, target) < 0 {
			current = current.next[i]
		}
	}
	it.node = current.next[0]
}

// Move to next.
func (it *Iterator) Next() {
	if it.node != nil {
		it.node = it.node.next[0]
	}
}

// Is valid?
func (it *Iterator) Valid() bool {
	return it.node != nil
}

// Current key.
func (it *Iterator) Key() []byte {
	if it.node == nil {
		return nil
	}
	return it.node.key
}

// Current value.
func (it *Iterator) Value() []byte {
	if it.node == nil {
		return nil
	}
	return it.node.value
}
