package iterators

// InternalIterator defines the interface for iterating over DB keys.
type InternalIterator interface {
	SeekToFirst()
	Next()
	Valid() bool
	Key() []byte
	Value() []byte
}
