package cache

import (
	"fmt"
	"sync"
	"testing"
)

func TestLRUCache_Basic(t *testing.T) {
	// Capacity: 10 bytes
	c := NewLRUCache(10)

	// Verify Put and Get operations.
	c.Put("a", []byte("1"))
	val := c.Get("a")
	if string(val) != "1" {
		t.Errorf("expected 1, got %s", val)
	}

	// Get non-existent
	if c.Get("b") != nil {
		t.Errorf("expected nil for non-existent key")
	}
}

func TestLRUCache_Overwrite(t *testing.T) {
	c := NewLRUCache(10)

	c.Put("a", []byte("1"))
	val := c.Get("a")
	if string(val) != "1" {
		t.Errorf("expected 1, got %s", val)
	}

	// Update existing key.
	c.Put("a", []byte("22"))
	val = c.Get("a")
	if string(val) != "22" {
		t.Errorf("expected 22, got %s", val)
	}
}

func TestLRUCache_Eviction(t *testing.T) {
	c := NewLRUCache(10)

	c.Put("k1", []byte("1234"))
	c.Put("k2", []byte("5678"))

	if c.Get("k1") == nil {
		t.Errorf("k1 should exist")
	}
	if c.Get("k2") == nil {
		t.Errorf("k2 should exist")
	}

	// Verify LRU eviction.
	c.Put("k3", []byte("abcd"))

	if c.Get("k1") != nil {
		t.Errorf("k1 should have been evicted")
	}
	if c.Get("k2") == nil {
		t.Errorf("k2 should remain")
	}
	if c.Get("k3") == nil {
		t.Errorf("k3 should remain")
	}
}

func TestLRUCache_LargeItem(t *testing.T) {
	// Capacity 10
	c := NewLRUCache(10)

	// Verify that items larger than capacity are evicted.
	c.Put("big", []byte("12345678901"))

	if c.Get("big") != nil {
		t.Errorf("item larger than capacity should be evicted (or not stored)")
	}
}

func TestLRUCache_Concurrency(t *testing.T) {
	c := NewLRUCache(1000)
	var wg sync.WaitGroup

	// 10 concurrent routines
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("k%d", id)
			val := []byte(fmt.Sprintf("v%d", id))

			// Check for race conditions under concurrent access.
			for j := 0; j < 100; j++ {
				c.Put(key, val)
				c.Get(key)
			}
		}(i)
	}
	wg.Wait()
}
