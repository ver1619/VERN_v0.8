package engine

import (
	"fmt"
	"path/filepath"

	"vern_kv0.8/iterators"
	"vern_kv0.8/memtable"
	"vern_kv0.8/sstable"
)

// flushMemtable flushes a memtable to an SSTable on disk.
// Returns the metadata for the new table.
func (db *DB) flushMemtable(mt *memtable.Memtable, fileNum uint64) (SSTableMeta, error) {
	filename := filepath.Join(db.dir, fmt.Sprintf("%06d.sst", fileNum))

	builder, err := sstable.NewBuilder(filename)
	if err != nil {
		return SSTableMeta{}, err
	}

	it := iterators.NewMemtableIterator(mt)
	it.SeekToFirst()

	var smallest, largest []byte
	var count uint64
	// var smallestSeq, largestSeq uint64

	first := true

	for it.Valid() {
		key := it.Key()
		value := it.Value()

		if err := builder.Add(key, value); err != nil {
			return SSTableMeta{}, err
		}

		// Track metadata
		if first {
			smallest = make([]byte, len(key))
			copy(smallest, key)
			// Initialize seq range?
			// We can extract seq from internal key if needed, or rely on memtable bounds if we tracked them.
			// For now, let's just use what we have.
			first = false
		}

		largest = make([]byte, len(key))
		copy(largest, key)

		// In a real implementation, we would extract the sequence number here to populate SmallestSeq/LargestSeq
		// but since we aren't rigorously using them for overlap checks yet, we can skip or extract if easy.
		// internal.ExtractTrailer(key) -> seq, type
		// leaving seq tracking for later if not strictly required for v0.8 functionality.

		count++
		it.Next()
	}

	if err := builder.Close(); err != nil {
		return SSTableMeta{}, err
	}

	// For v0.8, we dump everything to Level 0
	return SSTableMeta{
		FileNum:     fileNum,
		Level:       0,
		SmallestKey: smallest,
		LargestKey:  largest,
		// SmallestSeq: ...
		// LargestSeq: ...
	}, nil
}
