package engine

import (
	"bytes"
	"errors"
	"sort"
)

const NumLevels = 7

// SSTableMeta represents metadata for one SSTable.
type SSTableMeta struct {
	FileNum     uint64
	Level       uint32
	SmallestSeq uint64
	LargestSeq  uint64
	SmallestKey []byte
	LargestKey  []byte
}

// VersionSet is the in-memory representation of manifest state.
type VersionSet struct {
	// Levels holds the tables for each level.
	// L0 is unsorted (overlapping files).
	// L1+ is sorted by key and non-overlapping.
	Levels [NumLevels][]SSTableMeta

	Obsolete     map[uint64]bool
	WALCutoffSeq uint64
}

// NewVersionSet creates an empty VersionSet.
func NewVersionSet() *VersionSet {
	return &VersionSet{
		Levels:   [NumLevels][]SSTableMeta{},
		Obsolete: make(map[uint64]bool),
	}
}

// AddTable adds a new SSTable to the version set.
func (v *VersionSet) AddTable(meta SSTableMeta) error {
	if meta.Level >= NumLevels {
		return errors.New("invalid level")
	}

	v.Levels[meta.Level] = append(v.Levels[meta.Level], meta)

	// For L1+, we should keep them sorted by key
	if meta.Level > 0 {
		sort.Slice(v.Levels[meta.Level], func(i, j int) bool {
			return bytes.Compare(v.Levels[meta.Level][i].SmallestKey, v.Levels[meta.Level][j].SmallestKey) < 0
		})
	}
	// L0: overlapping files, naturally ordered by append sequence.

	return nil
}

// RemoveTable marks an SSTable as obsolete and removes it from levels.
func (v *VersionSet) RemoveTable(fileNum uint64) {
	v.Obsolete[fileNum] = true

	// Find and remove
	for l := 0; l < NumLevels; l++ {
		files := v.Levels[l]
		for i, t := range files {
			if t.FileNum == fileNum {
				// Remove
				v.Levels[l] = append(files[:i], files[i+1:]...)
				return
			}
		}
	}
}

// SetWALCutoff sets the WAL cutoff sequence.
func (v *VersionSet) SetWALCutoff(seq uint64) {
	if seq > v.WALCutoffSeq {
		v.WALCutoffSeq = seq
	}
}

// GetAllTables returns a flattened list of all tables (useful for recovery/scanning).
func (v *VersionSet) GetAllTables() []SSTableMeta {
	var all []SSTableMeta
	for _, files := range v.Levels {
		all = append(all, files...)
	}
	return all
}

// PickCompactionSimple is a placeholder for compaction picking logic.
// Returns level to compact and boolean indicating if compaction is needed.
func (v *VersionSet) PickCompactionSimple() (int, bool) {
	// L0 -> L1 triggers if L0 has >= 4 files
	if len(v.Levels[0]) >= 4 {
		return 0, true
	}

	// Check L1+ sizes (simplified count threshold for now)
	for l := 1; l < NumLevels-1; l++ {
		if len(v.Levels[l]) > 10 {
			return l, true
		}
	}

	return -1, false
}

// GetOverlappingInputs returns all tables in the specified level that overlap with [start, end].
func (v *VersionSet) GetOverlappingInputs(level int, start, end []byte) []SSTableMeta {
	var inputs []SSTableMeta
	for _, t := range v.Levels[level] {
		// Intersection check: ! (t.Largest < start || t.Smallest > end)
		// But bytes.Compare needed
		if bytes.Compare(t.LargestKey, start) < 0 || bytes.Compare(t.SmallestKey, end) > 0 {
			continue // No overlap
		}
		inputs = append(inputs, t)
	}
	return inputs
}
