package cache

// Cache interface.
type Cache interface {
	Get(key string) []byte

	Put(key string, value []byte)
}
