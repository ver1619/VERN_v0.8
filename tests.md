# Tests v0.8

```
=== RUN   TestCompactionL0toL1
--- PASS: TestCompactionL0toL1 (0.24s)
=== RUN   TestCompactionL1ToL2
--- PASS: TestCompactionL1ToL2 (0.01s)
=== RUN   TestConcurrencySWMR
--- PASS: TestConcurrencySWMR (11.00s)
=== RUN   TestDBPutGetDelete
--- PASS: TestDBPutGetDelete (0.02s)
=== RUN   TestDBRecovery
--- PASS: TestDBRecovery (0.01s)
=== RUN   TestGetWithSnapshot
--- PASS: TestGetWithSnapshot (0.02s)
=== RUN   TestSnapshotDeleteVisibility
--- PASS: TestSnapshotDeleteVisibility (0.02s)
=== RUN   TestImmutableMemtableReadVisibility
--- PASS: TestImmutableMemtableReadVisibility (0.05s)
=== RUN   TestFlushManual
--- PASS: TestFlushManual (0.06s)
=== RUN   TestFlushRecovery
--- PASS: TestFlushRecovery (0.05s)
=== RUN   TestFullCycleFlushAndRead
--- PASS: TestFullCycleFlushAndRead (0.11s)
=== RUN   TestSnapshotIteratorStability
--- PASS: TestSnapshotIteratorStability (0.02s)
=== RUN   TestIteratorDeleteVisibility
--- PASS: TestIteratorDeleteVisibility (0.02s)
=== RUN   TestFullRecovery
--- PASS: TestFullRecovery (0.02s)
=== RUN   TestRangeScanSnapshot
--- PASS: TestRangeScanSnapshot (0.05s)
=== RUN   TestPrefixScan
--- PASS: TestPrefixScan (0.03s)
=== RUN   TestSnapshotCapturesSequence
--- PASS: TestSnapshotCapturesSequence (0.02s)
=== RUN   TestSnapshotIsImmutable
--- PASS: TestSnapshotIsImmutable (0.02s)
=== RUN   TestVersionSetReplay
--- PASS: TestVersionSetReplay (0.03s)
=== RUN   TestVersionSet_AddTable
--- PASS: TestVersionSet_AddTable (0.00s)
=== RUN   TestVersionSet_RemoveTable
--- PASS: TestVersionSet_RemoveTable (0.00s)
=== RUN   TestVersionSet_PickCompaction
--- PASS: TestVersionSet_PickCompaction (0.00s)
=== RUN   TestVersionSet_GetOverlappingInputs
--- PASS: TestVersionSet_GetOverlappingInputs (0.00s)
PASS
ok      vern_kv0.8/engine       11.831s
=== RUN   TestInternalKeyEncodeDecode
--- PASS: TestInternalKeyEncodeDecode (0.00s)
=== RUN   TestComparatorOrdering
--- PASS: TestComparatorOrdering (0.00s)
=== RUN   TestExtractHelpers
--- PASS: TestExtractHelpers (0.00s)
PASS
ok      vern_kv0.8/internal     0.018s
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
ok      vern_kv0.8/internal/cache       0.024s
=== RUN   TestMergeMemtableAndSSTable
--- PASS: TestMergeMemtableAndSSTable (0.02s)
=== RUN   TestVersionFilterIterator
--- PASS: TestVersionFilterIterator (0.00s)
=== RUN   TestVersionFilterHidesFutureWrites
--- PASS: TestVersionFilterHidesFutureWrites (0.00s)
PASS
ok      vern_kv0.8/iterators    0.025s
=== RUN   TestManifestAppendAndDecode
--- PASS: TestManifestAppendAndDecode (0.03s)
=== RUN   TestManifestCorruption
--- PASS: TestManifestCorruption (0.00s)
PASS
ok      vern_kv0.8/manifest     0.048s
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
--- PASS: TestSkiplistLargeInsert (0.01s)
=== RUN   TestRandomLevel
--- PASS: TestRandomLevel (0.00s)
PASS
ok      vern_kv0.8/memtable     0.025s
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
--- PASS: TestFilterIntegration (0.03s)
=== RUN   TestFullSSTable
--- PASS: TestFullSSTable (0.02s)
=== RUN   TestSSTableIterator
--- PASS: TestSSTableIterator (0.01s)
PASS
ok      vern_kv0.8/sstable      0.066s
=== RUN   TestManifestCompaction
    manifest_test.go:65: Initial Manifest Size: 0, Final: 20
--- PASS: TestManifestCompaction (0.13s)
PASS
ok      vern_kv0.8/tests        0.154s
=== RUN   TestCrashConsistency
--- PASS: TestCrashConsistency (0.54s)
=== RUN   TestRecoveryIsDeterministic
--- PASS: TestRecoveryIsDeterministic (0.04s)
=== RUN   TestTruncationIdempotence
--- PASS: TestTruncationIdempotence (0.02s)
=== RUN   TestCrashBeforeWALFsync
--- PASS: TestCrashBeforeWALFsync (0.69s)
PASS
ok      vern_kv0.8/tests/crash  1.309s
?       vern_kv0.8/tests/crash/helpers  
=== RUN   TestReplayRepeatability
--- PASS: TestReplayRepeatability (0.02s)
PASS
ok      vern_kv0.8/tests/determinism    0.028s
=== RUN   TestAutoFlushKeyTrigger
--- PASS: TestAutoFlushKeyTrigger (35.64s)
=== RUN   TestIntegrationBasic
--- PASS: TestIntegrationBasic (0.66s)
=== RUN   TestIntegrationCompactionSpaceReclamation
--- PASS: TestIntegrationCompactionSpaceReclamation (0.07s)
=== RUN   TestOpenPutGet
--- PASS: TestOpenPutGet (0.01s)
PASS
ok      vern_kv0.8/tests/integration    36.393s
=== RUN   TestEncodeDecodeRecord
--- PASS: TestEncodeDecodeRecord (0.00s)
=== RUN   TestCRCFailureStopsDecode
--- PASS: TestCRCFailureStopsDecode (0.00s)
=== RUN   TestPartialRecordFails
--- PASS: TestPartialRecordFails (0.00s)
=== RUN   TestSegmentAppendAndSync
--- PASS: TestSegmentAppendAndSync (0.01s)
=== RUN   TestSegmentReopenAppend
--- PASS: TestSegmentReopenAppend (0.01s)
=== RUN   TestAppendAfterCloseFails
--- PASS: TestAppendAfterCloseFails (0.01s)
=== RUN   TestWALTruncation
--- PASS: TestWALTruncation (0.02s)
=== RUN   TestWALAppendAndRotate
--- PASS: TestWALAppendAndRotate (0.07s)
=== RUN   TestWALReopen
--- PASS: TestWALReopen (0.01s)
=== RUN   TestWALCloseClosesAllSegments
--- PASS: TestWALCloseClosesAllSegments (0.01s)
PASS
ok      vern_kv0.8/wal  0.156s
```