# High-Level Overview

## 1. Write Path (Put / Delete)

* Write(Batch)
* Single writer thread
* Each write is assigned monotonically increasing seq.number
* Each Wal entry is encoded  `[length | payload | checksum]`, stored in bytes
* Wal Append
* fsync (done before Ack)
* Memtable Insert

```python
Client
  |
  | Put / Delete
  ▼
Write(Batch)
  |
  | acquire write mutex (single writer)
  ▼
Assign Sequence Numbers
  |
  ▼
Encode WAL Batch (Batch + Frame Record + CRC)
  |
  ▼
Append to WAL (active segment)
  |
  ▼
fsync(WAL)   <-- DURABILITY BOUNDARY
  |
  ▼
Apply to Memtable
  |
  ▼
Release write mutex
  |
  ▼
Return success(Ack) to client
```

--- 

## 2. Internal key-value Semantics

Rules:
` (key ASC, seq DESC)`

- No in-place deletion is made
- Deletes are respected, stored as TOMBSTONES

```python
PUT(key, value) ->
Insert InternalKey(key, seq, VALUE)

DELETE(key) ->
Insert InternalKey(key, seq, TOMBSTONE)

```

---

## 3. WAL

**a) Record Format**
```go
+----------------+----------------+----------------+----------------+
| CRC32 (4B)     | Length (4B)    | Header (16B)   | Payload (N)    |
+----------------+----------------+----------------+----------------+
```

**b) Payload Format**<br>
Payload is a concatenation of logical records
```go
Record_1 | Record_2 | ... | Record_N
```

**c) Header-Fixed Size(16 bytes)**
```go
+----------------+----------------+----------------+
| Type (1B)      | Flags (1B)     | Reserved (2B)  |
+----------------+----------------+----------------+
| SeqStart (8B)                                    |
+--------------------------------------------------+
| Count (4B)                                       |
+--------------------------------------------------+
```

**d) Logical Record Format (Inside Payload)**
```go
+----------------+----------------+----------------+
| KeyLen (4B)    | ValLen (4B)    | Type (1B)      |
+----------------+----------------+----------------+
| Key (K bytes)                                    |
+--------------------------------------------------+
| Value (V bytes)  (Zero for tombstone)            |
+--------------------------------------------------+
```

**e) Record Type**<br>
```go
VALUE = 0x01
TOMBSTONE = 0x02
```

### WAL Record Framing
```python
Put / Delete request
        │
        ▼
Assign sequence number
        │
        ▼
Construct internal key
(user key + seq + type)
        │
        ▼
Encode value (or tombstone)
        │
        ▼
Build WAL payload
(internal key + value)
        │
        ▼
Compute record length
        │
        ▼
Compute checksum (CRC)
        │
        ▼
Assemble WAL record frame
[length | payload | checksum]
        │
        ▼
Write is now framed
```

**Atomicity Rules**<br>
A batch is:
- Fully replayed Or fully ignored
- Partial batches are never applied

How This Is Enforced?
- CRC covers entire batch
- Count and SeqStart are validated
- Any mismatch → discard entire record<br>

### WAL Durability
```python
Append record to WAL buffer
        │
        ▼
Write bytes to OS page cache
        │
        ▼
      fsync
        │
        ▼
OS guarantees persistence
        │
        ▼
Write is now durable
```

**Failure Scenarios**<br>
Case 1: **Crash mid-record**<br>
- Length read but payload incomplete
- CRC mismatch
- Replay stops safely<br>

Case 2: **Crash after fsync**<br>
- Record is valid
- CRC passes
- Batch is replayed fully<br>

Case 3: **Bit corruption**<br>
- CRC mismatch
- Replay stops
- Earlier data remains valid

### WAL Manager

The **WAL Manager** is the durability gatekeeper.<br>
It controls:<br>
- Write atomicity
- Durability boundary
- Crash recovery replay
- WAL lifecycle<br>

**How muliple WAL segments look?**
```go
wal/
├── wal_000001.log
├── wal_000002.log
├── wal_000003.log  ← active
```

**Each WAL record**<br>
`[ Length | Payload | CRC ]`

Each batch:<br>
- Has a starting sequence number
- Contains multiple logical records
- Is atomic

**Responsibilties**<br>
- Assign sequence ranges
- Encode atomic batches
- Append to active segment
- fsync before acknowledgment
- Rotate segments
- Support replay
- Support truncation

---

## 4. Memtable

The **Memtable** is the **in-memory data structure** that stores the most recently written data.<br>

Each memtable entry is stored as `InternalKey(user_key, seq, type)`, ordered by comparator. 

**Memtable Lifecycle**<br>
```python
Active Memtable
      ↓ size threshold reached
Freeze
      ↓
Immutable Memtable
      ↓
Flush to SSTable
```

**Mematble Freeze**<br>
A memtable has a fixed size. When it exceeds this size, it is frozen and scheduled for flush.<br>

**Action**
- Memtable is marked as immutable
- Immutable memtable is scheduled for flush to SSTable
- A new memtable is created
- New memtable continue accepting writes immediately 

---

## 5. Manifest
The **Manifest** is the **single authoritative metadata log** of the entire storage engine.<br>

It defines:
- Which SSTables exist
- Which levels they belong to
- Sequence number boundaries
- WAL truncation cutoffs<br>

>If something is not in the Manifest, it is not part of the database.<br>

**Internal Structure**<br>
Manifest is an append-only log:<br>

```python
MANIFEST
┌─────────────────────────────┐
│ Record 1 (Add SSTable)      │
│ Record 2 (Add SSTable)      │
│ Record 3 (Remove SSTable)   │
│ Record 4 (Set WAL Cutoff)   │
│ ...                         │
└─────────────────────────────┘
```

**Record Type**<br>

```go
ADD_SSTABLE = 0x01
REMOVE_SSTABLE = 0x02
SET_WAL_CUTOFF = 0x03
```

**Each record:**<br>
- Has a type
- Has a payload
- Is CRC-protected
- `[CRC | Length | Type | Payload]`
- Is fsync’d before becoming visible<br>

There is no in-place update.

**Responsibilities:**<br>
- Persist VersionSet evolution
- Act as recovery authority
- Define WAL truncation safety
- Enable deterministic replay
- Serialize lifecycle transitions (flush, compaction)


**Interaction**<br>

```python
Flush / Compaction
   |
   v
Write SSTable
   |
   v
Append MANIFEST record
   |
   v
fsync(MANIFEST)
   |
   v
SSTable becomes visible
```

**Recovery**<br>
`Replay MANIFEST → rebuild VersionSet`


**Working**<br>
```python
1. Open MANIFEST
2. Sequentially read records
3. Validate CRC
4. Apply to VersionSet:
        ADD_SSTABLE → insert file meta
        REMOVE_SSTABLE → delete file meta
        SET_WAL_CUTOFF → update cutoff
5. Stop at first invalid record
```

--- 

## 6. SSTable

SSTables are **persistent storage units** of the storage engine.<br>

SSTable write path begins when:<br>

`Active Memtable → Frozen (immutable)`<br>

and a background flush worker processes it.

**SSTables are:**<br>
- File on disk<br>
- Immutable<br>
- Sorted by InternalKey `(key ASC, seq DESC, type)`<br>
- Block based, CRC protected<br>
- Written once, never modified after creation<br>

**Flush Entry**<br>

```go
Immutable Memtable
        │
        ▼
Flush Worker
        │
        ▼
New SSTable Builder
```

**Responsibilities of flush worker:**<br>
- Create temporary SSTable file
- Flush sorted entries of immutable memtable
- Build Data blocks
- Build index blocks
- Build footer
- Data is written in sorted order in the temp file
- fsync(Saved to disk)
- Atomic Rename `.tmp -> .sst`
- Update MANIFEST (e.g, `ADD SSTABLE`)

```go
Write to temp file: 000123.sst.tmp
        │
        ▼
fsync(file)
        │
        ▼
rename → 000123.sst
        │
        ▼
Update MANIFEST
        │
        ▼
fsync(MANIFEST)
```

**Invariant:**<br>
1. A new SSTable is never written over an existing SSTable.<br>
2. Data is written to a temporary file<br>
3. The file is fully written and saved on disk.<br>
4. Only then the file is renamed to .sst<br>


**SSTABLE File Layout**<br>

```go
+-----------------------+
| Data Block 1          |
+-----------------------+
| Data Block 2          |
+-----------------------+
| ...                   |
+-----------------------+
| Data Block N          |
+-----------------------+
| Index Block           |
+-----------------------+
| Filter Block          |
+-----------------------+
| Metadata Block        |
+-----------------------+
| Footer                |
+-----------------------+
```

All blocks are written sequentially<br>

### Data Block<br>

- Small chunck of sorted ky-value pairs inside an SSTable.<br>
- SSTable contain multiple data blocks.<br>
- Typical size : 4KB - 32KB<br>
- Instead of reading the whole SSTable, the DB reads one block from disk (Read 1 block -> multiple keys loaded).<br>
- Blocks have fixed length<br>
- When a block is full, a new block is created.<br>

**Benefits:**<br>
- Faster reads<br>
- Can cache blocks in RAM<br>
- Reduced disk I/O<br>

**Data Block Layout**<br>
```python
[key ,values] -> contains many key-value pairs stored in sorted order.
[restart points] -> helps the DB jump close to the correct place when searching
[restart count] -> reader uses this to understand the restart list
[checksum]
```


### Index Block<br>

GOAL : Locate the correct data block quickly without scanning the entire SSTable.<br>
- The index block is a map of all data blocks in the SSTable.<br>
- It tells the database which data block to open when searching for a key.<br>

**Index Block Layout**<br>
```python
[last/first key of the block] [block offset] [block size]
```

### Footer

- The footer is the last part of the SSTable file.<br>
- Located at the end of the file.<br>
- It tells the database where the important metadata blocks are located.<br>

**When SSTable is opened, The DB:**<br>
1. Jumps to end of the file<br>
2. Reads the footer<br>
3. Uses the footer to locate the index block<br>
4. Uses the index block to locate the data block<br>
5. Reads the data block

**What footer stores?**<br>
Footer stores two block locations:<br>
1. Index block location<br>
2. Metaindex block location<br>

**Footer Layout**<br>
```go
--------------------------------
Index block offset
Index block size
--------------------------------
Metaindex block offset
Metaindex block size
--------------------------------
Magic number
--------------------------------
```

**Magic Number**<br>
Fixed constant at file end used to verify the SSTable file type and integrity.<br>

**How it works**<br>
When reading an SSTable:

- DB jumps to end of file
- Reads the magic number
- Checks if it matches the expected value
- If it does not match → file is rejected.<br>

It is just a fixed 64-bit constant, for example: `0x2468ACE13579BDF1`<br>

### Bloom Filter

- Bloom Filter acts as a high-performance "probabilistic gatekeeper" for SSTable Blocks<br>
- It's primary job is to reduce disk I/O by quickly identifying whether a key is present in a block or not.<br>

**Metaindex Block** tells where Bloom Filter block is located<br>
`"filter.bloom" → filter block offset + size`

When searching a key: `e.g, GET(key)`<br>
1. Check Bloom filter first
2. If filter says NO → skip this SSTable completely
3. If filter says MAYBE → then read index + data block

Important:
- Bloom filter is not 100% accurate
- False positives `key maybe in filter but not in block` are possible
- False negatives `key not in filter but in block` are not possible

What it stores?<br>
```
--------------------------------
Bloom filter data (bit array)
--------------------------------
hash functions
--------------------------------
```
Bloom filter stores hashes of keys in a compact bit array. It does not store keys.

**Bloom filter = fast check to skip SSTables that do not contain the key.**


### Block Cache
Block cache stores recently used SSTable blocks in RAM so the DB avoids reading from disk again.<br>

When reading a key:<br>

1. Check block cache first<br>
2. If block is in cache (`HIT`) → read from cache<br>
3. If block is not in cache (`MISS`) → read from disk and insert into cache<br>

**Block cache layout**<br>

```python
Block Cache (RAM)
--------------------------------
block_id → block_data
block_id → block_data
block_id → block_data
--------------------------------
Eviction policy (LRU)
```

**Eviction policy**<br>
`LRU` - Least Recently Used<br>
Oldest data is removed when memory limit is reached (system removes the data that has not been used for the longest time).

**Block cache = fast access to recently used SSTable blocks.**

---

## 7. WAL Truncation

**Purpose**<br>
WAL truncation deletes old WAL files that are no longer needed.<br>
It prevents WAL from growing forever.


**WAL files layout over time**<br>
`WAL-1  WAL-2  WAL-3`

Every write is first saved to WAL (Write Ahead Log) for crash recovery.<br>
Later, data is flushed to SSTables.<br>
Once data is safely inside SSTables, old WAL entries become obsolete.

**When truncation happens?**<br>

After MemTable flush → SSTable created<br>
```python
Data is safe in SSTable
Recovery no longer needs old WAL
```
So WAL can be cleaned<br>

**How WAL is truncated?**<br>

```python
1) Writes go to WAL + MemTable
2) MemTable becomes full
3) MemTable flushed → new SSTable created
4) Manifest updated (SSTable registered)
5) Old WAL files deleted
```

Important:<br>
WAL is deleted only after SSTable is safely recorded in Manifest.<br>
This guarantees crash safety.<br>

**Example:**<br>
BEFORE:
```python
WAL-1  WAL-2  WAL-3 (active)
```

AFTER:
```python
WAL-3 (active)
```

Older WAL segments are removed.<br>

---

## 8. READ PATH

### Snapshot Isolation Model

A snapshot gives a consistent view of the database at a specific time, even while new writes are happening.<br>
Reads must not see half-written or future data ( `> snapshotSequenceNumber`).<br>
Snapshot = **"see data up to seq N"**

**Read Flow:**<br>
```python
Client Get(key)
        │
        ▼
Determine snapshotSeq
        │
        ▼
Search Active Memtable
        │
        ▼
Search Immutable Memtables
        │
        ▼
Search SSTables (Level 0 → L1 → ...)
        │
        ▼
Return newest visible version
```

**Lookup Order**<br>
1) Start read with snapshot_seq
2) Search Active MemTable
3) Search Immutable MemTables (Newer -> Older)
4) Search SSTables (newest to oldest : L0 -> L1 -> ...)
5) Pick newest version ≤ snapshot_seq

**Newer version of data overrides older data versions**


### Bloom Filter interaction:<br>

**GOAL:** Prevent unnecessary SSTable reads and reduce disk I/O<br>
Invariants:<br>

- False positives allowed
- False negatives forbidden

For every SSTable read, the DB does:<br>

```python
Check Bloom filter
      ↓
If NO  → skip this SSTable completely
If MAYBE → check index block → read data block
```

**Read Flow with Bloom Filter:**<br>
```python
Client read request
        ↓
Check MemTable
        ↓
Check Immutable MemTable
        ↓
For each SSTable (newest → oldest):
    ├─ Check Bloom filter
         ├─ NO → skip SSTable
         └─ MAYBE → continue
                │
                ├─ Read Index Block
                ├─ Find Data Block location
                ├─ Read Data Block
                └─ Search key inside block
```

### Block Cache interaction:<br>

**Purpose:**<br>
- Avoid disk reads by using cached blocks in RAM.
- Block cache is used when reading index blocks and data blocks.

**During SSTable search:**<br>
`Bloom filter → Index block → Data block`<br>
Both **Index block** and **Data block** use the block cache.<br>

Block cache stores index and data blocks so repeated reads avoid disk I/O.

**Read Flow with Block Cache:**<br>
```python
1) Check Bloom filter
2) Load Index block
      ├─ Check block cache
      ├─ If hit → use it
      └─ If miss → read from disk → store in cache

3) Find Data block location

4) Load Data block
      ├─ Check block cache
      ├─ If hit → use it
      └─ If miss → read from disk → store in cache
```

In first read : `Disk read -> stored in cache`<br>
In next reads : `if key exists in cache -> Cache hit, no disk read required`<br>

### Tombstone semantics:<br>

If newest visible entry is:<br>
`InternalKey(key, seq, TOMBSTONE)`<br>

Then:<br>
- Key is logically deleted<br>
- Older versions suppressed<br>
- Return NotFound


### Complete Read Flow:
```python
Read(key)
   ↓
Choose snapshot_seq
   ↓
MemTable ?
   ├─ Found → return
   | 
Not found 
   ↓
Immutable MemTable ?
   ├─ Found → return
   |
Not found 
   ↓
For each SSTable (newest → oldest):
   Bloom filter ?
      ├─ NO → skip SSTable
      |
     MAYBE 
      ↓
   Index block
      ├─ Cache hit → use
      └─ Cache miss → read from disk → store in cache
      ↓
   Data block
      ├─ Cache hit → use
      └─ Cache miss → read from disk → store in cache
      ↓
   Key found & visible ?
      ├─ YES → return value / tombstone → not found
      └─ NO → next SSTable
      ↓
  key not found → return NotFound
```

---

## 9. Compaction

Compaction is the background maintenance process that manages the multi-level SSTable structure.<br> 
It prevents the system from having too many files, reclaims space by removing unwanted/obsolete data, and optimizes read performance.

**When does compaction happen?**<br>

**Level 0:**<br>
Triggered when the number of files exceeds a threshold (e.g., 4 files).<br> 
Since L0 files can have overlapping key ranges, having too many makes reads slow.

**Levels 1-6:**<br>
Triggered when the total size of the level exceeds its allowed capacity.<br>
Each level's capacity is typically 10x larger than the one above it ($L_1 = 10MB, L_2 = 100MB$, etc.).<br>

**Note:**
1. Version v0.8 uses threshold-based approach to trigger compaction. 
2. Version v0.8 supports only L0 -> L6 levels.

### Compaction Flow:
```python
Pick SSTables
      ↓
Read using Merge Iterator
      ↓
Merge keys in order
      ↓
Remove old versions & tombstones (when safe)
      ↓
Write new SSTable
      ↓
Update manifest
      ↓
Delete old SSTables

```

### Compaction Process:
Once the input files are selected, the engine performs the following steps:<br>

1) **Sorted merging:**<br>
SSTables are sorted internally, compaction reads many SSTables together using a merge iterator and scans keys in order.<br>
Keys are processed in order `user key ASC, sequence number DESC`.

2) **Garbage collection (space cleanup):**<br>
a) **Shadowing:**<br>
If multiple versions of the same key exist, only the newest version (highest sequence number) is kept, unless older versions are needed by an active snapshot.<br>
b) **Tombstone removal:**<br>
Delete markers are removed only when we are sure no older data exists in lower levels.

3) **Splitting:**<br>
As the merged data is written to new SSTables, the engine monitors the file size.<br>
If a new SSTable size exceeds the limit **(default 20MB)**, it is finalized, and a new one is started. This prevents SSTables from becoming unmanageably large.

4) **Manifest Update & Atomic Transition:**<br>
After compaction creates new SSTables, the database must make the change official.<br>

- New SSTables finished<br>
Compaction writes the new merged SSTable files completely to disk.<br>

- Update Manifest (log record)<br>
A new record is appended to the Manifest saying:<br>
```
ADD NEW SSTABLE
REMOVE OLD SSTABLE
```

- The in-memory VersionSet is updated to reflect the new state.

- The old files are now safe to be physically deleted from the disk.

**This process ensures that data "descends" through the levels, becoming more compact and better organized over time.**

--- 

## 10. User Visibility, Iterator Abstraction, Merge Iterator (Overview)

### User Visibility
Users see the database as one single `sorted key–value store`.<br>
Internally data exists in many places:
- MemTable
- Immutable MemTable
- Multiple SSTables<br>

But the read path hides this complexity.
To the user, reads always return the latest visible version of a key (respecting snapshot).<br>

**User Key**<br>
This is the key the user provides.<br>
User key = **plain key**

Example:<br>

```python
Put apple 10   
Get apple
```

User only knows:<br>
`key = apple, value = 10`

**Internal Key**<br>
Why internal keys exist?<br>
Database must handle:
- Multiple versions of same key
- Deletes (tombstones)
- Snapshots
- Compaction

So DB cannot store just the user key.
It creates an internal key.


Internal key = `[user key ASC, sequence number DESC, operation type]`

Operation type:<br>
`PUT or DELETE`

Sequence number:<br>
Order of writes (newer writes have bigger numbers)

**Example**<br>
User performs:

```python
Put A 10   
Put A 20   
Delete A   
```

Stored internally as:

```python
A | seq=3 | DELETE
A | seq=2 | PUT
A | seq=1 | PUT
```

**During reads:**<br>
- Pick newest version ≤ snapshot<br>
- Ignore older versions<br>
- Handle deletes safely<br>
- Support compaction<br>

User sees only: `latest value`

### Iterator Abstraction

**Purpose**<br>
Iterator abstraction provides one common interface to read sorted data, regardless of where the data lives.<br>

Data may be in:
- MemTable
- SSTables
- Data blocks


Instead of writing different read logic for each structure, the DB uses iterators.

**What an iterator does?**<br>
An iterator lets the DB move through keys in sorted order.

Basic operations:<br>
```python
Seek(key)   → jump to a key or next greater key
Next()      → move forward
Key()       → current key
Value()     → current value
Valid()     → is iterator still usable?
```

Every storage component implements this same interface.

**Iterator abstraction is used in:**<br>
- Point reads
- Range scans
- Merge iterator
- Compaction

### Merge Iterator

**Purpose**<br>
The merge iterator combines multiple sorted iterators into one sorted view of the database.<br>

- MemTable
- Immutable MemTable
- Many SSTables

Each provides its own iterator.<br>
Merge iterator unifies them.

**Merge iterator must:**

- Choose newest version (largest sequence number)
- Ignore older versions
- Ignore tombstones (deleted keys)

So user sees only the latest visible data.

**Merge iterator = merges many sorted iterators and returns newest valid version of a key.**

---