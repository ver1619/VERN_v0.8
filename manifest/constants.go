package manifest

// Record types (On-Disk Format).
// WARNING: Do not change values (compatibility break).
const (
	RecordTypeAddSSTable    uint8 = 0x01
	RecordTypeRemoveSSTable uint8 = 0x02
	RecordTypeSetWALCutoff  uint8 = 0x03
)
