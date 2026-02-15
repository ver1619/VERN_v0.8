# Tests Report v0.8

**This report provides a detailed breakdown of the 38 test files**

## Test Results
```python
=== RUN   TestCompactionSplit
    compaction_split_test.go:61: Starting manual compaction of L0...
    compaction_split_test.go:73: Compaction complete. L0=0, L1=2
    compaction_split_test.go:87: L1 File: Num=9, Size=21035277
    compaction_split_test.go:87: L1 File: Num=10, Size=11136359
--- PASS: TestCompactionSplit (6.26s)
=== RUN   TestCompactionL0toL1
--- PASS: TestCompactionL0toL1 (0.16s)
=== RUN   TestCompactionL1ToL2
--- PASS: TestCompactionL1ToL2 (0.02s)
=== RUN   TestConcurrencySWMR
--- PASS: TestConcurrencySWMR (7.80s)
=== RUN   TestDBPutGetDelete
--- PASS: TestDBPutGetDelete (0.02s)
=== RUN   TestDBRecovery
--- PASS: TestDBRecovery (0.01s)
=== RUN   TestGetWithSnapshot
--- PASS: TestGetWithSnapshot (0.01s)
=== RUN   TestSnapshotDeleteVisibility
--- PASS: TestSnapshotDeleteVisibility (0.01s)
=== RUN   TestImmutableMemtableReadVisibility
--- PASS: TestImmutableMemtableReadVisibility (0.04s)
=== RUN   TestFlushManual
--- PASS: TestFlushManual (0.04s)
=== RUN   TestFlushRecovery
--- PASS: TestFlushRecovery (0.03s)
=== RUN   TestFullCycleFlushAndRead
--- PASS: TestFullCycleFlushAndRead (0.06s)
=== RUN   TestSnapshotIteratorStability
--- PASS: TestSnapshotIteratorStability (0.02s)
=== RUN   TestIteratorDeleteVisibility
--- PASS: TestIteratorDeleteVisibility (0.01s)
=== RUN   TestRecoveryPaging
    recovery_paging_test.go:62: Recovered 5 L0 files
--- PASS: TestRecoveryPaging (0.05s)
=== RUN   TestFullRecovery
--- PASS: TestFullRecovery (0.01s)
=== RUN   TestRangeScanSnapshot
--- PASS: TestRangeScanSnapshot (0.03s)
=== RUN   TestPrefixScan
--- PASS: TestPrefixScan (0.02s)
=== RUN   TestSnapshotCapturesSequence
--- PASS: TestSnapshotCapturesSequence (0.01s)
=== RUN   TestSnapshotIsImmutable
--- PASS: TestSnapshotIsImmutable (0.01s)
=== RUN   TestTombstoneGC_SnapshotSafety
--- PASS: TestTombstoneGC_SnapshotSafety (0.74s)
=== RUN   TestVersionSetReplay
--- PASS: TestVersionSetReplay (0.01s)
=== RUN   TestVersionSet_AddTable
--- PASS: TestVersionSet_AddTable (0.00s)
=== RUN   TestVersionSet_RemoveTable
--- PASS: TestVersionSet_RemoveTable (0.00s)
=== RUN   TestVersionSet_PickCompaction
--- PASS: TestVersionSet_PickCompaction (0.00s)
=== RUN   TestVersionSet_GetOverlappingInputs
--- PASS: TestVersionSet_GetOverlappingInputs (0.00s)
PASS
ok      vern_kv0.8/engine       15.410s
=== RUN   TestInternalKeyEncodeDecode
--- PASS: TestInternalKeyEncodeDecode (0.00s)
=== RUN   TestComparatorOrdering
--- PASS: TestComparatorOrdering (0.00s)
=== RUN   TestExtractHelpers
--- PASS: TestExtractHelpers (0.00s)
PASS
ok      vern_kv0.8/internal     0.006s
=== RUN   TestLRUCache_Basic
--- PASS: TestLRUCache_Basic (0.00s)
=== RUN   TestLRUCache_Overwrite
--- PASS: TestLRUCache_Overwrite (0.00s)
=== RUN   TestLRUCache_Eviction
--- PASS: TestLRUCache_Eviction (0.00s)
=== RUN   TestLRUCache_LargeItem
--- PASS: TestLRUCache_LargeItem (0.00s)
=== RUN   TestLRUCache_Concurrency
--- PASS: TestLRUCache_Concurrency (0.00s)
PASS
ok      vern_kv0.8/internal/cache       0.010s
=== RUN   TestMergeMemtableAndSSTable
--- PASS: TestMergeMemtableAndSSTable (0.00s)
=== RUN   TestMergeIteratorNoDedup
--- PASS: TestMergeIteratorNoDedup (0.00s)
=== RUN   TestVersionFilterIterator
--- PASS: TestVersionFilterIterator (0.00s)
=== RUN   TestVersionFilterHidesFutureWrites
--- PASS: TestVersionFilterHidesFutureWrites (0.00s)
PASS
ok      vern_kv0.8/iterators    0.011s
=== RUN   TestManifestAppendAndDecode
--- PASS: TestManifestAppendAndDecode (0.02s)
=== RUN   TestManifestCorruption
--- PASS: TestManifestCorruption (0.00s)
PASS
ok      vern_kv0.8/manifest     0.028s
=== RUN   TestMemtableInsertAndSize
--- PASS: TestMemtableInsertAndSize (0.00s)
=== RUN   TestMemtableOrderingByUserKey
--- PASS: TestMemtableOrderingByUserKey (0.00s)
=== RUN   TestMemtableOrderingBySequenceDesc
--- PASS: TestMemtableOrderingBySequenceDesc (0.00s)
=== RUN   TestMemtableStoresTombstone
--- PASS: TestMemtableStoresTombstone (0.00s)
=== RUN   TestSkiplistInsertAndIterator
--- PASS: TestSkiplistInsertAndIterator (0.00s)
=== RUN   TestSkiplistInsertDuplicateUpdates
--- PASS: TestSkiplistInsertDuplicateUpdates (0.00s)
=== RUN   TestSkiplistOrdering
--- PASS: TestSkiplistOrdering (0.00s)
=== RUN   TestSkiplistIteratorEmpty
--- PASS: TestSkiplistIteratorEmpty (0.00s)
=== RUN   TestSkiplistIteratorSnapshot
--- PASS: TestSkiplistIteratorSnapshot (0.00s)
=== RUN   TestSkiplistLargeInsert
--- PASS: TestSkiplistLargeInsert (0.00s)
=== RUN   TestRandomLevel
--- PASS: TestRandomLevel (0.00s)
PASS
ok      vern_kv0.8/memtable     0.010s
=== RUN   TestBlockBuilder
--- PASS: TestBlockBuilder (0.00s)
=== RUN   TestBlockSeek
--- PASS: TestBlockSeek (0.00s)
=== RUN   TestCompression
    compression_test.go:12: Compression: 47 -> 26
--- PASS: TestCompression (0.00s)
=== RUN   TestEndToEndCompression
--- PASS: TestEndToEndCompression (0.00s)
=== RUN   TestBloomFilterLogic
--- PASS: TestBloomFilterLogic (0.00s)
=== RUN   TestFilterIntegration
--- PASS: TestFilterIntegration (0.01s)
=== RUN   TestFullSSTable
--- PASS: TestFullSSTable (0.03s)
=== RUN   TestSSTableIterator
--- PASS: TestSSTableIterator (0.01s)
=== RUN   TestPrefixCompression
--- PASS: TestPrefixCompression (0.01s)
PASS
ok      vern_kv0.8/sstable      0.072s
=== RUN   TestManifestCompaction
    manifest_test.go:65: Initial Manifest Size: 0, Final: 108
--- PASS: TestManifestCompaction (0.17s)
PASS
ok      vern_kv0.8/tests        0.174s
=== RUN   TestCrashConsistency
--- PASS: TestCrashConsistency (0.46s)
=== RUN   TestRecoveryIsDeterministic
--- PASS: TestRecoveryIsDeterministic (0.03s)
=== RUN   TestTruncationIdempotence
--- PASS: TestTruncationIdempotence (0.01s)
=== RUN   TestCrashBeforeWALFsync
--- PASS: TestCrashBeforeWALFsync (0.58s)
PASS
ok      vern_kv0.8/tests/crash  1.086s
=== RUN   TestReplayRepeatability
--- PASS: TestReplayRepeatability (0.02s)
PASS
ok      vern_kv0.8/tests/determinism    0.033s
=== RUN   TestAutoFlushKeyTrigger
--- PASS: TestAutoFlushKeyTrigger (24.96s)
=== RUN   TestIntegrationBasic
--- PASS: TestIntegrationBasic (0.43s)
=== RUN   TestIntegrationCompactionSpaceReclamation
--- PASS: TestIntegrationCompactionSpaceReclamation (0.04s)
=== RUN   TestOpenPutGet
--- PASS: TestOpenPutGet (0.01s)
PASS
ok      vern_kv0.8/tests/integration    25.450s
=== RUN   TestEncodeDecodeRecord
--- PASS: TestEncodeDecodeRecord (0.00s)
=== RUN   TestCRCFailureStopsDecode
--- PASS: TestCRCFailureStopsDecode (0.00s)
=== RUN   TestPartialRecordFails
--- PASS: TestPartialRecordFails (0.00s)
=== RUN   TestSegmentAppendAndSync
--- PASS: TestSegmentAppendAndSync (0.01s)
=== RUN   TestSegmentReopenAppend
--- PASS: TestSegmentReopenAppend (0.02s)
=== RUN   TestAppendAfterCloseFails
--- PASS: TestAppendAfterCloseFails (0.01s)
=== RUN   TestWALTruncation
--- PASS: TestWALTruncation (0.02s)
=== RUN   TestWALAppendAndRotate
--- PASS: TestWALAppendAndRotate (0.10s)
=== RUN   TestWALReopen
--- PASS: TestWALReopen (0.01s)
=== RUN   TestWALCloseClosesAllSegments
--- PASS: TestWALCloseClosesAllSegments (0.01s)
PASS
ok      vern_kv0.8/wal  0.203s
```

---

## Report

## Engine Tests

| Test File | Description |
|-----------|------------|
| `engine/compaction_test.go` | Validates L0 → L1 compaction logic and version precedence handling. |
| `engine/compaction_tiered_test.go` | Validates tiered compaction strategy (L1 → L2). |
| `engine/concurrency_test.go` | Verifies Single Writer Multiple Reader (SWMR) semantics and snapshot isolation under concurrent load. |
| `engine/db_test.go` | Tests core database operations: Put, Get, Delete, recovery, snapshots, and immutable memtable visibility. |
| `engine/flush_test.go` | Tests manual flush operations and post-flush recovery. |
| `engine/full_cycle_test.go` | Validates complete data lifecycle: Write → Flush → Read → Recover. |
| `engine/iterator_test.go` | Tests iterator semantics, including snapshot visibility and deleted key handling. |
| `engine/recovery_test.go` | Validates recovery from MANIFEST and WAL files. |
| `engine/scan_iterator_test.go` | Tests range scan and prefix scan functionality. |
| `engine/snapshot_test.go` | Validates snapshot creation, sequence capture, and immutability guarantees. |
| `engine/version_set_test.go` | Tests version set management: table add/remove, compaction selection, and overlap resolution. |
| `engine/compaction_split_test.go` | Validates compaction output splitting into multiple L1 files with proper size thresholds and version updates. |
| `engine/recovery_paging_test.go` | Validates recovery when multiple L0 files exist and ensures correct replay ordering and version reconstruction. |
| `engine/tombstone_snapshot_test.go` | Ensures tombstone garbage collection respects snapshot visibility guarantees (MVCC safety). |

## Internal Tests

| Test File | Description |
|-----------|------------|
| `internal/cache/cache_test.go` | Tests LRU cache behavior: basic operations, overwrite handling, eviction policy, large item handling, and concurrency safety. |
| `internal/internal_key_test.go` | Validates internal key encoding/decoding, user key extraction, and comparator ordering semantics. |

## Iterator Tests

| Test File | Description |
|-----------|------------|
| `iterators/iterator_test.go` | Tests merging iterators (Memtable + SSTable) and version-based visibility filtering logic. |

## Manifest Tests

| Test File | Description |
|-----------|------------|
| `manifest/manifest_test.go` | Validates MANIFEST operations: append, decode, and corruption detection mechanisms. |

## Memtable Tests

| Test File | Description |
|-----------|------------|
| `memtable/memtable_test.go` | Tests memtable operations: insert, size tracking, internal key ordering, and tombstone handling. |
| `memtable/skiplist_test.go` | Validates skiplist behavior: insert, iteration, duplicate updates, and ordering guarantees. |

## SSTable Tests

| Test File | Description |
|-----------|------------|
| `sstable/block_test.go` | Tests SSTable block construction and in-block seek functionality. |
| `sstable/compression_test.go` | Validates compression and decompression primitives. |
| `sstable/filter_test.go` | Tests Bloom filter creation, membership checks, and SSTable integration. |
| `sstable/full_test.go` | Validates end-to-end SSTable build and read paths (Scan, Seek) including prefix compression validation. |
| `sstable/sstable_test.go` | Tests core SSTable iterator functionality. |


## WAL Tests

| Test File | Description |
|-----------|------------|
| `wal/record_test.go` | Tests WAL record batch encoding/decoding and CRC validation. |
| `wal/segment_test.go` | Validates WAL segment lifecycle: append, sync, open, and close operations. |
| `wal/truncation_test.go` | Tests WAL truncation logic and boundary handling. |
| `wal/wal_test.go` | Validates WAL log rotation and reopen semantics. |


## System & Crash Tests

| Test File | Description |
|-----------|------------|
| `tests/crash/crash_test.go` | Verifies data consistency after a simulated crash and restart. |
| `tests/crash/recovery_test.go` | Ensures deterministic recovery (identical state across repeated recoveries). |
| `tests/crash/truncation_test.go` | Validates truncation idempotence and data persistence after restart. |
| `tests/crash/wal_fsync_test.go` | Tests crash consistency behavior before WAL fsync (via external process simulation). |


## Determinism & Integration Tests

| Test File | Description |
|-----------|------------|
| `tests/determinism/replay_repeatability_test.go` | Ensures that replaying database operations produces identical results across runs. |
| `tests/integration/flush_main_test.go` | Tests automatic flush triggering based on key count or size thresholds. |
| `tests/integration/full_test.go` | Comprehensive integration test: Put, batch writes, deletes, and recovery. |
| `tests/integration/open_put_get_test.go` | Verifies persistence across Open → Put → Close → Open → Get cycle. |
| `tests/manifest_test.go` | Tests manifest compaction (rewrite) logic. |


---


## Summary

**Total Test Files :** 38<br>
**Test Categories :**<br>
- **Unit Tests** : Low-level tests for specific packages (engine, sstable, wal, etc.)
- **Integration Tests** : Ensuring components work together (engine level).
- **System & Crash/Recovery Tests** : Verifying system stability and data consistency after crashes.
- **Determinism & Integration Tests** : Validates deterministic behavior and thread-safety under concurrent execution.