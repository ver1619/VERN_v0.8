package engine

import (
	"os"
	"path/filepath"

	"vern_kv0.5/manifest"
)

// bootstrapIfNeeded initializes a fresh database if needed.
func bootstrapIfNeeded(dir string) error {
	// Ensure base dir exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	manifestPath := filepath.Join(dir, "MANIFEST")
	walDir := filepath.Join(dir, "wal")

	// Ensure WAL directory exists
	if err := os.MkdirAll(walDir, 0755); err != nil {
		return err
	}

	// If MANIFEST exists, nothing to do
	if _, err := os.Stat(manifestPath); err == nil {
		return nil
	}

	// Create initial MANIFEST
	m, err := manifest.OpenManifest(manifestPath)
	if err != nil {
		return err
	}

	// Initial WAL cutoff = 0
	err = m.Append(manifest.Record{
		Type: manifest.RecordTypeSetWALCutoff,
		Data: manifest.SetWALCutoff{Seq: 0},
	})
	if err != nil {
		return err
	}

	return m.Close()
}
