package cache

import (
	"container/list"
	"sync"
)

// LRUCache implements a simple LRU cache.
type LRUCache struct {
	mu       sync.Mutex
	capacity int
	usage    int
	list     *list.List
	items    map[string]*list.Element
}

type entry struct {
	key   string
	value []byte
	size  int
}

// NewLRUCache creates a new LRU cache with the given capacity in bytes.
func NewLRUCache(capacity int) *LRUCache {
	return &LRUCache{
		capacity: capacity,
		list:     list.New(),
		items:    make(map[string]*list.Element),
	}
}

func (c *LRUCache) Get(key string) []byte {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.list.MoveToFront(elem)
		return elem.Value.(*entry).value
	}
	return nil
}

func (c *LRUCache) Put(key string, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If exists, update
	if elem, ok := c.items[key]; ok {
		c.list.MoveToFront(elem)
		ent := elem.Value.(*entry)
		c.usage -= ent.size
		ent.value = value
		ent.size = len(value)
		c.usage += ent.size
		c.evict()
		return
	}

	// Insert
	size := len(value)
	ent := &entry{key: key, value: value, size: size}
	elem := c.list.PushFront(ent)
	c.items[key] = elem
	c.usage += size

	c.evict()
}

func (c *LRUCache) evict() {
	for c.usage > c.capacity && c.list.Len() > 0 {
		elem := c.list.Back()
		ent := elem.Value.(*entry)
		c.list.Remove(elem)
		delete(c.items, ent.key)
		c.usage -= ent.size
	}
}
