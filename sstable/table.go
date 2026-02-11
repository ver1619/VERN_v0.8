package sstable

import (
	"encoding/binary"
	"errors"
	"os"
)

var (
	ErrCorruptSSTable = errors.New("corrupt sstable")
	ErrBlockCorrupt   = errors.New("corrupt block")
)

const (
	MagicNumber uint64 = 0x5645524E_00000008
	FooterSize         = 16 + 16 + 8
)

// BlockHandle represents a block's position and size.
type BlockHandle struct {
	Offset uint64
	Length uint64
}

func (h BlockHandle) EncodeTo(dst []byte) {
	binary.LittleEndian.PutUint64(dst[0:], h.Offset)
	binary.LittleEndian.PutUint64(dst[8:], h.Length)
}

func Open(path string) (*os.File, error) {
	return os.Open(path)
}

func DecodeBlockHandle(src []byte) BlockHandle {
	return BlockHandle{
		Offset: binary.LittleEndian.Uint64(src[0:]),
		Length: binary.LittleEndian.Uint64(src[8:]),
	}
}

// Footer is fixed size.
type Footer struct {
	MetaindexHandle BlockHandle
	IndexHandle     BlockHandle
}

// EncodeTo serializes the footer.
func (f Footer) EncodeTo(dst []byte) {
	f.MetaindexHandle.EncodeTo(dst[0:])
	f.IndexHandle.EncodeTo(dst[16:])
	binary.LittleEndian.PutUint64(dst[32:], MagicNumber)
}

// DecodeFooter deserializes the footer.
func DecodeFooter(src []byte) (Footer, error) {
	if len(src) < FooterSize {
		return Footer{}, ErrCorruptSSTable
	}

	magic := binary.LittleEndian.Uint64(src[32:])
	if magic != MagicNumber {
		return Footer{}, ErrCorruptSSTable
	}

	return Footer{
		MetaindexHandle: DecodeBlockHandle(src[0:]),
		IndexHandle:     DecodeBlockHandle(src[16:]),
	}, nil
}
