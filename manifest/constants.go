package manifest

// Record types
const (
	RecordTypeAddSSTable    uint8 = 0x01
	RecordTypeRemoveSSTable uint8 = 0x02
	RecordTypeSetWALCutoff  uint8 = 0x03
)
