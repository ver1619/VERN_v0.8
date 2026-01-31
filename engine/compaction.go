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

// CompactLevel executes compaction for a given level.
// Simplest strategy: L0 -> L1.
// 1. Pick all L0 files.
// 2. Pick overlapping L1 files.
// 3. Merge.
// 4. Update Manifest.
func (db *DB) CompactLevel(level int) error {
	if level != 0 {
		return fmt.Errorf("only L0->L1 compaction supported in v0.8")
	}

	// 1. Pick inputs under lock
	db.mu.Lock()
	l0 := db.version.Levels[0]
	if len(l0) == 0 {
		db.mu.Unlock()
		return nil
	}

	l1 := db.version.Levels[1]

	var inputs []SSTableMeta
	inputs = append(inputs, l0...)
	inputs = append(inputs, l1...)

	fileNum := db.nextFileNum
	db.nextFileNum++
	db.mu.Unlock()

	// 2. Create Iterator
	var iters []iterators.InternalIterator
	for _, meta := range inputs {
		path := filepath.Join(db.dir, fmt.Sprintf("%06d.sst", meta.FileNum))
		sstIt, err := sstable.NewIterator(path)
		if err != nil {
			return err
		}
		iters = append(iters, sstIt)
	}

	merge := iterators.NewMergeIterator(iters)
	merge.SeekToFirst()

	// 3. Write output(s)
	// We might produce multiple files if size > limit.
	// For v0.8 basic, single file output if small enough.

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
		// If this is a tombstone and we are merging into L1,
		// and we know for sure there is no data at L2..L6 for this key range,
		// we can drop it.
		// For v0.8 simplified: if L2..L6 are empty, we drop all tombstones merging into L1.
		if typ == internal.RecordTypeTombstone {
			isBottom := true
			for l := 2; l < NumLevels; l++ {
				if len(db.version.Levels[l]) > 0 {
					// We'd need overlap check here for production,
					// but for v0.8 if any higher level exists, we play safe.
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
		Level:       1, // Always L0->L1
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
	needsCompaction := len(db.version.Levels[0]) >= 4
	db.mu.RUnlock()

	if needsCompaction {
		db.CompactLevel(0)
	}
}
