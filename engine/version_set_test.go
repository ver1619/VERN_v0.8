package engine

import (
	"path/filepath"
	"testing"

	"vern_kv0.8/manifest"
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

	if len(vs.GetAllTables()) != 0 {
		t.Fatalf("expected no active tables")
	}
	if !vs.Obsolete[1] {
		t.Fatalf("expected table 1 to be obsolete")
	}
	if vs.WALCutoffSeq != 10 {
		t.Fatalf("unexpected WAL cutoff seq")
	}
}

func TestVersionSet_AddTable(t *testing.T) {
	vs := NewVersionSet()

	// Add tables to L1 (should be sorted by SmallestKey)
	t1 := SSTableMeta{
		FileNum:     1,
		Level:       1,
		SmallestKey: []byte("c"),
		LargestKey:  []byte("d"),
	}
	t2 := SSTableMeta{
		FileNum:     2,
		Level:       1,
		SmallestKey: []byte("a"),
		LargestKey:  []byte("b"),
	}
	t3 := SSTableMeta{
		FileNum:     3,
		Level:       1,
		SmallestKey: []byte("e"),
		LargestKey:  []byte("f"),
	}

	if err := vs.AddTable(t1); err != nil {
		t.Fatal(err)
	}
	if err := vs.AddTable(t2); err != nil {
		t.Fatal(err)
	}
	if err := vs.AddTable(t3); err != nil {
		t.Fatal(err)
	}

	// Verify order: a, c, e
	files := vs.Levels[1]
	if len(files) != 3 {
		t.Fatalf("expected 3 files in L1, got %d", len(files))
	}
	if string(files[0].SmallestKey) != "a" {
		t.Errorf("expected first key 'a', got %s", files[0].SmallestKey)
	}
	if string(files[1].SmallestKey) != "c" {
		t.Errorf("expected second key 'c', got %s", files[1].SmallestKey)
	}
	if string(files[2].SmallestKey) != "e" {
		t.Errorf("expected third key 'e', got %s", files[2].SmallestKey)
	}
}

func TestVersionSet_RemoveTable(t *testing.T) {
	vs := NewVersionSet()
	t1 := SSTableMeta{FileNum: 1, Level: 0}
	vs.AddTable(t1)

	if len(vs.Levels[0]) != 1 {
		t.Fatal("setup failed")
	}

	vs.RemoveTable(1)

	if len(vs.Levels[0]) != 0 {
		t.Error("table not removed from level")
	}
	if !vs.Obsolete[1] {
		t.Error("table not marked obsolete")
	}
}

func TestVersionSet_PickCompaction(t *testing.T) {
	vs := NewVersionSet()

	// Case 1: L0 triggers compaction
	// L0Trigger = 4. Add 5 tables.
	for i := 0; i < 5; i++ {
		vs.AddTable(SSTableMeta{FileNum: uint64(i), Level: 0})
	}

	level, score := vs.PickCompaction(4, 1024)
	if !score {
		t.Error("expected compaction trigger for L0")
	}
	if level != 0 {
		t.Errorf("expected level 0, got %d", level)
	}

	// Case 2: L1 triggers compaction
	// Clear L0
	vs.Levels[0] = nil

	// L1 max bytes = 1000. Add 1 file of size 2MB (simulated in PickCompaction logic as ~2MB per file)
	// Code: currentSize := float64(len(v.Levels[l])) * 2 * 1024 * 1024
	// Target: 1000
	// Score should be massive.
	vs.AddTable(SSTableMeta{FileNum: 10, Level: 1})

	l, ok := vs.PickCompaction(4, 1000)
	if !ok {
		t.Error("expected compaction trigger for L1")
	}
	if l != 1 {
		t.Errorf("expected level 1, got %d", l)
	}
}

func TestVersionSet_GetOverlappingInputs(t *testing.T) {
	vs := NewVersionSet()

	// L1: [a, b], [d, e], [g, h]
	vs.AddTable(SSTableMeta{FileNum: 1, Level: 1, SmallestKey: []byte("a"), LargestKey: []byte("b")})
	vs.AddTable(SSTableMeta{FileNum: 2, Level: 1, SmallestKey: []byte("d"), LargestKey: []byte("e")})
	vs.AddTable(SSTableMeta{FileNum: 3, Level: 1, SmallestKey: []byte("g"), LargestKey: []byte("h")})

	// Range [b, d] overlaps with tables 1 and 2.
	inputs := vs.GetOverlappingInputs(1, []byte("b"), []byte("d"))

	if len(inputs) != 2 {
		t.Fatalf("expected 2 overlapping inputs, got %d", len(inputs))
	}
	if inputs[0].FileNum != 1 || inputs[1].FileNum != 2 {
		for i, inp := range inputs {
			t.Logf("Input %d: FileNum %d", i, inp.FileNum)
		}
		t.Errorf("unexpected overlap result")
	}

	// Range [e, f] overlaps with table 2.
	inputs = vs.GetOverlappingInputs(1, []byte("e"), []byte("f"))
	if len(inputs) != 1 {
		t.Fatalf("expected 1 overlapping input, got %d", len(inputs))
	}
}
