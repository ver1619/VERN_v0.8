package engine

import (
	"path/filepath"
	"testing"

	"vern_kv0.5/manifest"
)

func TestVersionSetReplay(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "MANIFEST")

	m, err := manifest.OpenManifest(path)
	if err != nil {
		t.Fatal(err)
	}

	// Add SSTable
	m.Append(manifest.Record{
		Type: manifest.RecordTypeAddSSTable,
		Data: manifest.AddSSTable{
			FileNum:     1,
			Level:       0,
			SmallestSeq: 1,
			LargestSeq:  10,
			SmallestKey: []byte("a"),
			LargestKey:  []byte("z"),
		},
	})

	// Set WAL cutoff
	m.Append(manifest.Record{
		Type: manifest.RecordTypeSetWALCutoff,
		Data: manifest.SetWALCutoff{Seq: 10},
	})

	// Remove SSTable
	m.Append(manifest.Record{
		Type: manifest.RecordTypeRemoveSSTable,
		Data: manifest.RemoveSSTable{FileNum: 1},
	})

	m.Close()

	vs, err := ReplayManifest(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(vs.Tables) != 0 {
		t.Fatalf("expected no active tables")
	}
	if !vs.Obsolete[1] {
		t.Fatalf("expected table 1 to be obsolete")
	}
	if vs.WALCutoffSeq != 10 {
		t.Fatalf("unexpected WAL cutoff seq")
	}
}
