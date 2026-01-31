package manifest

import (
	"os"
)

type Manifest struct {
	file *os.File
}

// OpenManifest opens or creates the manifest file.
func OpenManifest(path string) (*Manifest, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &Manifest{file: f}, nil
}

// Append writes a record and fsyncs.
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
