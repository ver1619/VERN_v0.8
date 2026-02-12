# VernKV v0.8 — Invariants

This document defines the core invariants that the storage engine must maintain for correctness, durability, and consistency. Each invariant is derived directly from the codebase.

---

## 1. Sequence Number Invariants

| ID  | Invariant | Source |
|-----|-----------|--------|
| S1  | Every write is assigned a globally unique, **monotonically increasing** 64-bit sequence number. | `db.go:Put`, `db.go:Write` |
| S2  | Sequence numbers occupy the **lower 56 bits** of the trailer; the upper 8 bits are reserved for the record type. Sequences exceeding 56 bits cause a panic. | `internal_key.go:EncodeInternalKey` |
| S3  | `nextSeq` is incremented **after** the memtable insert, never before. A crash mid-write cannot leave `nextSeq` pointing ahead of data. | `db.go:Put` (line 196), `db.go:Write` (line 237) |
| S4  | `WALCutoffSeq` is **monotonically non-decreasing**. It is only updated when the new value is strictly greater. | `version_set.go:SetWALCutoff` |

---

## 2. Write Path Invariants

| ID  | Invariant | Source |
|-----|-----------|--------|
| W1  | The write mutex (`DB.mu`) is held for the **entire** write path: sequence assignment → WAL append → memtable insert → seq increment. | `db.go:Put`, `db.go:Delete`, `db.go:Write` |
| W2  | **WAL-Before-Memtable**: Data is appended to the WAL **before** being inserted into the memtable. | `db.go:Put` (lines 179–189) |
| W3  | **Sync-Before-Ack** (when `SyncWrites=true`): `wal.Sync()` (fsync) is called before the memtable insert, guaranteeing durability before acknowledgement. | `db.go:Put` (lines 182–186) |
| W4  | **No in-place mutation**: Deletes are stored as tombstone records, never as physical deletions. | `Architecture.md`, `db.go:Delete` |
| W5  | Background errors **block all future writes**. `checkBackgroundError()` is called at the top of every write path. | `db.go:Put`, `db.go:Delete`, `db.go:Write` |

---

## 3. Internal Key Invariants

| ID  | Invariant | Source |
|-----|-----------|--------|
| K1  | An internal key is `[UserKey | Trailer(8B)]` where `Trailer = (SeqNum << 8) \| RecordType`. | `internal_key.go:EncodeInternalKey` |
| K2  | **Sort order** is `(UserKey ASC, SeqNum DESC, RecordType ASC)`. For the same user key, newer versions always appear first. | `comparator.go:Compare` |
| K3  | `RecordType` can only be `VALUE (0x01)` or `TOMBSTONE (0x02)`. Any other value causes a decode error. | `internal_key.go:DecodeInternalKey` |

---

## 4. Memtable Invariants

| ID  | Invariant | Source |
|-----|-----------|--------|
| M1  | The memtable is backed by a **Skiplist** with max level 12 and promotion probability 0.5. | `skiplist.go` |
| M2  | Keys are maintained in **sorted order** per the `Comparator` (K2). | `skiplist.go:Insert` |
| M3  | **Single writer, multiple readers**: Writes acquire `Memtable.mu.Lock()`; reads (iterators, size queries) acquire `Memtable.mu.RLock()`. | `memtable.go:Insert`, `memtable.go:Iterator` |
| M4  | When `ApproximateSize() >= MemtableSizeLimit` (default 4 MB), the active memtable is **rotated** (frozen) and a flush is scheduled. | `db.go:Put` (lines 191–194) |
| M5  | **Immutability after rotation**: Once a memtable is moved to `db.immutables`, no further inserts can occur because a fresh memtable replaces it. | `db.go:rotateMemtableLocked` |
| M6  | Immutable memtables are flushed **in FIFO order** (oldest first): `db.immutables[0]` is always flushed next. | `db.go:MaybeScheduleFlush` (line 500) |

---

## 5. WAL Invariants

| ID  | Invariant | Source |
|-----|-----------|--------|
| A1  | The WAL is **append-only**. Active segments are never modified after writing. | `Architecture.md` |
| A2  | **Record frame format**: `[CRC32 (4B) \| Length (4B) \| Header (16B) \| Payload (NB)]`. | `Architecture.md` |
| A3  | **Batch atomicity**: A batch is either fully replayed or fully ignored. CRC mismatch → discard entire record. | `Architecture.md`, `recovery.go` (line 74–76) |
| A4  | **Replay cutoff**: During recovery, batches where `batchMaxSeq <= WALCutoffSeq` are **skipped**. Only unflushed data is replayed into the memtable. | `recovery.go` (lines 80–83) |
| A5  | **Replay stops on corruption**: If `DecodeRecord` fails, replay halts — earlier valid data remains intact, later corrupt data is discarded. | `recovery.go` (lines 74–76) |
| A6  | WAL segments are sorted lexicographically by filename and replayed **in order**. | `recovery.go` (line 62) |
| A7  | WAL truncation only occurs **after** a successful flush commits its `LargestSeq` as the new cutoff. | `db.go:MaybeScheduleFlush` (lines 554–558) |

---

## 6. Flush Invariants

| ID  | Invariant | Source |
|-----|-----------|--------|
| F1  | Flushes are serialized: only **one flush runs at a time** (`flushMu`). | `db.go:MaybeScheduleFlush` (line 484) |
| F2  | Flushed SSTables are **always placed at Level 0**. | `flush.go` (line 82) |
| F3  | **Flush ordering**: Manifest AddSSTable record → Manifest SetWALCutoff record → remove from `db.immutables` → WAL truncation. | `db.go:MaybeScheduleFlush` (lines 516–558) |
| F4  | A flush failure sets a **background error**, halting all future writes. The immutable memtable is **not** discarded on failure. | `db.go:MaybeScheduleFlush` (lines 508–510) |
| F5  | File number allocation (`nextFileNum++`) happens under `DB.mu`, ensuring unique file numbers. | `db.go:MaybeScheduleFlush` (lines 502–503) |

---

## 7. SSTable Invariants

| ID  | Invariant | Source |
|-----|-----------|--------|
| T1  | Once written, an `.sst` file is **never modified**. It is only deleted when obsolete. | Design principle |
| T2  | **File layout**: `[DataBlock₁..DataBlockₙ \| FilterBlock \| MetaIndexBlock \| IndexBlock \| Footer]`. | `builder.go:Close` |
| T3  | Data blocks are flushed when `CurrentSize() >= blockSize` (default 4 KB). | `builder.go:Add` (line 76) |
| T4  | Each data block contains **restart points** every 16 entries to enable binary search within the block. | `block.go` (line 19, `restartInterval = 16`) |
| T5  | Block integrity: blocks with `< 4 bytes` are rejected as corrupt (`ErrBlockCorrupt`). | `block.go:NewBlockIterator` (line 91) |
| T6  | The **Footer** is a fixed-size structure at the end of the file containing handles (offset + length) to the MetaIndex and Index blocks. | `builder.go:Close` (lines 236–245) |
| T7  | **Bloom filter**: A table-wide bloom filter (10 bits/key) is stored in the FilterBlock and indexed via the MetaIndexBlock. | `builder.go` (line 45), `filter.go` |
| T8  | Compression is applied **per-block**. Zlib compression is only used if it reduces size by ≥ 2 bytes; otherwise the block is stored uncompressed. | `builder.go:Flush` (lines 96–108) |

---

## 8. Manifest Invariants

| ID  | Invariant | Source |
|-----|-----------|--------|
| N1  | The Manifest is the **single source of truth**. If an SSTable is not in the Manifest, it does not exist in the database. | `Architecture.md` |
| N2  | The Manifest is **append-only**. Records are appended and `fsync`'d; there is no in-place update. | `Architecture.md` |
| N3  | Record types: `ADD_SSTABLE (0x01)`, `REMOVE_SSTABLE (0x02)`, `SET_WAL_CUTOFF (0x03)`. | `Architecture.md` |
| N4  | Each record is **CRC-protected** and must be `fsync`'d before it is considered visible. | `Architecture.md` |
| N5  | **Recovery = Manifest replay**: On startup, the Manifest is replayed to reconstruct the `VersionSet`. | `recovery.go:Recover`, `manifest_replay.go` |
| N6  | **Manifest compaction** rewrites the manifest as a clean snapshot of the current state under `DB.mu`. If rewrite fails, the old manifest is re-opened. | `db.go:CompactManifest` |

---

## 9. VersionSet & Level Invariants

| ID  | Invariant | Source |
|-----|-----------|--------|
| V1  | The engine has **7 levels** (L0–L6). `NumLevels = 7`. | `version_set.go` (line 10) |
| V2  | **L0 allows overlap**: SSTables in L0 may have overlapping key ranges. | `version_set.go` (line 25, comment) |
| V3  | **L1+ are sorted**: SSTables in levels ≥ 1 are sorted by `SmallestKey` and must **not** have overlapping key ranges. | `version_set.go:AddTable` (lines 50–54) |
| V4  | Reads merge across **all levels**: Active memtable → immutables → SSTables (sorted by FileNum DESC for recency). | `db.go:GetWithOptions` |
| V5  | For point reads, SSTables whose key range does not overlap with the target key are **skipped** (bloom filter + key range check). | `db.go:GetWithOptions` (lines 311–317) |
| V6  | Level is invalid if `>= NumLevels`; `AddTable` returns an error. | `version_set.go:AddTable` (line 43) |

---

## 10. Compaction Invariants

| ID  | Invariant | Source |
|-----|-----------|--------|
| C1  | Compaction is serialized: only **one compaction runs at a time** (`compactionMu`). | `db.go:MaybeScheduleCompaction` (line 202) |
| C2  | **L0 compaction** merges **all L0 files with all L1 files** into a single output at L1. | `compaction.go` (lines 30–40) |
| C3  | **L1+ compaction** picks the first file in the level and merges it with overlapping files from the next level. | `compaction.go` (lines 42–53) |
| C4  | **Tombstone garbage collection**: Tombstones are dropped during compaction **only** if no deeper levels contain data (i.e., the target level is the bottommost). | `compaction.go` (lines 100–112) |
| C5  | Compaction **input deletion** is manifest-logged (REMOVE_SSTABLE) **before** the output addition (ADD_SSTABLE). | `compaction.go` (lines 170–198) |
| C6  | **Compaction trigger (L0)**: Triggered when `len(L0) / L0CompactionTrigger >= 1.0` (default trigger = 4 files). | `version_set.go:PickCompaction`, `config.go` |
| C7  | **Compaction trigger (L1+)**: Triggered when total level size exceeds `L1MaxBytes * 2^(level-1)`. | `version_set.go:PickCompaction` (line 115) |
| C8  | Cannot compact the **max level** (L6). Attempting to do so returns an error. | `compaction.go` (line 22) |

---

## 11. Recovery Invariants

| ID  | Invariant | Source |
|-----|-----------|--------|
| R1  | Recovery order: **Manifest replay → WAL replay**. The VersionSet is rebuilt first, then unflushed WAL data is replayed into a fresh memtable. | `recovery.go:Recover` |
| R2  | `NextSeq` after recovery is `maxSeq + 1`, where `maxSeq` is the highest sequence seen across all SSTables and replayed WAL batches. | `recovery.go` (line 108) |
| R3  | `NextFileNum` after recovery is `maxFileNum + 1` (from existing SSTables). | `db.go:Open` (line 106) |
| R4  | On startup, the **existence** of every SSTable referenced by the VersionSet is verified. A missing file is a **consistency error**. | `db.go:Open` (lines 114–120) |

---

## 12. Concurrency Invariants

| ID  | Invariant | Source |
|-----|-----------|--------|
| X1  | **Lock hierarchy** (acquire in this order): `DB.mu` → `flushMu` → `compactionMu`. Never acquire in reverse. | `db.go`, `compaction.go` |
| X2  | `DB.mu` is held as a **write lock** for mutations and as a **read lock** for reads/iterator creation. | `db.go:Put`, `db.go:Get` |
| X3  | Flush and compaction release `DB.mu` during I/O-heavy work (SSTable writing) and re-acquire it only for metadata commits. | `db.go:MaybeScheduleFlush` (lines 504–514), `compaction.go` (lines 58–155) |
| X4  | `VersionSet.mu` provides fine-grained locking for level metadata, separate from `DB.mu`. | `version_set.go` |
| X5  | **Snapshot isolation**: `GetSnapshot()` captures `nextSeq - 1` under `RLock`, guaranteeing a consistent read point. | `db.go:GetSnapshot` |

---

## 13. Garbage Collection Invariants

| ID  | Invariant | Source |
|-----|-----------|--------|
| G1  | Removed SSTables are first tracked in `VersionSet.Obsolete`; physical deletion happens asynchronously in `cleanupObsoleteFiles()`. | `version_set.go:RemoveTable`, `db.go:cleanupObsoleteFiles` |
| G2  | An obsolete file is removed from the `Obsolete` map **only after** successful `os.Remove` (or if the file is already gone). | `db.go:cleanupObsoleteFiles` (lines 150–155) |
| G3  | Cleanup runs **after flush and compaction**, not during writes. | `db.go:MaybeScheduleFlush` (line 565) |

---

## 14. Configuration Defaults

| Parameter | Default | Invariant Implication |
|-----------|---------|----------------------|
| `MemtableSizeLimit` | 4 MB | Memtable rotates at this threshold |
| `BlockSize` | 4 KB | SSTable data block flush threshold |
| `L0CompactionTrigger` | 4 files | L0 compaction fires at ≥ 4 files |
| `L1MaxBytes` | 64 MB | L1 size budget; each deeper level is 2× larger |
| `SyncWrites` | `true` | Every write is fsync'd (W3) |
| `WAL Segment Size` | 64 MB | Max WAL segment before rotation |
| `LRU Cache` | 8 MB | Block cache for SSTable reads |
| `Bloom Filter` | 10 bits/key | False positive rate ~1% |
