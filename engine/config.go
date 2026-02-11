package engine

import "vern_kv0.8/sstable"

// Config holds the configuration for the database.
type Config struct {
	// WalDir is the directory for WAL files.
	WalDir string

	// MemtableSizeLimit is the size threshold for flushing memtable (bytes).
	MemtableSizeLimit int

	// CompressionType specifies the block compression algorithm.
	CompressionType int

	// BlockSize is the size of uncompressed SSTable blocks (bytes).
	BlockSize int

	// L0CompactionTrigger is the number of L0 files to trigger compaction.
	L0CompactionTrigger int

	// L1MaxBytes is the max total size for L1 (bytes).
	L1MaxBytes int64

	// SyncWrites controls whether each write is fsynced to WAL.
	// When true (default), every Put/Delete/Write is durable after return.
	// When false, writes are buffered and may be lost on crash.
	SyncWrites bool
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		WalDir:              "wal",
		MemtableSizeLimit:   4 * 1024 * 1024, // 4MB
		CompressionType:     sstable.NoCompression,
		BlockSize:           4 * 1024, // 4KB
		L0CompactionTrigger: 4,
		L1MaxBytes:          64 * 1024 * 1024, // 64MB
		SyncWrites:          true,
	}
}
