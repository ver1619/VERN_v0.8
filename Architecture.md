# High-Level Overview

## 1. Write Path (Put / Delete)

* Write(Batch)
* Single writer thread
* Each write is assigned monotonically increasing seq.number
* Each Wal entry is encoded  `[length | payload | checksum]`, stored in bytes
* Wal Append
* fsync (done before Ack)
* Memtable Insert

```
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
```
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
```
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

## Manifest
The **Manifest** is the **single authoritative metadata log** of the entire storage engine.<br>
>If something is not in the Manifest, it is not part of the database.<br>

It defines:
- Which SSTables exist
- Which levels they belong to
- Sequence number boundaries
- WAL truncation cutoffs<br>

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
- Is fsync’d before becoming visible<br>

There is no in-place update.

**Responsibilities:**<br>
- Persist VersionSet evolution
- Act as recovery authority
- Define WAL truncation safety
- Enable deterministic replay
- Serialize lifecycle transitions (flush, compaction)


**Interaction**<br>

```
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
