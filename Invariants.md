# VernKV v0.8 — Invariants

This document defines the core invariants that the storage engine must maintain for correctness, durability, and consistency. 

---

## Sequence Number 

**1.** Decide which version of a key is the newest and ensure correct recovery after crashes.<br>
**2.** Each `Put` or `Delete` operation receives a globally unique and monotonically increasing `64-bit sequence number`. No two writes ever share the same sequence.<br>
**3.** Sequence numbers use the `lower 56 bits` of the trailer, with the `upper 8 bits` reserved for record type **(Value or Tombstone)**.<br>
**4.** The cutoff sequence **(used for safe WAL truncation)** increases and never decreases. <br>
**5.** The sequence number must be assigned before WAL append, and must never be reused even if the write fails.<br>

##  Write Path

**1.** **Only one write happens at a time**.<br>
A lock is used so that two writes cannot modify the database at the same time. This prevents data corruption.<br>
**2.** **Every write is first recorded in the WAL (Write-Ahead Log)**.<br>
Before updating memory, the database writes the changes to a log file(WAL). This ensures the data can be recovered if the system crashes.<br>
**3.** **No physical deletion of data happens**<br>
Deletions are handled via `tombstone` records rather than physical modification (no in-place mutation).<br>
**4.** A write is acknowledged to the client only after WAL fsync.

## Internal Key

**1.** Each internal key consists of the user key followed by an `8-byte trailer` containing the sequence number and record type.<br>
**2.** Keys are sorted by `user key (ascending)`, then `sequence number (descending)` so newer versions appear first.<br>
**3.** Record types are strictly limited to `Value` or `Tombstone` types; others cause validation errors.<br>


## WAL

**1.** WAL is strictly `append-only`. New writes are always append at the end.<br>
**2.** Multiple writes are grouped into `batches`(treated as one complete unit).<br>
**3.** If a batch is corrupted, the entire batch is ignored. **Partial writes are not accepted**.<br>
**4.** The WAL is deleted or truncated only after data is safely saved to disk(**SSTable**).<br>
**5.** Recovery replays records until the first corruption boundary; remaining log is discarded.<br>
**6.** Memtable is built only after **writes are successfully written to WAL**.<br>

## Memtable

**1.** Memtable is an `in-memory state`.<br>
**2.** The memtable uses a `Skiplist` structure to maintain keys in sorted order.<br>
**3.** Access follows a `single-writer`, `multiple-reader` model with appropriate locking.<br>
**4.** Active memtables are rotated to immutable status when they exceed the size limit.<br>
**5.** **Immutable memtables** are read-only states<br>
**6.** Memtable visibility is governed by snapshot sequence numbers.<br>
**7.** Immutable memtables are flushed to disk<br>

## Flush

**1.** Flush operations are serialized so only one runs at a time.<br>
**2.** New SSTables from flushes are always placed at `Level 0`.<br>
**3.** Flushing follows a strict order: **write SSTable, fsync SSTable, update manifest,fsync mainfest, version installation, update safe WAL cutoff, truncate old WAL**.<br>
**4.** Manifest must be durable before WAL truncation.<br>
**5.** Any flush failure halts all future writes without discarding the pending data.<br>

## SSTable

**1.** SSTable files are `immutable` once written and only deleted when obsolete.<br>
**2.** SSTable file numbers are globally unique, strictly increasing, and never reused.<br>
**3.** The file layout includes **data blocks, filter blocks, index blocks, and a fixed-size footer**.<br>
**4.** `Data blocks` contain restart points to enable efficient binary search within the block.<br>
**5.** `Compression` and `bloom filters` are applied to optimize storage and read performance.<br>

## Bloom Filter

**1.** Bloom filters may produce false positives.<br>
**2.** Bloom filters must never produce false negatives.<br>
**3.** A negative bloom filter result guarantees the key is not present in that file.<br>

## Manifest

**1.** The Manifest is the **single source of truth for the database state**.<br>
**2.** The file is append-only with CRC protection for each record.<br>
**3.** Recovery consists of replaying the Manifest to reconstruct the `version set`.<br>
**4.** The Manifest is periodically compacted to a clean snapshot to prevent unbound growth.<br>

## VersionSet & Level

**1.** The engine maintains rigid layering (typically `7 levels`).<br>
**2.** Level 0 allows overlapping key ranges, while higher levels must be strictly sorted with no overlap.<br>
**3.** Reads merge data across all levels, **prioritizing memtables and newer SSTables**.<br>
**4.** Queries skip SSTables that cannot contain the target key based on `range checks` and `bloom filters`.<br>

## Snapshot Isolation 

**1.** A snapshot captures a fixed sequence number at creation.<br>
**2.** All reads under that snapshot must use that captured sequence.<br>
**3.** Writes occurring after snapshot creation must not affect snapshot reads.<br>
**4.** Snapshot reads must be repeatable and deterministic.<br>

## Tombstone Shadowing 

**1.** If the first visible entry for a key is a `Tombstone`, the key is considered deleted.<br>
**2.** Older values for that key must not be returned.<br>
**3.** Tombstones must shadow all lower-level entries with smaller sequence numbers.<br>

## Compaction Rules

**1.** **Only one compaction runs at a time** to avoid conflicts.<br>
**2.** Level 0 compaction merges all its files with overlapping Level 1 files.<br>
**3.** Higher-level compaction merges one file with overlapping files in the next level.<br>
**4.** `Tombstones` are removed only when no older versions exist in deeper levels.<br>
**5.** Compaction produces new SSTables. A single manifest record atomically adds new files and removes obsolete files. Readers must never observe a partial compaction state.<br>
**6.** Old files must not be physically deleted until no active version references them.<br>


## Recovery Rules

**1.** Recovery proceeds by replaying the Manifest first, then the WAL.<br>
**2.** The engine checks the largest sequence number and file number used before the crash and continues from the next number.<br>
**3.** At startup, the system verifies that every SSTable listed in manifest is actually present on disk.<br>

## Garbage Collection Rules

**1.** When an SSTable becomes obsolete, it is first logically marked as deleted.<br>
**2.** Logically marked SSTables are tracked before being physically deleted.<br>
**3.** Files must not be physically deleted while referenced by any active Version or snapshot.<br>
**4.** Files are only removed after successful filesystem deletion.<br>
**5.** Cleanup operations occur after flush and compaction cycles to avoid blocking writes.<br>

## Configuration

**1.** Once the in-memory table(Memtable) reaches a size limit, it is flushed and replaced with a new one.<br>
**2.** If L0 reaches a file-count threshold, compaction is triggered.<br>
**3.** Writes are synchronized to disk by default to ensure durability.<br>
**4.** Bloom filters and caches are configured to balance performance and memory usage.


