package engine

import (
	"os"

	"vern_kv0.8/manifest"
)

// ReplayManifest rebuilds VersionSet from manifest file.
func ReplayManifest(path string) (*VersionSet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	vs := NewVersionSet()
	offset := 0

	for offset < len(data) {
		rec, n, err := manifest.DecodeRecord(data[offset:])
		if err != nil {
			// Hard stop on corruption
			break
		}

		switch rec.Type {
		case manifest.RecordTypeAddSSTable:
			r := rec.Data.(manifest.AddSSTable)
			vs.AddTable(SSTableMeta{
				FileNum:     r.FileNum,
				Level:       r.Level,
				SmallestSeq: r.SmallestSeq,
				LargestSeq:  r.LargestSeq,
				SmallestKey: r.SmallestKey,
				LargestKey:  r.LargestKey,
				FileSize:    r.FileSize,
			})

		case manifest.RecordTypeRemoveSSTable:
			r := rec.Data.(manifest.RemoveSSTable)
			vs.RemoveTable(r.FileNum)

		case manifest.RecordTypeSetWALCutoff:
			r := rec.Data.(manifest.SetWALCutoff)
			vs.SetWALCutoff(r.Seq)
		}

		offset += n
	}

	return vs, nil
}
