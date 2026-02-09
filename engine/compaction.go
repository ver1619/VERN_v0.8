package engine

import (
	"fmt"
	"path/filepath"

	"vern_kv0.8/internal"
	"vern_kv0.8/iterators"
	"vern_kv0.8/manifest"
	"vern_kv0.8/sstable"
)

// PickCompactionSimple picks a compaction.
// Uses logic from VersionSet (which was updated to provide simple picking).
func (db *DB) PickCompaction() (int, bool) {
	return db.version.PickCompactionSimple()
}

// CompactLevel executes compaction.
// L0 -> L1: merge all L0 + overlaps.
// L1+ -> L+1: pick one file + overlaps.
func (db *DB) CompactLevel(level int) error {
	if level >= NumLevels-1 {
		return fmt.Errorf("cannot compact max level")
	}

	// 1. Pick inputs under lock
	db.mu.Lock()
	var inputs []SSTableMeta

	if level == 0 {
		// L0 compaction strategy stays same: compact all L0 + all L1 into L1
		l0 := db.version.Levels[0]
		if len(l0) == 0 {
			db.mu.Unlock()
			return nil
		}
		l1 := db.version.Levels[1]

		inputs = append(inputs, l0...)
		inputs = append(inputs, l1...)
	} else {
		// L1+ strategy: Pick ONE file from current level
		currentLevel := db.version.Levels[level]
		if len(currentLevel) == 0 {
			db.mu.Unlock()
			return nil
		}
		picked := currentLevel[0] // Simple policy: pick first
		inputs = append(inputs, picked)

		// Pick overlaps from next level
		overlaps := db.version.GetOverlappingInputs(level+1, picked.SmallestKey, picked.LargestKey)
		inputs = append(inputs, overlaps...)
	}

	fileNum := db.nextFileNum
	db.nextFileNum++
	db.mu.Unlock()

	// 2. Create Iterator
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

	// 3. Write output(s)
	// We might produce multiple files if size > limit.

	// Determine target level
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
	first := true

	for merge.Valid() {
		key := merge.Key()
		val := merge.Value()

		_, typ, _ := internal.ExtractTrailer(key)

		// Tombstone GC:
		// Drop tombstones if L2+ levels are empty (safe bottom-level GC).
		if typ == internal.RecordTypeTombstone {
			isBottom := true
			for l := targetLevel + 1; l < NumLevels; l++ {
				if len(db.version.Levels[l]) > 0 {
					// Skip overlap check for v0.8 (conservative approach).
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

		merge.Next()
	}

	if err := builder.Close(); err != nil {
		return err
	}

	// 4. Update Versions under lock
	db.mu.Lock()
	defer db.mu.Unlock()

	// Remove inputs, add output
	newMeta := SSTableMeta{
		FileNum:     fileNum,
		Level:       uint32(targetLevel),
		SmallestKey: smallest,
		LargestKey:  largest,
	}

	// Log deletions
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

	// Log addition
	edit := manifest.Record{
		Type: manifest.RecordTypeAddSSTable,
		Data: manifest.AddSSTable{
			FileNum:     newMeta.FileNum,
			Level:       newMeta.Level,
			SmallestKey: newMeta.SmallestKey,
			LargestKey:  newMeta.LargestKey,
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
