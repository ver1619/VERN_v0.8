package engine

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
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

// DB represents the database instance.
type DB struct {
	mu           sync.RWMutex
	flushMu      sync.Mutex
	compactionMu sync.Mutex
	wal          *wal.WAL
	memtable     *memtable.Memtable
	immutables   []*memtable.Memtable

	version *VersionSet
	nextSeq uint64
	dir     string
	opts    *Config // Configuration

	manifest    *manifest.Manifest
	nextFileNum uint64
	cache       cache.Cache

	bgErr   error      // Background error
	bgErrMu sync.Mutex // Protects bgErr
}

// Open initializes and opens the database.
func Open(dir string, options ...*Config) (*DB, error) {
	opts := DefaultConfig()
	if len(options) > 0 && options[0] != nil {
		opts = options[0]
	}

	manifestPath := filepath.Join(dir, "MANIFEST")
	walDir := filepath.Join(dir, opts.WalDir)

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
		opts:    opts,

		manifest:    m,
		nextFileNum: state.NextFileNum + 1,
	}

	if db.nextFileNum == 0 {
		db.nextFileNum = 1
	}

	// Verify existence of all SSTables.
	for _, meta := range db.version.GetAllTables() {
		path := filepath.Join(db.dir, fmt.Sprintf("%06d.sst", meta.FileNum))
		if _, err := os.Stat(path); os.IsNotExist(err) {
			db.Close()
			return nil, fmt.Errorf("consistency error: missing sstable %s", path)
		}
	}

	// Initialize 8MB cache.
	db.cache = cache.NewLRUCache(8 * 1024 * 1024)

	return db, nil
}

func (db *DB) Close() error {
	if err := db.wal.Close(); err != nil {
		return err
	}
	// Final GC on close
	db.cleanupObsoleteFiles()
	return db.manifest.Close()
}

// cleanupObsoleteFiles deletes unused SSTables.
func (db *DB) cleanupObsoleteFiles() {
	// Identify obsolete files.
	db.version.mu.RLock()
	var obsolete []uint64
	for fileNum := range db.version.Obsolete {
		obsolete = append(obsolete, fileNum)
	}
	db.version.mu.RUnlock()

	// Delete files
	for _, fileNum := range obsolete {
		path := filepath.Join(db.dir, fmt.Sprintf("%06d.sst", fileNum))
		if err := os.Remove(path); err == nil || os.IsNotExist(err) {
			// Remove from obsolete text map on success.
			db.version.mu.Lock()
			delete(db.version.Obsolete, fileNum)
			db.version.mu.Unlock()
		}
	}
}

func (db *DB) Put(key, value []byte) error {
	if err := db.checkBackgroundError(); err != nil {
		return err
	}
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
	if db.opts.SyncWrites {
		if err := db.wal.Sync(); err != nil {
			return err
		}
	}

	ikey := internal.EncodeInternalKey(key, seq, internal.RecordTypeValue)
	db.memtable.Insert(ikey, value)

	if db.memtable.ApproximateSize() >= db.opts.MemtableSizeLimit {
		db.rotateMemtableLocked()
		go db.MaybeScheduleFlush()
	}

	db.nextSeq++
	return nil
}

// Write applies a batch of operations.
func (db *DB) Write(batch *wal.Batch) error {
	if err := db.checkBackgroundError(); err != nil {
		return err
	}
	db.mu.Lock()
	defer db.mu.Unlock()

	batch.SeqStart = db.nextSeq

	// Write to WAL
	if err := db.wal.Append(*batch); err != nil {
		return err
	}
	if db.opts.SyncWrites {
		if err := db.wal.Sync(); err != nil {
			return err
		}
	}

	// Write to Memtable
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

	if db.memtable.ApproximateSize() >= db.opts.MemtableSizeLimit {
		db.rotateMemtableLocked()
		go db.MaybeScheduleFlush()
	}

	db.nextSeq = seq
	return nil
}

func (db *DB) Delete(key []byte) error {
	if err := db.checkBackgroundError(); err != nil {
		return err
	}
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
	if db.opts.SyncWrites {
		if err := db.wal.Sync(); err != nil {
			return err
		}
	}

	ikey := internal.EncodeInternalKey(key, seq, internal.RecordTypeTombstone)
	db.memtable.Insert(ikey, nil)

	db.nextSeq++
	return nil
}

func (db *DB) GetWithOptions(key []byte, opts *ReadOptions) ([]byte, error) {
	if err := db.checkBackgroundError(); err != nil {
		return nil, err
	}
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

	// Skip SSTables that don't overlap with the target key.
	sstables := db.getSortedCandidatedTables()

	for _, meta := range sstables {
		// Use key range metadata to filter SSTables.
		if len(meta.SmallestKey) > 0 && len(meta.LargestKey) > 0 {
			smallestUser := internal.ExtractUserKey(meta.SmallestKey)
			largestUser := internal.ExtractUserKey(meta.LargestKey)
			if bytes.Compare(key, smallestUser) < 0 || bytes.Compare(key, largestUser) > 0 {
				continue // Key out of range.
			}
		}

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

	db.mu.RUnlock()

	merge := iterators.NewMergeIterator(iters)
	merge.SeekToFirst()

	// Stop iterating once we pass the target key.
	for merge.Valid() {
		userKey := internal.ExtractUserKey(merge.Key())
		cmp := bytes.Compare(userKey, key)
		if cmp == 0 {
			_, typ, _ := internal.ExtractTrailer(merge.Key())
			if typ == internal.RecordTypeTombstone {
				return nil, ErrNotFound
			}
			return merge.Value(), nil
		}
		if cmp > 0 {
			// Target key not found.
			break
		}
		merge.Next()
	}

	return nil, ErrNotFound
}

func (db *DB) Get(key []byte) ([]byte, error) {
	return db.GetWithOptions(key, nil)
}

func (db *DB) GetSnapshot() *Snapshot {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return &Snapshot{
		ReadSeq: db.nextSeq - 1,
	}
}

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

// getSortedCandidatedTables returns tables sorted by file number (descending).
func (db *DB) getSortedCandidatedTables() []SSTableMeta {
	// Sort by FileNum descending
	var metas []SSTableMeta
	for _, m := range db.version.GetAllTables() {
		metas = append(metas, m)
	}

	sort.Slice(metas, func(i, j int) bool {
		return metas[i].FileNum > metas[j].FileNum
	})

	return metas
}

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

// rotateMemtableLocked moves active memtable to immutables.
func (db *DB) rotateMemtableLocked() {
	frozen := db.memtable
	db.immutables = append(db.immutables, frozen)
	db.memtable = memtable.New()
}

// freezeMemtable rotates the memtable and schedules a flush.
func (db *DB) freezeMemtable() {
	db.mu.Lock()
	db.rotateMemtableLocked()
	db.mu.Unlock()

	db.MaybeScheduleFlush()
}

func (db *DB) MaybeScheduleFlush() {

	db.flushMu.Lock()
	defer db.flushMu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			db.setBackgroundError(fmt.Errorf("panic in flush: %v", r))
		}
	}()

	for {
		runtime.Gosched() // Yield to prevent starvation
		db.mu.Lock()
		if len(db.immutables) == 0 {
			db.mu.Unlock()
			break
		}
		im := db.immutables[0]

		fileNum := db.nextFileNum
		db.nextFileNum++
		db.mu.Unlock()

		// Flush without holding lock.
		meta, err := db.flushMemtable(im, fileNum)
		if err != nil {
			db.setBackgroundError(err)
			return
		}

		// Commit
		db.mu.Lock()

		if err := db.version.AddTable(meta); err != nil {
			db.mu.Unlock()
			db.setBackgroundError(err)
			return
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
		if err := db.manifest.Append(edit); err != nil {
			db.mu.Unlock()
			db.setBackgroundError(err)
			return
		}

		cutOffEdit := manifest.Record{
			Type: manifest.RecordTypeSetWALCutoff,
			Data: manifest.SetWALCutoff{Seq: meta.LargestSeq},
		}
		if err := db.manifest.Append(cutOffEdit); err != nil {
			db.mu.Unlock()
			db.setBackgroundError(err)
			return
		}
		db.version.SetWALCutoff(meta.LargestSeq)

		db.immutables = db.immutables[1:]
		db.mu.Unlock()

		// Truncate old WAL segments.
		if meta.LargestSeq > 0 {
			walDir := filepath.Join(db.dir, db.opts.WalDir)
			wal.Truncate(walDir, meta.LargestSeq)
		}
	}

	// Trigger compaction if needed
	db.MaybeScheduleCompaction()

	// Reclaim space
	db.cleanupObsoleteFiles()
}

func (db *DB) setBackgroundError(err error) {
	db.bgErrMu.Lock()
	defer db.bgErrMu.Unlock()
	if db.bgErr == nil {
		db.bgErr = err
	}
}

func (db *DB) checkBackgroundError() error {
	db.bgErrMu.Lock()
	defer db.bgErrMu.Unlock()
	return db.bgErr
}

// CompactManifest rewrites the manifest.
func (db *DB) CompactManifest() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Snapshot current state
	var records []manifest.Record

	// Tables
	for _, meta := range db.version.GetAllTables() {
		records = append(records, manifest.Record{
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
		})
	}

	// WAL Cutoff
	records = append(records, manifest.Record{
		Type: manifest.RecordTypeSetWALCutoff,
		Data: manifest.SetWALCutoff{Seq: db.version.WALCutoffSeq},
	})

	// Rewrite
	manifestPath := filepath.Join(db.dir, "MANIFEST")

	if err := db.manifest.Close(); err != nil {
		return err
	}

	if err := manifest.Rewrite(manifestPath, records); err != nil {
		// Attempt to reopen existing manifest if rewrite fails.
		m, reopenErr := manifest.OpenManifest(manifestPath)
		if reopenErr == nil {
			db.manifest = m
		}
		return err
	}

	// Reopen
	m, err := manifest.OpenManifest(manifestPath)
	if err != nil {
		return err
	}
	db.manifest = m

	return nil
}
