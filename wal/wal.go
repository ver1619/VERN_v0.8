package wal

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const (
	defaultSegmentSize       = 64 * 1024 * 1024 // 64MB
	LogicalTypePut     uint8 = 0x01
	LogicalTypeDelete  uint8 = 0x02
)

// WAL manages write-ahead log segments.
type WAL struct {
	mu           sync.Mutex
	dir          string
	maxSegmentSz int64
	active       *Segment
	activeNum    uint64
	segments     map[uint64]*Segment
}

// OpenWAL opens or creates a WAL.
func OpenWAL(dir string, maxSegmentSize int64) (*WAL, error) {
	if maxSegmentSize <= 0 {
		maxSegmentSize = defaultSegmentSize
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	w := &WAL{
		dir:          dir,
		maxSegmentSz: maxSegmentSize,
		segments:     make(map[uint64]*Segment),
	}

	if err := w.loadExistingSegments(); err != nil {
		return nil, err
	}

	if w.active == nil {
		if err := w.createNewSegment(); err != nil {
			return nil, err
		}
	}

	return w, nil
}

// Append writes a batch to the log.
func (w *WAL) Append(batch Batch) error {
	record, err := EncodeRecord(batch)
	if err != nil {
		return err
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// Rotate if needed.
	if w.active.Size()+int64(len(record)) > w.maxSegmentSz {
		if err := w.rotate(); err != nil {
			return err
		}
	}

	return w.active.Append(record)
}

// Sync flushes the active segment.
func (w *WAL) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.active.Sync()
}

// Close closes all segments.
func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, seg := range w.segments {
		if err := seg.Close(); err != nil {
			return err
		}
	}
	return nil
}

// Segments returns sorted WAL file paths.
func (w *WAL) Segments() []string {
	w.mu.Lock()
	defer w.mu.Unlock()

	var nums []uint64
	for n := range w.segments {
		nums = append(nums, n)
	}
	sort.Slice(nums, func(i, j int) bool { return nums[i] < nums[j] })

	var paths []string
	for _, n := range nums {
		paths = append(paths, w.segmentPath(n))
	}
	return paths
}

/* ---------- Internal Helpers ---------- */

func (w *WAL) loadExistingSegments() error {
	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return err
	}

	var nums []uint64
	for _, e := range entries {
		var n uint64
		if _, err := fmt.Sscanf(e.Name(), "wal_%06d.log", &n); err == nil {
			nums = append(nums, n)
		}
	}

	if len(nums) == 0 {
		return nil
	}

	sort.Slice(nums, func(i, j int) bool { return nums[i] < nums[j] })

	for _, n := range nums {
		seg, err := OpenSegment(w.segmentPath(n))
		if err != nil {
			return err
		}
		w.segments[n] = seg
		w.active = seg
		w.activeNum = n
	}

	return nil
}

func (w *WAL) rotate() error {
	if err := w.active.Close(); err != nil {
		return err
	}
	return w.createNewSegment()
}

func (w *WAL) createNewSegment() error {
	w.activeNum++
	path := w.segmentPath(w.activeNum)

	seg, err := OpenSegment(path)
	if err != nil {
		return err
	}

	w.segments[w.activeNum] = seg
	w.active = seg
	return nil
}

func (w *WAL) segmentPath(n uint64) string {
	return filepath.Join(w.dir, fmt.Sprintf("wal_%06d.log", n))
}

// IsWALFile checks format.
func IsWALFile(name string) bool {
	return strings.HasPrefix(name, "wal_") && strings.HasSuffix(name, ".log")
}

// PathJoin helper.
func PathJoin(dir, name string) string {
	return filepath.Join(dir, name)
}
