package engine

import (
	"errors"
	"path/filepath"

	"vern_kv0.5/internal"
	"vern_kv0.5/iterators"
	"vern_kv0.5/memtable"
	"vern_kv0.5/wal"
)

var ErrNotFound = errors.New("key not found")

// DB is the main engine handle.
type DB struct {
	wal      *wal.WAL
	memtable *memtable.Memtable
	version  *VersionSet
	nextSeq  uint64
	dir      string
}

// Open opens or creates a database at dir.
func Open(dir string) (*DB, error) {
	// Bootstrap if needed
	if err := bootstrapIfNeeded(dir); err != nil {
		return nil, err
	}

	manifestPath := filepath.Join(dir, "MANIFEST")
	walDir := filepath.Join(dir, "wal")

	// Recover state
	state, err := Recover(manifestPath, walDir)
	if err != nil {
		return nil, err
	}

	// Open WAL
	w, err := wal.OpenWAL(walDir, 64*1024*1024)
	if err != nil {
		return nil, err
	}

	return &DB{
		wal:      w,
		memtable: state.Memtable,
		version:  state.VersionSet,
		nextSeq:  state.NextSeq,
		dir:      dir,
	}, nil
}

// Put inserts or updates a key.
func (db *DB) Put(key, value []byte) error {
	seq := db.nextSeq

	batch := wal.Batch{
		SeqStart: seq,
		Records: []wal.LogicalRecord{
			{
				Key:   key,
				Value: value,
				Type:  wal.LogicalTypePut,
			},
		},
	}

	if err := db.wal.Append(batch); err != nil {
		return err
	}
	if err := db.wal.Sync(); err != nil {
		return err
	}

	ikey := internal.EncodeInternalKey(key, seq, internal.RecordTypeValue)
	db.memtable.Insert(ikey, value)

	db.nextSeq++
	return nil
}

// Delete removes a key by inserting a tombstone.
func (db *DB) Delete(key []byte) error {
	seq := db.nextSeq

	batch := wal.Batch{
		SeqStart: seq,
		Records: []wal.LogicalRecord{
			{
				Key:  key,
				Type: wal.LogicalTypeDelete,
			},
		},
	}

	if err := db.wal.Append(batch); err != nil {
		return err
	}
	if err := db.wal.Sync(); err != nil {
		return err
	}

	ikey := internal.EncodeInternalKey(key, seq, internal.RecordTypeTombstone)
	db.memtable.Insert(ikey, nil)

	db.nextSeq++
	return nil
}

// Get returns the value for key or ErrNotFound.
func (db *DB) Get(key []byte) ([]byte, error) {
	mtIt := iterators.NewMemtableIterator(db.memtable)

	merge := iterators.NewMergeIterator([]iterators.InternalIterator{
		mtIt,
	})

	merge.SeekToFirst()

	for merge.Valid() {
		userKey := internal.ExtractUserKey(merge.Key())
		if string(userKey) == string(key) {
			_, typ, _ := internal.ExtractTrailer(merge.Key())
			if typ == internal.RecordTypeTombstone {
				return nil, ErrNotFound
			}
			return merge.Value(), nil
		}
		merge.Next()
	}

	return nil, ErrNotFound
}
