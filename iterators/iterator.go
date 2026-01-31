package iterators

// InternalIterator defines the common iterator interface.
type InternalIterator interface {
	SeekToFirst()
	Next()
	Valid() bool
	Key() []byte
	Value() []byte
}
