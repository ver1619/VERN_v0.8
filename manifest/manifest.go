package manifest

import (
	"os"
)

// Manifest tracks DB state changes.
type Manifest struct {
	file *os.File
}

// OpenManifest opens or creates a manifest file.
func OpenManifest(path string) (*Manifest, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &Manifest{file: f}, nil
}

// Append writes a record.
func (m *Manifest) Append(rec Record) error {
	raw, err := EncodeRecord(rec)
	if err != nil {
		return err
	}

	if _, err := m.file.Write(raw); err != nil {
		return err
	}

	return m.file.Sync()
}

// Close closes the manifest.
func (m *Manifest) Close() error {
	return m.file.Close()
}

// Rewrite creates a new manifest with current state.
func Rewrite(path string, records []Record) error {
	tmpPath := path + ".tmp"
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, rec := range records {
		raw, err := EncodeRecord(rec)
		if err != nil {
			return err
		}
		if _, err := f.Write(raw); err != nil {
			return err
		}
	}

	if err := f.Sync(); err != nil {
		return err
	}
	f.Close()

	return os.Rename(tmpPath, path)
}
