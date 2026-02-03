## High-Level Overview

**1. Write Path (Put / Delete)**

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
  v
Write(Batch)
  |
  | acquire write mutex (single writer)
  v
Assign Sequence Numbers
  |
  v
Encode WAL Batch (Batch + Frame Record + CRC)
  |
  v
Append to WAL (active segment)
  |
  v
fsync(WAL)   <-- DURABILITY BOUNDARY
  |
  v
Apply to Memtable
  |
  v
Release write mutex
  |
  v
Return success(Ack) to client
```

--- 

**2. Internal key-value Semantics**

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
