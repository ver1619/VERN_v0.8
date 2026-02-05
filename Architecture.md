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

---

**3. WAL**

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

**Atomicity Rules**<br>
A batch is:
- Fully replayed Or fully ignored
- Partial batches are never applied

How This Is Enforced?
- CRC covers entire batch
- Count and SeqStart are validated
- Any mismatch → discard entire record







