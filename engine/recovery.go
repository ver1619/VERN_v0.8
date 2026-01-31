package engine

import (
	"os"
	"sort"

	"vern_kv0.5/internal"
	"vern_kv0.5/memtable"
	"vern_kv0.5/wal"
)

// RecoveredState is the result of crash recovery.
type RecoveredState struct {
	VersionSet *VersionSet
	Memtable   *memtable.Memtable
	NextSeq    uint64
}

// Recover performs full crash recovery.
func Recover(manifestPath string, walDir string) (*RecoveredState, error) {
	// Manifest replay
	vs, err := ReplayManifest(manifestPath)
	if err != nil {
		return nil, err
	}

	// Initialize memtable
	mt := memtable.New()

	// Determine highest durable sequence number
	var maxSeq uint64
	for _, meta := range vs.Tables {
		if meta.LargestSeq > maxSeq {
			maxSeq = meta.LargestSeq
		}
	}
	if vs.WALCutoffSeq > maxSeq {
		maxSeq = vs.WALCutoffSeq
	}

	// Discover WAL segments
	entries, err := os.ReadDir(walDir)
	if err != nil {
		return nil, err
	}

	var segments []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if wal.IsWALFile(e.Name()) {
			segments = append(segments, wal.PathJoin(walDir, e.Name()))
		}
	}

	sort.Strings(segments)

	// WAL replay
	for _, path := range segments {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		offset := 0
		for offset < len(data) {
			batch, n, err := wal.DecodeRecord(data[offset:])
			if err != nil {
				// HARD STOP on corruption
				break
			}

			batchMaxSeq := batch.SeqStart + uint64(len(batch.Records)) - 1
			if batchMaxSeq <= vs.WALCutoffSeq {
				offset += n
				continue
			}

			// Apply batch atomically
			seq := batch.SeqStart
			for _, r := range batch.Records {
				ikey := internal.EncodeInternalKey(
					r.Key,
					seq,
					convertLogicalType(r.Type),
				)
				mt.Insert(ikey, r.Value)

				if seq > maxSeq {
					maxSeq = seq
				}
				seq++
			}

			offset += n
		}
	}

	return &RecoveredState{
		VersionSet: vs,
		Memtable:   mt,
		NextSeq:    maxSeq + 1,
	}, nil
}

// convert WAL logical record type → InternalKey record type
func convertLogicalType(t uint8) internal.RecordType {
	switch t {
	case wal.LogicalTypePut:
		return internal.RecordTypeValue
	case wal.LogicalTypeDelete:
		return internal.RecordTypeTombstone
	default:
		panic("unknown WAL logical record type")
	}
}
