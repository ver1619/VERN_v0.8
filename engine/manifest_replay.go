package engine

import (
	"os"

	"vern_kv0.5/manifest"
)

// ReplayManifest replays the manifest file and builds a VersionSet.
// Replay stops at the first invalid record.
func ReplayManifest(path string) (*VersionSet, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

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
