package engine

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"sync"

	"vern_kv0.8/internal"
	"vern_kv0.8/internal/cache"
	"vern_kv0.8/iterators"
	"vern_kv0.8/manifest"
	"vern_kv0.8/memtable"
	"vern_kv0.8/sstable"
	"vern_kv0.8/wal"
)

var ErrNotFound = errors.New("key not found")

const MemtableSizeLimit = 4 * 1024 * 1024 // 4MB

// DB is the main engine handle.
type DB struct {
	mu           sync.RWMutex
	flushMu      sync.Mutex // Serializes flushes
	compactionMu sync.Mutex // Serializes compactions
	wal          *wal.WAL
	memtable     *memtable.Memtable   // active (mutable)
	immutables   []*memtable.Memtable // frozen (read-only)

	version *VersionSet
	nextSeq uint64
	dir     string

	manifest    *manifest.Manifest
	nextFileNum uint64
	cache       cache.Cache
}

//
// Open / initialization
//

// Open opens or creates a database at dir.
func Open(dir string) (*DB, error) {
	manifestPath := filepath.Join(dir, "MANIFEST")
	walDir := filepath.Join(dir, "wal")

	var state *RecoveredState

	// Fresh DB
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		if err := os.MkdirAll(walDir, 0755); err != nil {
			return nil, err
		}

		state = &RecoveredState{
			VersionSet: NewVersionSet(),
			Memtable:   memtable.New(),
			NextSeq:    1,
		}

		f, err := os.Create(manifestPath)
		if err != nil {
			return nil, err
		}
		f.Close()
	} else {
		// Existing DB
		var err error
		state, err = Recover(manifestPath, walDir)
		if err != nil {
			return nil, err
		}
	}

	w, err := wal.OpenWAL(walDir, 64*1024*1024)
	if err != nil {
		return nil, err
	}

	m, err := manifest.OpenManifest(manifestPath)
	if err != nil {
		return nil, err
	}

	db := &DB{
		wal:        w,
		memtable:   state.Memtable,
		immutables: make([]*memtable.Memtable, 0),

		version: state.VersionSet,
		nextSeq: state.NextSeq,
		dir:     dir,

		manifest:    m,
		nextFileNum: state.NextFileNum + 1,
	}

	// If fresh, nextFileNum starts at 1
	if db.nextFileNum == 0 {
		db.nextFileNum = 1
	}

	// Consistency Check: Verify SSTables exist
	for _, meta := range db.version.GetAllTables() {
		path := filepath.Join(db.dir, fmt.Sprintf("%06d.sst", meta.FileNum))
		if _, err := os.Stat(path); os.IsNotExist(err) {
			db.Close()
			return nil, fmt.Errorf("consistency error: missing sstable %s", path)
		}
	}

	// Initialize Block Cache (8MB)
	db.cache = cache.NewLRUCache(8 * 1024 * 1024)

	return db, nil
}

// Close closes the database.
func (db *DB) Close() error {
	if err := db.wal.Close(); err != nil {
		return err
	}
	// Final GC on close
	db.cleanupObsoleteFiles()
	return db.manifest.Close()
}

// cleanupObsoleteFiles deletes SSTable files that are no longer in the VersionSet.
func (db *DB) cleanupObsoleteFiles() {
	db.mu.Lock()
	defer db.mu.Unlock()

	for fileNum := range db.version.Obsolete {
		path := filepath.Join(db.dir, fmt.Sprintf("%06d.sst", fileNum))
		if err := os.Remove(path); err == nil {
			delete(db.version.Obsolete, fileNum)
		}
	}
}

//
// Write path
//

// Put inserts or updates a key.
func (db *DB) Put(key, value []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()

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

	if db.memtable.ApproximateSize() >= MemtableSizeLimit {
		db.rotateMemtableLocked()
		go db.MaybeScheduleFlush()
	}

	db.nextSeq++
	return nil
}

// Write applies a batch of operations atomically.
func (db *DB) Write(batch *wal.Batch) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// 1. Sequence number assignment
	batch.SeqStart = db.nextSeq

	// 2. Write to WAL
	if err := db.wal.Append(*batch); err != nil {
		return err
	}
	if err := db.wal.Sync(); err != nil {
		return err
	}

	// 3. Write to Memtable
	seq := batch.SeqStart
	for _, r := range batch.Records {
		typ := internal.RecordTypeValue
		if r.Type == wal.LogicalTypeDelete {
			typ = internal.RecordTypeTombstone
		}
		ikey := internal.EncodeInternalKey(r.Key, seq, typ)
		db.memtable.Insert(ikey, r.Value)
		seq++
	}

	if db.memtable.ApproximateSize() >= MemtableSizeLimit {
		db.rotateMemtableLocked()
		go db.MaybeScheduleFlush()
	}

	db.nextSeq = seq
	return nil
}

// Delete removes a key by inserting a tombstone.
func (db *DB) Delete(key []byte) error {
	db.mu.Lock()
	defer db.mu.Unlock()

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

//
// Read path — point lookups
//

// GetWithOptions returns the value for key using optional read options.
func (db *DB) GetWithOptions(key []byte, opts *ReadOptions) ([]byte, error) {
	db.mu.RLock()

	var iters []iterators.InternalIterator

	// Active memtable
	var mtIt iterators.InternalIterator = iterators.NewMemtableIterator(db.memtable)
	if opts != nil && opts.Snapshot != nil {
		mtIt = iterators.NewVersionFilterIterator(
			mtIt,
			opts.Snapshot.ReadSeq,
		)
	}
	iters = append(iters, mtIt)

	// Immutable memtables
	for _, im := range db.immutables {
		var imIt iterators.InternalIterator = iterators.NewMemtableIterator(im)
		if opts != nil && opts.Snapshot != nil {
			imIt = iterators.NewVersionFilterIterator(
				imIt,
				opts.Snapshot.ReadSeq,
			)
		}
		iters = append(iters, imIt)
	}

	// SSTables
	sstables := db.getSortedCandidatedTables() // Uses VersionSet (safe under RLock)

	// Create iterators for SSTables (IO).
	// Holds RLock to block conflicting writes/flushes, but allows concurrent reads.

	for _, meta := range sstables {
		path := filepath.Join(db.dir, fmt.Sprintf("%06d.sst", meta.FileNum))
		sstIt, err := sstable.NewIterator(path, db.cache)
		if err != nil {
			db.mu.RUnlock()
			return nil, err
		}

		var it iterators.InternalIterator = sstIt
		if opts != nil && opts.Snapshot != nil {
			it = iterators.NewVersionFilterIterator(it, opts.Snapshot.ReadSeq)
		}
		iters = append(iters, it)
	}

	db.mu.RUnlock() // Release lock before iterating

	merge := iterators.NewMergeIterator(iters)
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

// Get preserves v0.5 behavior (latest read).
func (db *DB) Get(key []byte) ([]byte, error) {
	return db.GetWithOptions(key, nil)
}

//
// Snapshots
//

// GetSnapshot returns a stable snapshot of the current DB state.
func (db *DB) GetSnapshot() *Snapshot {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return &Snapshot{
		ReadSeq: db.nextSeq - 1,
	}
}

//
// Iteration (snapshot-consistent)
//

// NewIterator returns a snapshot-consistent iterator.
func (db *DB) NewIterator(opts *ReadOptions) Iterator {
	db.mu.RLock()

	var iters []iterators.InternalIterator

	// Active memtable
	var mtIt iterators.InternalIterator = iterators.NewMemtableIterator(db.memtable)
	if opts != nil && opts.Snapshot != nil {
		mtIt = iterators.NewVersionFilterIterator(
			mtIt,
			opts.Snapshot.ReadSeq,
		)
	}
	iters = append(iters, mtIt)

	// Immutable memtables
	for _, im := range db.immutables {
		var imIt iterators.InternalIterator = iterators.NewMemtableIterator(im)
		if opts != nil && opts.Snapshot != nil {
			imIt = iterators.NewVersionFilterIterator(
				imIt,
				opts.Snapshot.ReadSeq,
			)
		}
		iters = append(iters, imIt)
	}

	// SSTables
	sstables := db.getSortedCandidatedTables()
	validSSTs := make([]iterators.InternalIterator, 0)

	for _, meta := range sstables {
		path := filepath.Join(db.dir, fmt.Sprintf("%06d.sst", meta.FileNum))
		sstIt, err := sstable.NewIterator(path, db.cache)
		if err != nil {
			db.mu.RUnlock()
			panic(fmt.Sprintf("failed to open sstable %s: %v", path, err))
		}

		var it iterators.InternalIterator = sstIt
		if opts != nil && opts.Snapshot != nil {
			it = iterators.NewVersionFilterIterator(it, opts.Snapshot.ReadSeq)
		}
		validSSTs = append(validSSTs, it)
	}
	iters = append(iters, validSSTs...)

	db.mu.RUnlock()

	merge := iterators.NewMergeIterator(iters)

	return &dbIterator{
		inner: merge,
	}
}

// getSortedCandidatedTables returns all tables sorted by newness (L0 logic)
func (db *DB) getSortedCandidatedTables() []SSTableMeta {
	// For v0.8, we assume all tables are L0 and overlapping.
	// Sort by FileNum descending (Newer first).

	var metas []SSTableMeta
	for _, m := range db.version.GetAllTables() {
		metas = append(metas, m)
	}

	sort.Slice(metas, func(i, j int) bool {
		return metas[i].FileNum > metas[j].FileNum
	})

	return metas
}

//
// Range / prefix scans
//

// NewRangeIterator returns an iterator over keys in [start, end).
func (db *DB) NewRangeIterator(
	start []byte,
	end []byte,
	opts *ReadOptions,
) Iterator {
	base := db.NewIterator(opts)
	return &scanIterator{
		inner: base,
		start: start,
		end:   end,
	}
}

// NewPrefixIterator returns an iterator over keys with the given prefix.
func (db *DB) NewPrefixIterator(
	prefix []byte,
	opts *ReadOptions,
) Iterator {
	base := db.NewIterator(opts)
	return &scanIterator{
		inner:  base,
		prefix: prefix,
	}
}

//
// Lifecycle (Phase L1)
//

// rotateMemtableLocked moves the active memtable into immutables.
// Caller must hold db.mu.
func (db *DB) rotateMemtableLocked() {
	frozen := db.memtable
	db.immutables = append(db.immutables, frozen)
	db.memtable = memtable.New()
}

// freezeMemtable moves the active memtable into immutables
// and initializes a new active memtable. (Thread-safe wrapper)
func (db *DB) freezeMemtable() {
	db.mu.Lock()
	db.rotateMemtableLocked()
	db.mu.Unlock() // Release DB lock before flush

	db.MaybeScheduleFlush()
}

func (db *DB) MaybeScheduleFlush() {
	// Ensure only one flush runs at a time
	db.flushMu.Lock()
	defer db.flushMu.Unlock()

	for {
		db.mu.Lock()
		if len(db.immutables) == 0 {
			db.mu.Unlock()
			break
		}
		im := db.immutables[0]

		fileNum := db.nextFileNum
		db.nextFileNum++
		db.mu.Unlock()

		// IO without lock
		meta, err := db.flushMemtable(im, fileNum)
		if err != nil {
			panic(fmt.Sprintf("flush failed: %v", err))
		}

		// Commit with lock
		db.mu.Lock()

		if err := db.version.AddTable(meta); err != nil {
			panic(err)
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
			},
		}
		if err := db.manifest.Append(edit); err != nil {
			panic(err)
		}

		cutOffEdit := manifest.Record{
			Type: manifest.RecordTypeSetWALCutoff,
			Data: manifest.SetWALCutoff{Seq: meta.LargestSeq},
		}
		if err := db.manifest.Append(cutOffEdit); err != nil {
			panic(err)
		}
		db.version.SetWALCutoff(meta.LargestSeq)

		db.immutables = db.immutables[1:]
		db.mu.Unlock()
	}

	// Trigger compaction if needed
	db.MaybeScheduleCompaction()

	// Reclaim space
	db.cleanupObsoleteFiles()
}
