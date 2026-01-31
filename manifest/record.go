package manifest

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
)

var ErrInvalidRecord = errors.New("invalid manifest record")

// Record is a decoded manifest entry.
type Record struct {
	Type uint8
	Data any
}

// ADD_SSTABLE payload
type AddSSTable struct {
	FileNum     uint64
	Level       uint32
	SmallestSeq uint64
	LargestSeq  uint64
	SmallestKey []byte
	LargestKey  []byte
}

// REMOVE_SSTABLE payload
type RemoveSSTable struct {
	FileNum uint64
}

// SET_WAL_CUTOFF payload
type SetWALCutoff struct {
	Seq uint64
}

func EncodeRecord(rec Record) ([]byte, error) {
	var payload bytes.Buffer

	switch rec.Type {
	case RecordTypeAddSSTable:
		r := rec.Data.(AddSSTable)
		binary.Write(&payload, binary.LittleEndian, r.FileNum)
		binary.Write(&payload, binary.LittleEndian, r.Level)
		binary.Write(&payload, binary.LittleEndian, r.SmallestSeq)
		binary.Write(&payload, binary.LittleEndian, r.LargestSeq)

		binary.Write(&payload, binary.LittleEndian, uint32(len(r.SmallestKey)))
		payload.Write(r.SmallestKey)

		binary.Write(&payload, binary.LittleEndian, uint32(len(r.LargestKey)))
		payload.Write(r.LargestKey)

	case RecordTypeRemoveSSTable:
		r := rec.Data.(RemoveSSTable)
		binary.Write(&payload, binary.LittleEndian, r.FileNum)

	case RecordTypeSetWALCutoff:
		r := rec.Data.(SetWALCutoff)
		binary.Write(&payload, binary.LittleEndian, r.Seq)

	default:
		return nil, ErrInvalidRecord
	}

	header := []byte{
		rec.Type,
		0, // flags
		0, 0,
	}

	length := uint32(len(header) + payload.Len())

	buf := bytes.NewBuffer(nil)
	binary.Write(buf, binary.LittleEndian, uint32(0)) // CRC placeholder
	binary.Write(buf, binary.LittleEndian, length)
	buf.Write(header)
	buf.Write(payload.Bytes())

	raw := buf.Bytes()
	crc := crc32.ChecksumIEEE(raw[4:])
	binary.LittleEndian.PutUint32(raw[0:], crc)

	return raw, nil
}

func DecodeRecord(data []byte) (Record, int, error) {
	if len(data) < 12 {
		return Record{}, 0, ErrInvalidRecord
	}

	wantCRC := binary.LittleEndian.Uint32(data[0:4])
	length := binary.LittleEndian.Uint32(data[4:8])
	total := int(8 + length)

	if len(data) < total {
		return Record{}, 0, ErrInvalidRecord
	}

	if crc32.ChecksumIEEE(data[4:total]) != wantCRC {
		return Record{}, 0, ErrInvalidRecord
	}

	h := data[8:12]
	if h[1] != 0 {
		return Record{}, 0, ErrInvalidRecord
	}

	payload := data[12:total]
	rec := Record{Type: h[0]}

	switch rec.Type {
	case RecordTypeAddSSTable:
		var out AddSSTable
		rd := bytes.NewReader(payload)
		binary.Read(rd, binary.LittleEndian, &out.FileNum)
		binary.Read(rd, binary.LittleEndian, &out.Level)
		binary.Read(rd, binary.LittleEndian, &out.SmallestSeq)
		binary.Read(rd, binary.LittleEndian, &out.LargestSeq)

		var n uint32
		binary.Read(rd, binary.LittleEndian, &n)
		out.SmallestKey = make([]byte, n)
		rd.Read(out.SmallestKey)

		binary.Read(rd, binary.LittleEndian, &n)
		out.LargestKey = make([]byte, n)
		rd.Read(out.LargestKey)

		rec.Data = out

	case RecordTypeRemoveSSTable:
		var out RemoveSSTable
		binary.Read(bytes.NewReader(payload), binary.LittleEndian, &out.FileNum)
		rec.Data = out

	case RecordTypeSetWALCutoff:
		var out SetWALCutoff
		binary.Read(bytes.NewReader(payload), binary.LittleEndian, &out.Seq)
		rec.Data = out

	default:
		return Record{}, 0, ErrInvalidRecord
	}

	return rec, total, nil
}
