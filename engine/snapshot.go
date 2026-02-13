package engine

// Snapshot represents a stable read view.
//
// A snapshot guarantees that reads will only observe
// versions with sequence numbers <= ReadSeq.
type Snapshot struct {
	ReadSeq uint64

	db   *DB
	prev *Snapshot
	next *Snapshot
}

// ReadOptions controls read behavior.
//
// If Snapshot is nil, reads observe the latest state.
type ReadOptions struct {
	Snapshot *Snapshot
}
