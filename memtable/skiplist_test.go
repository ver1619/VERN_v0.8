package memtable

import (
	"fmt"
	"testing"

	"vern_kv0.8/internal"
)

// ikey encodes an internal key for testing.
func ikey(user string, seq uint64) []byte {
	return internal.EncodeInternalKey([]byte(user), seq, internal.RecordTypeValue)
}

func TestSkiplistInsertAndIterator(t *testing.T) {
	sl := NewSkiplist()

	sl.Insert(ikey("a", 1), []byte("v1"))
	sl.Insert(ikey("b", 1), []byte("v2"))
	sl.Insert(ikey("c", 1), []byte("v3"))

	if sl.count != 3 {
		t.Fatalf("expected count 3, got %d", sl.count)
	}

	it := sl.NewIterator()
	it.SeekToFirst()

	expected := []struct {
		key string
		val string
	}{
		{"a", "v1"},
		{"b", "v2"},
		{"c", "v3"},
	}

	for _, exp := range expected {
		if !it.Valid() {
			t.Fatalf("iterator ended early, expected key %s", exp.key)
		}
		userKey := string(internal.ExtractUserKey(it.Key()))
		if userKey != exp.key {
			t.Errorf("expected key %s, got %s", exp.key, userKey)
		}
		if string(it.Value()) != exp.val {
			t.Errorf("expected value %s, got %s", exp.val, it.Value())
		}
		it.Next()
	}

	if it.Valid() {
		t.Error("expected iterator to be exhausted")
	}
}

func TestSkiplistInsertDuplicateUpdates(t *testing.T) {
	sl := NewSkiplist()

	key := ikey("x", 1)
	sl.Insert(key, []byte("old"))
	sl.Insert(key, []byte("new"))

	// Same internal key updates existing node.
	if sl.count != 1 {
		t.Fatalf("expected count 1 after update, got %d", sl.count)
	}

	it := sl.NewIterator()
	it.SeekToFirst()
	if !it.Valid() {
		t.Fatal("iterator invalid")
	}
	if string(it.Value()) != "new" {
		t.Errorf("expected updated value 'new', got %s", it.Value())
	}
}

func TestSkiplistOrdering(t *testing.T) {
	sl := NewSkiplist()

	// Insert out of order.
	sl.Insert(ikey("d", 1), []byte("4"))
	sl.Insert(ikey("b", 1), []byte("2"))
	sl.Insert(ikey("a", 1), []byte("1"))
	sl.Insert(ikey("c", 1), []byte("3"))

	it := sl.NewIterator()
	it.SeekToFirst()

	order := []string{"a", "b", "c", "d"}
	for _, want := range order {
		if !it.Valid() {
			t.Fatalf("iterator ended early, expected key %s", want)
		}
		got := string(internal.ExtractUserKey(it.Key()))
		if got != want {
			t.Errorf("expected %s, got %s", want, got)
		}
		it.Next()
	}
}

func TestSkiplistIteratorEmpty(t *testing.T) {
	sl := NewSkiplist()

	it := sl.NewIterator()
	it.SeekToFirst()

	if it.Valid() {
		t.Error("expected iterator on empty skiplist to be invalid")
	}
}

func TestSkiplistIteratorSnapshot(t *testing.T) {
	sl := NewSkiplist()
	sl.Insert(ikey("a", 1), []byte("1"))
	sl.Insert(ikey("b", 1), []byte("2"))

	// Take an iterator snapshot.
	it := sl.NewIterator()

	// Mutate the skiplist after iterator creation.
	sl.Insert(ikey("c", 1), []byte("3"))

	// The iterator should see the new entry (Live Iterator).
	count := 0
	it.SeekToFirst()
	for it.Valid() {
		count++
		it.Next()
	}

	if count != 3 {
		t.Fatalf("live iterator should see 3 entries, got %d", count)
	}
}

func TestSkiplistLargeInsert(t *testing.T) {
	sl := NewSkiplist()
	n := 1000 // Test with 1000 entries.

	for i := 0; i < n; i++ {
		k := ikey(fmt.Sprintf("key%04d", i), 1)
		sl.Insert(k, []byte(fmt.Sprintf("val%04d", i)))
	}

	if sl.count != n {
		t.Fatalf("expected count %d, got %d", n, sl.count)
	}

	// Verify sorted order via iterator.
	it := sl.NewIterator()
	it.SeekToFirst()

	prev := ""
	seen := 0
	for it.Valid() {
		cur := string(internal.ExtractUserKey(it.Key()))
		if cur <= prev && prev != "" {
			t.Fatalf("sort violation: %s after %s", cur, prev)
		}
		prev = cur
		seen++
		it.Next()
	}

	if seen != n {
		t.Fatalf("expected %d entries, iterated %d", n, seen)
	}
}

func TestRandomLevel(t *testing.T) {
	for i := 0; i < 10000; i++ {
		lvl := randomLevel()
		if lvl < 1 || lvl > maxLevel {
			t.Fatalf("randomLevel() = %d, want [1, %d]", lvl, maxLevel)
		}
	}
}
