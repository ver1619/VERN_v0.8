package engine

import (
	"bytes"
	"errors"
	"sort"
	"sync"
)

const NumLevels = 7

type SSTableMeta struct {
	FileNum     uint64
	Level       uint32
	SmallestSeq uint64
	LargestSeq  uint64
	SmallestKey []byte
	LargestKey  []byte
	FileSize    int64
}

type VersionSet struct {
	mu sync.RWMutex

	// Levels[0] overlaps; L1+ are sorted.
	Levels [NumLevels][]SSTableMeta

	Obsolete     map[uint64]bool
	WALCutoffSeq uint64
}

func NewVersionSet() *VersionSet {
	return &VersionSet{
		Levels:   [NumLevels][]SSTableMeta{},
		Obsolete: make(map[uint64]bool),
	}
}

func (v *VersionSet) AddTable(meta SSTableMeta) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if meta.Level >= NumLevels {
		return errors.New("invalid level")
	}

	v.Levels[meta.Level] = append(v.Levels[meta.Level], meta)

	// Sort L1+ by key.
	if meta.Level > 0 {
		sort.Slice(v.Levels[meta.Level], func(i, j int) bool {
			return bytes.Compare(v.Levels[meta.Level][i].SmallestKey, v.Levels[meta.Level][j].SmallestKey) < 0
		})
	}

	return nil
}

func (v *VersionSet) RemoveTable(fileNum uint64) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.Obsolete[fileNum] = true

	// Remove table from level.
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

func (v *VersionSet) SetWALCutoff(seq uint64) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if seq > v.WALCutoffSeq {
		v.WALCutoffSeq = seq
	}
}

func (v *VersionSet) GetAllTables() []SSTableMeta {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var all []SSTableMeta
	for _, files := range v.Levels {
		all = append(all, files...)
	}
	return all
}

// PickCompaction identifies level needing compaction.
func (v *VersionSet) PickCompaction(l0Trigger int, l1MaxBytes int64) (int, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	bestScore := 0.0
	bestLevel := -1

	// Calculate L0 score.
	score := float64(len(v.Levels[0])) / float64(l0Trigger)
	if score > bestScore {
		bestScore = score
		bestLevel = 0
	}

	// Calculate L1+ score.
	for l := 1; l < NumLevels-1; l++ {
		targetSize := float64(l1MaxBytes) * float64(int64(1)<<(l-1))
		var currentSize float64
		for _, t := range v.Levels[l] {
			if t.FileSize > 0 {
				currentSize += float64(t.FileSize)
			} else {
				currentSize += 2 * 1024 * 1024 // Fallback: estimate 2MB per file
			}
		}

		s := currentSize / targetSize
		if s > bestScore {
			bestScore = s
			bestLevel = l
		}
	}

	if bestScore >= 1.0 {
		return bestLevel, true
	}

	return -1, false
}

func (v *VersionSet) GetOverlappingInputs(level int, start, end []byte) []SSTableMeta {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var inputs []SSTableMeta
	for _, t := range v.Levels[level] {

		if bytes.Compare(t.LargestKey, start) < 0 || bytes.Compare(t.SmallestKey, end) > 0 {
			continue // No overlap
		}
		inputs = append(inputs, t)
	}
	return inputs
}
