package engine

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"vern_kv0.8/internal"
	"vern_kv0.8/iterators"
	"vern_kv0.8/memtable"
	"vern_kv0.8/sstable"
)

// flushMemtable flushes memtable to disk.
func (db *DB) flushMemtable(mt *memtable.Memtable, fileNum uint64) (SSTableMeta, error) {
	filename := filepath.Join(db.dir, fmt.Sprintf("%06d.sst", fileNum))

	builder, err := sstable.NewBuilder(filename)
	if err != nil {
		return SSTableMeta{}, err
	}

	it := iterators.NewMemtableIterator(mt)
	it.SeekToFirst()

	var smallest, largest []byte
	var smallestSeq uint64 = math.MaxUint64
	var largestSeq uint64
	var count uint64

	first := true

	for it.Valid() {
		key := it.Key()
		value := it.Value()

		if err := builder.Add(key, value); err != nil {
			return SSTableMeta{}, err
		}

		// Update metadata.
		if first {
			smallest = make([]byte, len(key))
			copy(smallest, key)
			first = false
		}

		largest = make([]byte, len(key))
		copy(largest, key)

		// Update sequence bounds.
		seq, _, _ := internal.ExtractTrailer(key)
		if seq < smallestSeq {
			smallestSeq = seq
		}
		if seq > largestSeq {
			largestSeq = seq
		}

		count++
		it.Next()
	}

	if err := builder.Close(); err != nil {
		return SSTableMeta{}, err
	}

	// Determine file size.
	var fileSize int64
	if info, err := os.Stat(filename); err == nil {
		fileSize = info.Size()
	}

	if count == 0 {
		smallestSeq = 0
	}

	// Return L0 metadata.
	return SSTableMeta{
		FileNum:     fileNum,
		Level:       0,
		SmallestKey: smallest,
		LargestKey:  largest,
		SmallestSeq: smallestSeq,
		LargestSeq:  largestSeq,
		FileSize:    fileSize,
	}, nil
}
