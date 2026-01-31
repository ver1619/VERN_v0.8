package wal

import (
	"errors"
	"os"
	"sync"
)

var (
	errSegmentClosed = errors.New("wal segment closed")
)

// Segment represents a single WAL segment file.
type Segment struct {
	mu     sync.Mutex
	file   *os.File
	path   string
	size   int64
	closed bool
}

// OpenSegment opens (or creates) a WAL segment at path.
// If the file already exists, it is opened in append mode.
func OpenSegment(path string) (*Segment, error) {
	file, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_RDWR|os.O_APPEND,
		0644,
	)
	if err != nil {
		return nil, err
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, err
	}

	return &Segment{
		file: file,
		path: path,
		size: info.Size(),
	}, nil
}

// Append appends a full WAL record to the segment.
// The record must already be fully encoded.
func (s *Segment) Append(record []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return errSegmentClosed
	}

	n, err := s.file.Write(record)
	if err != nil {
		return err
	}

	s.size += int64(n)
	return nil
}

// Sync fsyncs the segment file.
func (s *Segment) Sync() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return errSegmentClosed
	}

	return s.file.Sync()
}

// Size returns the current size of the segment.
func (s *Segment) Size() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.size
}

// Close fsyncs and closes the segment.
// After Close, no further appends are allowed.
func (s *Segment) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	if err := s.file.Sync(); err != nil {
		return err
	}

	err := s.file.Close()
	s.closed = true
	return err
}
