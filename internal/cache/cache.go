package cache

// Cache is a thread-safe key-value cache.
type Cache interface {
	// Get returns the value for key, or nil if not found.
	Get(key string) []byte

	// Put inserts a key-value pair.
	Put(key string, value []byte)
}
