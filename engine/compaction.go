package engine

import (
	"bytes"
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

// Run compaction for this level.
func (db *DB) CompactLevel(level int) error {
	if level >= NumLevels-1 {
		return fmt.Errorf("cannot compact max level")
	}

	// Pick inputs (hold lock).
	db.mu.Lock()
	oldestSnapshotSeq := db.getOldestSnapshotSeq()
	var inputs []SSTableMeta

	if level == 0 {
		// L0 to L1.
		l0 := db.version.Levels[0]
		if len(l0) == 0 {
			db.mu.Unlock()
			return nil
		}

		// Grab all L0, plus overlapping L1.
		inputs = append(inputs, l0...)

		// Find L0 range.
		var smallest, largest []byte
		first := true
		cmp := internal.Comparator{}

		for _, f := range l0 {
			if first {
				smallest = f.SmallestKey
				largest = f.LargestKey
				first = false
				continue
			}
			if cmp.Compare(f.SmallestKey, smallest) < 0 {
				smallest = f.SmallestKey
			}
			if cmp.Compare(f.LargestKey, largest) > 0 {
				largest = f.LargestKey
			}
		}

		// Get L1 overlaps.
		l1 := db.version.GetOverlappingInputs(1, smallest, largest)
		inputs = append(inputs, l1...)
	} else {
		// Standard level compaction.
		currentLevel := db.version.Levels[level]
		if len(currentLevel) == 0 {
			db.mu.Unlock()
			return nil
		}
		picked := currentLevel[0]
		inputs = append(inputs, picked)

		// Grab overlaps from next level.
		overlaps := db.version.GetOverlappingInputs(level+1, picked.SmallestKey, picked.LargestKey)
		inputs = append(inputs, overlaps...)
	}

	db.mu.Unlock()

	// Spin up iterators.
	var iters []iterators.InternalIterator
	for _, meta := range inputs {
		path := filepath.Join(db.dir, fmt.Sprintf("%06d.sst", meta.FileNum))
		sstIt, err := sstable.NewIterator(path, db.cache)
		if err != nil {
			return err
		}
		iters = append(iters, sstIt)
	}

	merge := iterators.NewMergeIterator(iters, false)
	merge.SeekToFirst()

	// Next level down.
	targetLevel := level + 1
	if level == 0 {
		targetLevel = 1
	}

	var newFiles []SSTableMeta

	// Out file state.
	var (
		builder     *sstable.Builder
		currentMeta SSTableMeta
		first       bool
	)

	// Close current file.
	finishFile := func() error {
		if builder == nil {
			return nil
		}
		if err := builder.Close(); err != nil {
			return err
		}

		// Get size.
		path := filepath.Join(db.dir, fmt.Sprintf("%06d.sst", currentMeta.FileNum))
		if info, err := os.Stat(path); err == nil {
			currentMeta.FileSize = info.Size()
		}

		newFiles = append(newFiles, currentMeta)
		builder = nil
		return nil
	}

	// Make new file.
	startFile := func() error {
		db.mu.Lock()
		fileNum := db.nextFileNum
		db.nextFileNum++
		db.mu.Unlock()

		path := filepath.Join(db.dir, fmt.Sprintf("%06d.sst", fileNum))
		b, err := sstable.NewBuilder(path)
		if err != nil {
			return err
		}

		builder = b
		currentMeta = SSTableMeta{
			FileNum:     fileNum,
			Level:       uint32(targetLevel),
			SmallestSeq: math.MaxUint64,
		}
		first = true
		return nil
	}

	// First file.
	if err := startFile(); err != nil {
		return err
	}

	for merge.Valid() {
		key := merge.Key()
		val := merge.Value()

		// Too big? Rotate.
		if builder.Size() >= 20*1024*1024 { // 20MB split.
			if err := finishFile(); err != nil {
				return err
			}
			if err := startFile(); err != nil {
				return err
			}
		}

		seq, typ, _ := internal.ExtractTrailer(key)

		// GC Tombstones.
		// Drop if bottom-most AND invisible to snapshots.
		if typ == internal.RecordTypeTombstone && seq <= oldestSnapshotSeq {
			isBottom := true
			for l := targetLevel + 1; l < NumLevels; l++ {
				if len(db.version.Levels[l]) > 0 {
					// Has overlap below? Keep it.
					isBottom = false
					break
				}
			}
			if isBottom {
				// Safe to drop.
				// Also skip shadowed versions.
				userKey := internal.ExtractUserKey(key)
				for {
					merge.Next()
					if !merge.Valid() {
						break
					}
					nextKey := internal.ExtractUserKey(merge.Key())
					if !bytes.Equal(userKey, nextKey) {
						break
					}
					// Drop shadowed.
				}
				continue
			}
		}

		if err := builder.Add(key, val); err != nil {
			return err
		}

		if first {
			currentMeta.SmallestKey = make([]byte, len(key))
			copy(currentMeta.SmallestKey, key)
			first = false
		}
		currentMeta.LargestKey = make([]byte, len(key))
		copy(currentMeta.LargestKey, key)

		// Update seq bounds.
		if seq < currentMeta.SmallestSeq {
			currentMeta.SmallestSeq = seq
		}
		if seq > currentMeta.LargestSeq {
			currentMeta.LargestSeq = seq
		}

		merge.Next()
	}

	if err := finishFile(); err != nil {
		return err
	}

	// Update manifest.
	db.mu.Lock()
	defer db.mu.Unlock()

	// Log deletions.
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

	// Log additions.
	for _, meta := range newFiles {
		edit := manifest.Record{
			Type: manifest.RecordTypeAddSSTable,
			Data: manifest.AddSSTable{
				FileNum:     meta.FileNum,
				Level:       meta.Level,
				SmallestKey: meta.SmallestKey,
				LargestKey:  meta.LargestKey,
				SmallestSeq: meta.SmallestSeq,
				LargestSeq:  meta.LargestSeq,
				FileSize:    meta.FileSize,
			},
		}
		if err := db.manifest.Append(edit); err != nil {
			return err
		}
		if err := db.version.AddTable(meta); err != nil {
			return err
		}
	}

	return nil
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
