package manifest

// Compile-time assertions to lock on-disk constants.
// These MUST fail to compile if values change.

const (
	_ = uint8(1) / (uint8(1) - (RecordTypeAddSSTable ^ 0x01))
	_ = uint8(1) / (uint8(1) - (RecordTypeRemoveSSTable ^ 0x02))
	_ = uint8(1) / (uint8(1) - (RecordTypeSetWALCutoff ^ 0x03))
)
