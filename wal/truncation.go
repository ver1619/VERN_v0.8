package wal

import (
	"os"
	"sort"
)

// Truncate safely removes old WAL segments.
// Persists change by syncing directory.
func Truncate(walDir string, cutoffSeq uint64) error {
	entries, err := os.ReadDir(walDir)
	if err != nil {
		return err
	}

	var segments []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if IsWALFile(e.Name()) {
			segments = append(segments, PathJoin(walDir, e.Name()))
		}
	}

	sort.Strings(segments)

	var deletable []string

	for i, path := range segments {
		// Always preserve the last (active) segment to avoid deleting a file the WAL handle is still writing
		if i == len(segments)-1 {
			break
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		var maxSeq uint64
		offset := 0

		for offset < len(data) {
			batch, n, err := DecodeRecord(data[offset:])
			if err != nil {
				// Stop on corruption
				return nil
			}

			batchMax := batch.SeqStart + uint64(len(batch.Records)) - 1
			if batchMax > maxSeq {
				maxSeq = batchMax
			}

			offset += n
		}

		if maxSeq <= cutoffSeq {
			deletable = append(deletable, path)
		} else {
			break // prefix rule
		}
	}

	// Delete deletable segments
	for _, path := range deletable {
		_ = os.Remove(path)
	}

	// Ensure directory durability
	dir, err := os.Open(walDir)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
