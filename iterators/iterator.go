package iterators

// InternalIterator interface.
type InternalIterator interface {
	SeekToFirst()
	Next()
	Valid() bool
	Key() []byte
	Value() []byte
}
