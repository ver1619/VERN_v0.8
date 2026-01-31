**Project Tree**

```
├── engine
│   ├── bootstrap.go
│   ├── compaction.go
│   ├── compaction_test.go
│   ├── concurrency_test.go
│   ├── db.go
│   ├── db_test.go
│   ├── flush.go
│   ├── flush_test.go
│   ├── full_cycle_test.go
│   ├── iterator.go
│   ├── iterator_test.go
│   ├── manifest_replay.go
│   ├── recovery.go
│   ├── recovery_test.go
│   ├── scan_iterator.go
│   ├── scan_iterator_test.go
│   ├── snapshot.go
│   ├── snapshot_test.go
│   ├── version_set.go
│   └── version_set_test.go
├── internal
│   ├── comparator.go
│   ├── internal_key.go
│   └── internal_key_test.go
├── iterators
│   ├── iterator.go
│   ├── iterator_test.go
│   ├── memtable_iterator.go
│   ├── merge_iterator.go
│   └── version_filter_iterator.go
├── manifest
│   ├── assertions.go
│   ├── constants.go
│   ├── manifest.go
│   ├── manifest_test.go
│   └── record.go
├── memtable
│   ├── memtable.go
│   └── memtable_test.go
├── sstable
│   ├── block.go
│   ├── block_test.go
│   ├── builder.go
│   ├── filter.go
│   ├── filter_test.go
│   ├── full_test.go
│   ├── iterator.go
│   ├── reader.go
│   ├── sstable_test.go
│   └── table.go
├── tests
│   ├── crash
│   │   ├── helpers
│   │   │   └── crash_main.go
│   │   ├── crash_test.go
│   │   ├── recovery_test.go
│   │   ├── truncation_test.go
│   │   └── wal_fsync_test.go
│   ├── determinism
│   │   └── replay_repeatability_test.go
│   └── integration
│       ├── full_test.go
│       └── open_put_get_test.go
├── wal
│   ├── crash.go
│   ├── record.go
│   ├── record_test.go
│   ├── segment.go
│   ├── segment_test.go
│   ├── truncation.go
│   ├── truncation_test.go
│   ├── wal.go
│   └── wal_test.go
├── LICENSE
├── README.md
└── go.mod
```

