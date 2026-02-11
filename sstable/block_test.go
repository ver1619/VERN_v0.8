package sstable

import (
	"fmt"
	"testing"
)

func TestBlockBuilder(t *testing.T) {
	b := NewBlockBuilder()

	b.Add([]byte("key1"), []byte("value1"))
	b.Add([]byte("key2"), []byte("value2"))
	b.Add([]byte("key3"), []byte("value3"))

	content := b.Finish()

	it := NewBlockIterator(content)

	// Test SeekToFirst
	it.SeekToFirst()
	if !it.Valid() {
		t.Fatalf("Iterator invalid after SeekToFirst")
	}
	if string(it.Key()) != "key1" {
		t.Errorf("Expected key1, got %s", it.Key())
	}
	if string(it.Value()) != "value1" {
		t.Errorf("Expected value1, got %s", it.Value())
	}

	// Test Next
	it.Next()
	if !it.Valid() {
		t.Fatalf("Iterator invalid after Next")
	}
	if string(it.Key()) != "key2" {
		t.Errorf("Expected key2, got %s", it.Key())
	}

	it.Next()
	if string(it.Key()) != "key3" {
		t.Errorf("Expected key3, got %s", it.Key())
	}

	it.Next()
	if it.Valid() {
		t.Errorf("Iterator should be invalid after last element")
	}
}

func TestBlockSeek(t *testing.T) {
	b := NewBlockBuilder()

	// Trigger multiple restart points.
	for i := 0; i < 100; i++ {
		key := []byte(fmt.Sprintf("key%03d", i))
		b.Add(key, []byte("val"))
	}

	content := b.Finish()
	it := NewBlockIterator(content)

	tests := []struct {
		target string
		want   string
	}{
		{"key000", "key000"},
		{"key050", "key050"},
		{"key099", "key099"},
		{"key010", "key010"},
		{"key00", "key000"},   // Seek to before first
		{"key049a", "key050"}, // Seek to between keys
	}

	for _, tt := range tests {
		it.Seek([]byte(tt.target))
		if !it.Valid() {
			t.Errorf("Seek(%s) returned invalid iterator", tt.target)
			continue
		}
		if string(it.Key()) != tt.want {
			t.Errorf("Seek(%s) want %s, got %s", tt.target, tt.want, it.Key())
		}
	}

	// Seek past end
	it.Seek([]byte("key999"))
	if it.Valid() {
		t.Errorf("Seek past end should be invalid")
	}
}
