package manifest

// Compile-time checks to prevent accidental constant changes.

const (
	_ = uint8(1) / (uint8(1) - (RecordTypeAddSSTable ^ 0x01))
	_ = uint8(1) / (uint8(1) - (RecordTypeRemoveSSTable ^ 0x02))
	_ = uint8(1) / (uint8(1) - (RecordTypeSetWALCutoff ^ 0x03))
)
