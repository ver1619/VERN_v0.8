package wal

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
)

const (
	recordTypeWriteBatch uint8 = 0x01

	logicalTypePut    uint8 = 0x01
	logicalTypeDelete uint8 = 0x02
)

var (
	errInvalidRecord = errors.New("invalid wal record")
)

// LogicalRecord represents a single PUT or DELETE inside a batch.
type LogicalRecord struct {
	Key   []byte
	Value []byte
	Type  uint8 // logicalTypePut or logicalTypeDelete
}

// Batch represents an atomic WAL batch.
type Batch struct {
	SeqStart uint64
	Records  []LogicalRecord
}

// EncodeRecord encodes a batch into a WAL record.
func EncodeRecord(batch Batch) ([]byte, error) {
	if len(batch.Records) == 0 {
		return nil, errors.New("empty batch")
	}
	if batch.SeqStart>>56 != 0 {
		return nil, errors.New("sequence number overflow")
	}

	var payload bytes.Buffer

	for _, r := range batch.Records {
		if len(r.Key) == 0 {
			return nil, errors.New("empty key")
		}

		binary.Write(&payload, binary.LittleEndian, uint32(len(r.Key)))

		if r.Type == logicalTypeDelete {
			binary.Write(&payload, binary.LittleEndian, uint32(0))
		} else {
			binary.Write(&payload, binary.LittleEndian, uint32(len(r.Value)))
		}

		payload.WriteByte(r.Type)
		payload.Write(r.Key)

		if r.Type == logicalTypePut {
			payload.Write(r.Value)
		}
	}

	header := make([]byte, 16)
	header[0] = recordTypeWriteBatch
	header[1] = 0 // flags
	binary.LittleEndian.PutUint64(header[4:], batch.SeqStart)
	binary.LittleEndian.PutUint32(header[12:], uint32(len(batch.Records)))

	length := uint32(len(header) + payload.Len())

	var buf bytes.Buffer
	binary.Write(&buf, binary.LittleEndian, uint32(0)) // placeholder CRC
	binary.Write(&buf, binary.LittleEndian, length)    // length
	buf.Write(header)
	buf.Write(payload.Bytes())

	raw := buf.Bytes()

	crc := crc32.ChecksumIEEE(raw[4:])
	binary.LittleEndian.PutUint32(raw[0:], crc)

	return raw, nil
}

// DecodeRecord decodes a WAL record and validates CRC.
func DecodeRecord(data []byte) (Batch, int, error) {
	if len(data) < 4+4+16 {
		return Batch{}, 0, errInvalidRecord
	}

	expectedCRC := binary.LittleEndian.Uint32(data[0:4])
	length := binary.LittleEndian.Uint32(data[4:8])

	total := int(8 + length)
	if len(data) < total {
		return Batch{}, 0, errInvalidRecord
	}

	actualCRC := crc32.ChecksumIEEE(data[4:total])
	if actualCRC != expectedCRC {
		return Batch{}, 0, errInvalidRecord
	}

	header := data[8 : 8+16]
	if header[0] != recordTypeWriteBatch || header[1] != 0 {
		return Batch{}, 0, errInvalidRecord
	}

	seqStart := binary.LittleEndian.Uint64(header[4:])
	count := binary.LittleEndian.Uint32(header[12:])

	payload := data[24:total]
	records := make([]LogicalRecord, 0, count)

	offset := 0
	for i := uint32(0); i < count; i++ {
		if offset+9 > len(payload) {
			return Batch{}, 0, errInvalidRecord
		}

		keyLen := binary.LittleEndian.Uint32(payload[offset:])
		valLen := binary.LittleEndian.Uint32(payload[offset+4:])
		typ := payload[offset+8]
		offset += 9

		if offset+int(keyLen) > len(payload) {
			return Batch{}, 0, errInvalidRecord
		}

		key := make([]byte, keyLen)
		copy(key, payload[offset:offset+int(keyLen)])
		offset += int(keyLen)

		var value []byte
		if typ == logicalTypePut {
			if offset+int(valLen) > len(payload) {
				return Batch{}, 0, errInvalidRecord
			}
			value = make([]byte, valLen)
			copy(value, payload[offset:offset+int(valLen)])
			offset += int(valLen)
		}

		records = append(records, LogicalRecord{
			Key:   key,
			Value: value,
			Type:  typ,
		})
	}

	return Batch{
		SeqStart: seqStart,
		Records:  records,
	}, total, nil
}
