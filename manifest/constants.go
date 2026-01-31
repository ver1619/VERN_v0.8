package manifest

// Manifest record type constants.
// THESE ARE PART OF THE ON-DISK FORMAT.
// Changing numeric values BREAKS compatibility.
const (
	RecordTypeAddSSTable    uint8 = 0x01
	RecordTypeRemoveSSTable uint8 = 0x02
	RecordTypeSetWALCutoff  uint8 = 0x03
)
