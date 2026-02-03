package internal

import (
	"encoding/binary"
	"errors"
)

// RecordType represents the kind of record stored.
type RecordType uint8

const (
	RecordTypeValue     RecordType = 0x01
	RecordTypeTombstone RecordType = 0x02
)

// InternalKey combines UserKey and SequenceNumber/Type.
// Layout: [ UserKey | Seq (7 bytes) | Type (1 byte) ]
type InternalKey struct {
	UserKey []byte
	Seq     uint64
	Type    RecordType
}

// EncodeInternalKey packs userKey, seq, and type.
func EncodeInternalKey(userKey []byte, seq uint64, typ RecordType) []byte {
	if seq>>56 != 0 {
		panic("sequence number exceeds 56 bits")
	}

	buf := make([]byte, len(userKey)+8)
	copy(buf, userKey)

	trailer := (seq << 8) | uint64(typ)
	binary.LittleEndian.PutUint64(buf[len(userKey):], trailer)

	return buf
}

// DecodeInternalKey decodes an InternalKey from raw bytes.
func DecodeInternalKey(b []byte) (InternalKey, error) {
	if len(b) < 8 {
		return InternalKey{}, errors.New("internal key too short")
	}

	userKey := make([]byte, len(b)-8)
	copy(userKey, b[:len(b)-8])

	trailer := binary.LittleEndian.Uint64(b[len(b)-8:])

	seq := trailer >> 8
	typ := RecordType(trailer & 0xFF)

	if typ != RecordTypeValue && typ != RecordTypeTombstone {
		return InternalKey{}, errors.New("invalid record type")
	}

	return InternalKey{
		UserKey: userKey,
		Seq:     seq,
		Type:    typ,
	}, nil
}

// ExtractUserKey splits the user key from the trailer.
func ExtractUserKey(b []byte) []byte {
	if len(b) < 8 {
		return nil
	}
	return b[:len(b)-8]
}

// ExtractTrailer parses sequence and type.
func ExtractTrailer(b []byte) (uint64, RecordType, error) {
	if len(b) < 8 {
		return 0, 0, errors.New("invalid internal key")
	}

	trailer := binary.LittleEndian.Uint64(b[len(b)-8:])
	seq := trailer >> 8
	typ := RecordType(trailer & 0xFF)

	return seq, typ, nil
}
