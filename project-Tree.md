## Project Tree (VERN_v0.8)

Total Files : 85<br>
Total Code Files : 75<br>
Total Test Files : 38<br>
Total Source Files : 37<br>
Documentation and others : 10<br>

```
├── 📁 cmd
│   └── 📁 vern-cli
│       └── 📄 main.go
├── 📁 engine
│   ├── 📄 compaction.go
│   ├── 📄 compaction_test.go
│   ├── 📄 compaction_tiered_test.go
│   ├── 📄 concurrency_test.go
│   ├── 📄 config.go
│   ├── 📄 db.go
│   ├── 📄 db_test.go
│   ├── 📄 flush.go
│   ├── 📄 flush_test.go
│   ├── 📄 full_cycle_test.go
│   ├── 📄 iterator.go
│   ├── 📄 iterator_test.go
│   ├── 📄 manifest_replay.go
│   ├── 📄 recovery_paging_test.go
│   ├── 📄 recovery.go
│   ├── 📄 recovery_test.go
│   ├── 📄 scan_iterator.go
│   ├── 📄 scan_iterator_test.go
│   ├── 📄 snapshot.go
│   ├── 📄 snapshot_test.go
│   ├── 📄 tombstone_snapshot_test.go
│   ├── 📄 version_set.go
│   └── 📄 version_set_test.go
├── 📁 internal
│   ├── 📁 cache
│   │   ├── 📄 cache.go
│   │   ├── 📄 cache_test.go
│   │   └── 📄 lru.go
│   ├── 📄 comparator.go
│   ├── 📄 internal_key.go
│   └── 📄 internal_key_test.go
├── 📁 iterators
│   ├── 📄 iterator.go
│   ├── 📄 iterator_test.go
│   ├── 📄 memtable_iterator.go
│   ├── 📄 merge_iterator.go
│   └── 📄 version_filter_iterator.go
├── 📁 manifest
│   ├── 📄 assertions.go
│   ├── 📄 constants.go
│   ├── 📄 manifest.go
│   ├── 📄 manifest_test.go
│   └── 📄 record.go
├── 📁 memtable
│   ├── 📄 memtable.go
│   ├── 📄 memtable_test.go
│   ├── 📄 skiplist.go
│   └── 📄 skiplist_test.go
├── 📁 sstable
│   ├── 📄 block.go
│   ├── 📄 block_test.go
│   ├── 📄 builder.go
│   ├── 📄 compression.go
│   ├── 📄 compression_test.go
│   ├── 📄 filter.go
│   ├── 📄 filter_test.go
│   ├── 📄 full_test.go
│   ├── 📄 iterator.go
│   ├── 📄 reader.go
│   ├── 📄 sstable_test.go
│   └── 📄 table.go
├── 📁 tests
│   ├── 📁 crash
│   │   ├── 📁 helpers
│   │   │   └── 📄 crash_main.go
│   │   ├── 📄 crash_test.go
│   │   ├── 📄 recovery_test.go
│   │   ├── 📄 truncation_test.go
│   │   └── 📄 wal_fsync_test.go
│   ├── 📁 determinism
│   │   └── 📄 replay_repeatability_test.go
│   ├── 📁 integration
│   │   ├── 📄 flush_main_test.go
│   │   ├── 📄 full_test.go
│   │   └── 📄 open_put_get_test.go
│   └── 📄 manifest_test.go
├── 📁 wal
│   ├── 📄 record.go
│   ├── 📄 record_test.go
│   ├── 📄 segment.go
│   ├── 📄 segment_test.go
│   ├── 📄 truncation.go
│   ├── 📄 truncation_test.go
│   ├── 📄 wal.go
│   └── 📄 wal_test.go
├── ⚙️ .gitignore
├── 📝 Architecture.md
├── 📝 CLI.md
├── 📄 go.mod
├── 📄 go.sum
├── 📄 Invariants.md
├── 🪪 LICENSE
├── 🌳 project-Tree.md
├── 📝 README.md
└── 🧪 tests.md

```
