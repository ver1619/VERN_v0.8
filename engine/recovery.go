package engine

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"

	"vern_kv0.8/internal"
	"vern_kv0.8/manifest"
	"vern_kv0.8/memtable"
	"vern_kv0.8/sstable"
	"vern_kv0.8/wal"
)

type RecoveredState struct {
	VersionSet  *VersionSet
	Memtable    *memtable.Memtable
	NextSeq     uint64
	NextFileNum uint64
}

// Recover restores DB state.
func Recover(dbDir, walDir string, memtableLimit int) (*RecoveredState, error) {
	manifestPath := filepath.Join(dbDir, "MANIFEST")

	// Replay manifest.
	vs, err := ReplayManifest(manifestPath)
	if err != nil {
		return nil, err
	}

	// Initialize memtable.
	mt := memtable.New()

	// Find max sequence and file number.
	var maxSeq uint64
	var maxFileNum uint64

	for _, meta := range vs.GetAllTables() {
		if meta.LargestSeq > maxSeq {
			maxSeq = meta.LargestSeq
		}
		if meta.FileNum > maxFileNum {
			maxFileNum = meta.FileNum
		}
	}
	if vs.WALCutoffSeq > maxSeq {
		maxSeq = vs.WALCutoffSeq
	}

	// Find WAL files.
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

	// Replay WAL.
	for _, path := range segments {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		offset := 0
		for offset < len(data) {
			batch, n, err := wal.DecodeRecord(data[offset:])
			if err != nil {
				// Stop on corruption.
				break
			}

			batchMaxSeq := batch.SeqStart + uint64(len(batch.Records)) - 1
			if batchMaxSeq <= vs.WALCutoffSeq {
				offset += n
				continue
			}

			// Apply batch.
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

			// Paging: Flush if memtable grows too large.
			if mt.ApproximateSize() > memtableLimit {
				fileNum := maxFileNum + 1
				maxFileNum++

				sstPath := filepath.Join(dbDir, fmt.Sprintf("%06d.sst", fileNum))
				meta, err := writeMemtableToSSTable(mt, sstPath, fileNum)
				if err != nil {
					return nil, err
				}

				// Update VersionSet.
				if err := vs.AddTable(meta); err != nil {
					return nil, err
				}

				// Update Manifest.
				m, err := manifest.OpenManifest(manifestPath)
				if err != nil {
					return nil, err
				}

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
				if err := m.Append(edit); err != nil {
					m.Close()
					return nil, err
				}
				m.Close()

				// Reset Memtable.
				mt = memtable.New()
			}
		}
	}

	return &RecoveredState{
		VersionSet:  vs,
		Memtable:    mt,
		NextSeq:     maxSeq + 1,
		NextFileNum: maxFileNum,
	}, nil
}

func writeMemtableToSSTable(mt *memtable.Memtable, filename string, fileNum uint64) (SSTableMeta, error) {
	iter := mt.Iterator()
	iter.SeekToFirst()

	builder, err := sstable.NewBuilder(filename)
	if err != nil {
		return SSTableMeta{}, err
	}

	var meta SSTableMeta
	meta.FileNum = fileNum
	meta.Level = 0
	meta.SmallestSeq = math.MaxUint64

	first := true
	for iter.Valid() {
		key := iter.Key()
		val := iter.Value()

		if err := builder.Add(key, val); err != nil {
			return SSTableMeta{}, err
		}

		if first {
			meta.SmallestKey = make([]byte, len(key))
			copy(meta.SmallestKey, key)
			first = false
		}
		meta.LargestKey = make([]byte, len(key))
		copy(meta.LargestKey, key)

		seq, _, _ := internal.ExtractTrailer(key)
		if seq < meta.SmallestSeq {
			meta.SmallestSeq = seq
		}
		if seq > meta.LargestSeq {
			meta.LargestSeq = seq
		}

		iter.Next()
	}

	if err := builder.Close(); err != nil {
		return SSTableMeta{}, err
	}

	meta.FileSize = int64(builder.Size())
	return meta, nil
}

// convertLogicalType converts WAL type to internal type.
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
