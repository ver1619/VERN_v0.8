package engine

import (
	"errors"
)

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
	Tables       map[uint64]SSTableMeta
	Obsolete     map[uint64]bool
	WALCutoffSeq uint64
}

// NewVersionSet creates an empty VersionSet.
func NewVersionSet() *VersionSet {
	return &VersionSet{
		Tables:   make(map[uint64]SSTableMeta),
		Obsolete: make(map[uint64]bool),
	}
}

// AddTable adds a new SSTable to the version set.
func (v *VersionSet) AddTable(meta SSTableMeta) error {
	if _, exists := v.Tables[meta.FileNum]; exists {
		return errors.New("sstable already exists in versionset")
	}
	v.Tables[meta.FileNum] = meta
	return nil
}

// RemoveTable marks an SSTable as obsolete.
func (v *VersionSet) RemoveTable(fileNum uint64) {
	delete(v.Tables, fileNum)
	v.Obsolete[fileNum] = true
}

// SetWALCutoff sets the WAL cutoff sequence.
func (v *VersionSet) SetWALCutoff(seq uint64) {
	if seq > v.WALCutoffSeq {
		v.WALCutoffSeq = seq
	}
}
