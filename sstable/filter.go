package sstable

import (
	"hash/crc32" // Using crc32 as a simple hash function for now, usually murmur3 is preferred but standard lib is safer without deps
)

// FilterPolicy interface.
type FilterPolicy interface {
	Name() string
	CreateFilter(keys [][]byte) []byte
	KeyMayMatch(key, filter []byte) bool
}

// BloomFilter implementation.
type BloomFilter struct {
	bitsPerKey int
}

func NewBloomFilter(bitsPerKey int) *BloomFilter {
	return &BloomFilter{
		bitsPerKey: bitsPerKey,
	}
}

func (f *BloomFilter) Name() string {
	return "vern.filter.bloom"
}

func (f *BloomFilter) CreateFilter(keys [][]byte) []byte {
	n := len(keys)
	bits := n * f.bitsPerKey
	if bits < 64 {
		bits = 64
	}
	bytes := (bits + 7) / 8
	bits = bytes * 8

	filter := make([]byte, bytes+1)

	// Calculate k
	k := int(float64(f.bitsPerKey) * 0.69)
	if k < 1 {
		k = 1
	}
	if k > 30 {
		k = 30
	}
	filter[bytes] = uint8(k)

	for _, key := range keys {
		h := hash(key)
		delta := (h >> 17) | (h << 15) // Rotate right 17 bits
		for i := 0; i < k; i++ {
			bitPos := h % uint32(bits)
			filter[bitPos/8] |= (1 << (bitPos % 8))
			h += delta
		}
	}

	return filter
}

func (f *BloomFilter) KeyMayMatch(key, filter []byte) bool {
	n := len(filter)
	if n < 2 {
		return false
	}

	bits := uint32((n - 1) * 8)
	k := int(filter[n-1])
	if k > 30 {
		// Reserved/invalid
		return true
	}

	h := hash(key)
	delta := (h >> 17) | (h << 15)
	for i := 0; i < k; i++ {
		bitPos := h % bits
		if (filter[bitPos/8] & (1 << (bitPos % 8))) == 0 {
			return false
		}
		h += delta
	}
	return true
}

func hash(b []byte) uint32 {
	return crc32.ChecksumIEEE(b) // Simple hash
}
