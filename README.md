# ***VernKV v0.8***

[![Go Reference](https://pkg.go.dev/badge/github.com/ver1619/VERN_v0.8.svg?style=flat-square)](https://pkg.go.dev/github.com/ver1619/VERN_v0.8)
![Go Version](https://img.shields.io/badge/Go-1.22%2B-1f2937?logo=go&logoColor=00ADD8&style=flat-square)
![License](https://img.shields.io/github/license/ver1619/VERN_v0.8?label=license&color=yellow)
![Storage Engine](https://img.shields.io/badge/type-LSM--tree--KV--store-purple)
![Docs](https://img.shields.io/badge/docs-available-blue)
[![Go Report Card](https://goreportcard.com/badge/github.com/ver1619/VERN_v0.8)](https://goreportcard.com/report/github.com/ver1619/VERN_v0.8)


<p align="center">
  <img 
    src="https://github.com/user-attachments/assets/43ce00ec-3c74-4f7e-a5ce-0f5b4019f1f7"
    alt="VernKV logo"
    width="300"
  >
</p>

> **Persistent Log-Structured Merge (LSM) key-value store in GO.**

***VERN KV*** is a robust, embedded storage engine designed for reliability, heavy write throughput, and crash consistency. It implements a classic **LSM-tree architecture**

## Key Features
- **Store Data Safely**: Uses a Write-Ahead Log (WAL) with CRC checksums to ensure data survives crashes.
- **High Performance**: 
  - **Fast Writes**: In-memory `Memtable` (Skiplist) buffers writes before flushing to disk.
  - **Fast Reads**: Uses `Bloom Filters` and `Block Cache` to minimize expensive disk I/O.
- **Snapshot Isolation**: Readers see a consistent snapshot of the database without blocking writers (MVCC).
- **LSM Architecture**: 
  - Writes are append-only (efficient for SSDs).
  - Background **Compaction** (Leveled: L0-L6) merges files and reclaims space.
- **Interactive CLI**: Built-in command-line tool to explore and modify the database directly.
- **Strict Durability**: Operations are `fsync`'d to disk by default (configurable).

---

## 🏗 Architecture Overview

***VERN KV*** follows a tiered storage architecture:

1.  **WAL (Write-Ahead Log)**: Every write (Put/Delete) is first appended here for durability.
2.  **Memtable**: Data is then inserted into an in-memory Skiplist (sorted).
3.  **SSTables (Sorted String Tables)**: When the Memtable is full, it's flushed to disk as an immutable SSTable.
4.  **Compaction**: In the background, older SSTables are merged into higher levels (L0 → L1 → L6) to remove obsolete data and manage disk space.
5.  **Manifest**: Record Book of the database. Tracks the state of the database (e.g, which SSTables belong to which level).

📚 **Want a deep dive?** **Check out [Architecture.md](Architecture.md) and [Invariants.md](Invariants.md).**

---

## Project Structure

👉 **[project-Tree.md](project-Tree.md)**


## 🚀 Quick Start

### Requirements

`Go 1.22+`

### 1. Installation

Clone the repository and start using VernKV:

```go
git clone https://github.com/ver1619/VERN_v0.8.git
cd VERN_v0.8
go build -o bin/vern-cli ./cmd/vern-cli
```

### 2. Using the CLI

VERN KV provides a friendly interactive shell.

```bash
# Start the CLI, by running the following command
./bin/vern-cli
```

👉 **Go through [CLI.md](CLI.md) for the complete CLI guide.**

---

## 💻 Library Usage (Go)

You can embed VERN KV directly into your Go applications.

### Import
```go
import (
    "fmt"
    "vern_kv0.8/engine"
)
```

### Example Usage
**Continuous Ingestion Loop**
```go
for i := 0; i < 1000000; i++ {
	key := fmt.Sprintf("event:%d", i)
	value := fmt.Sprintf("payload-%d", i)

	if err := db.Put([]byte(key), []byte(value)); err != nil {
		log.Fatal(err)
	}
}
```
Represents:
- Log ingestion
- Analytics events
- Streaming workloads


## 💡 Configuration Tips

You can tune the engine for your workload:

```go
opts := &engine.Config{
    MemtableSizeLimit: 64 * 1024 * 1024, // 64MB Memtable
    WalDir:            "wal_logs",      // Custom WAL directory
    SyncWrites:        false,           // Faster writes (risk of data loss on power failure)
}
db, _ := engine.Open("./data", opts)
```

---

## Full Implementation Example:
This code demonstrates a High-Throughput Streaming Workload. It proves that VERN KV can handle 1 Million writes sequentially without exhausting memory, while maintaining data integrity through its background flush and compaction cycles.

```go
package main

import (
    "fmt"
    "log"
    "vern_kv0.8/engine"
)

func main() {
    // 1. Define your custom performance tuning
    opts := &engine.Config{
        MemtableSizeLimit: 64 * 1024 * 1024, // 64MB for high throughput
        WalDir:            "custom_wal",     // Specific folder for logs
        SyncWrites:        false,            // GO FAST mode (async writes)
    }

    // 2. Open the DB with these options
    db, err := engine.Open("./my_database", opts)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    // 3. Start high-speed ingestion...
    fmt.Println("Starting ingestion of 1M records...")
    for i := 0; i < 1000000; i++ {
	    key := fmt.Sprintf("event:%d", i)
	    value := fmt.Sprintf("payload-%d", i)

	    if err := db.Put([]byte(key), []byte(value)); err != nil {
		    log.Fatal(err)
	    }
        if i%100000 == 0 {
            fmt.Printf("Ingested %d records...\n", i)
        }
    }
    fmt.Println("Ingestion complete!")
}
```

**Output:**
```
Starting ingestion of 1M records...
Ingested 0 records...
Ingested 100000 records...
Ingested 200000 records...
Ingested 300000 records...
Ingested 400000 records...
Ingested 500000 records...
Ingested 600000 records...
Ingested 700000 records...
Ingested 800000 records...
Ingested 900000 records...
Ingestion complete!
```

---

## 🧪 Testing & Reliability

***VERN KV*** is rigorously validated through invariant-driven testing, WAL replay verification, compaction correctness checks, and crash-recovery simulations to ensure consistency, durability, and data integrity across all core subsystems.

To run the tests yourself:

```bash
go test ./...
```

👉 **Go through [tests.md](tests.md) for the full test report.**

## 📜 License

This project is open-source. See the **[LICENSE](LICENSE)** file for details.