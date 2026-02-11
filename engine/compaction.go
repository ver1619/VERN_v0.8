package engine

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"vern_kv0.8/internal"
	"vern_kv0.8/iterators"
	"vern_kv0.8/manifest"
	"vern_kv0.8/sstable"
)

// PickCompaction picks a compaction.
func (db *DB) PickCompaction() (int, bool) {
	return db.version.PickCompaction(db.opts.L0CompactionTrigger, db.opts.L1MaxBytes)
}

// CompactLevel executes compaction logic.
func (db *DB) CompactLevel(level int) error {
	if level >= NumLevels-1 {
		return fmt.Errorf("cannot compact max level")
	}

	// Select inputs while holding lock.
	db.mu.Lock()
	var inputs []SSTableMeta

	if level == 0 {
		// L0 compaction strategy.
		l0 := db.version.Levels[0]
		if len(l0) == 0 {
			db.mu.Unlock()
			return nil
		}
		l1 := db.version.Levels[1]

		inputs = append(inputs, l0...)
		inputs = append(inputs, l1...)
	} else {
		// L1+ compaction strategy.
		currentLevel := db.version.Levels[level]
		if len(currentLevel) == 0 {
			db.mu.Unlock()
			return nil
		}
		picked := currentLevel[0]
		inputs = append(inputs, picked)

		// Include overlapping files from next level.
		overlaps := db.version.GetOverlappingInputs(level+1, picked.SmallestKey, picked.LargestKey)
		inputs = append(inputs, overlaps...)
	}

	fileNum := db.nextFileNum
	db.nextFileNum++
	db.mu.Unlock()

	// Create iterators.
	var iters []iterators.InternalIterator
	for _, meta := range inputs {
		path := filepath.Join(db.dir, fmt.Sprintf("%06d.sst", meta.FileNum))
		sstIt, err := sstable.NewIterator(path, db.cache)
		if err != nil {
			return err
		}
		iters = append(iters, sstIt)
	}

	merge := iterators.NewMergeIterator(iters)
	merge.SeekToFirst()

	// Target level
	targetLevel := level + 1
	if level == 0 {
		targetLevel = 1
	}

	filename := filepath.Join(db.dir, fmt.Sprintf("%06d.sst", fileNum))

	builder, err := sstable.NewBuilder(filename)
	if err != nil {
		return err
	}

	var smallest, largest []byte
	var smallestSeq uint64 = math.MaxUint64
	var largestSeq uint64
	first := true
	var count uint64

	for merge.Valid() {
		key := merge.Key()
		val := merge.Value()

		_, typ, _ := internal.ExtractTrailer(key)

		// Garbage collect tombstones.
		if typ == internal.RecordTypeTombstone {
			isBottom := true
			for l := targetLevel + 1; l < NumLevels; l++ {
				if len(db.version.Levels[l]) > 0 {
					// Skip overlap check.
					isBottom = false
					break
				}
			}
			if isBottom {
				merge.Next()
				continue
			}
		}

		if err := builder.Add(key, val); err != nil {
			return err
		}

		if first {
			smallest = make([]byte, len(key))
			copy(smallest, key)
			first = false
		}
		largest = make([]byte, len(key))
		copy(largest, key)

		// Track seq bounds
		seq, _, _ := internal.ExtractTrailer(key)
		if seq < smallestSeq {
			smallestSeq = seq
		}
		if seq > largestSeq {
			largestSeq = seq
		}

		count++
		merge.Next()
	}

	if count == 0 {
		smallestSeq = 0
	}

	if err := builder.Close(); err != nil {
		return err
	}

	// Get actual file size
	var fileSize int64
	if info, err := os.Stat(filename); err == nil {
		fileSize = info.Size()
	}

	// Update version set.
	db.mu.Lock()
	defer db.mu.Unlock()

	// Remove inputs, add output
	newMeta := SSTableMeta{
		FileNum:     fileNum,
		Level:       uint32(targetLevel),
		SmallestKey: smallest,
		LargestKey:  largest,
		SmallestSeq: smallestSeq,
		LargestSeq:  largestSeq,
		FileSize:    fileSize,
	}

	// Record deletions.
	for _, in := range inputs {
		edit := manifest.Record{
			Type: manifest.RecordTypeRemoveSSTable,
			Data: manifest.RemoveSSTable{FileNum: in.FileNum},
		}
		if err := db.manifest.Append(edit); err != nil {
			return err
		}
		db.version.RemoveTable(in.FileNum)
	}

	// Record addition.
	edit := manifest.Record{
		Type: manifest.RecordTypeAddSSTable,
		Data: manifest.AddSSTable{
			FileNum:     newMeta.FileNum,
			Level:       newMeta.Level,
			SmallestKey: newMeta.SmallestKey,
			LargestKey:  newMeta.LargestKey,
			SmallestSeq: newMeta.SmallestSeq,
			LargestSeq:  newMeta.LargestSeq,
			FileSize:    newMeta.FileSize,
		},
	}
	if err := db.manifest.Append(edit); err != nil {
		return err
	}

	return db.version.AddTable(newMeta)
}

func (db *DB) MaybeScheduleCompaction() {
	db.compactionMu.Lock()
	defer db.compactionMu.Unlock()

	db.mu.RLock()
	level, needs := db.PickCompaction()
	db.mu.RUnlock()

	if needs {
		db.CompactLevel(level)
	}
}
