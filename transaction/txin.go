package transaction

import (
	"io"

	"git.fleta.io/fleta/common/util"
)

// TxIn TODO
type TxIn struct {
	Height uint32
	Index  uint16
	N      uint16
}

// NewTxIn TODO
func NewTxIn(id uint64) *TxIn {
	height, index, n := UnmarshalID(id)
	return &TxIn{
		Height: height,
		Index:  index,
		N:      n,
	}
}

// ID TODO
func (in *TxIn) ID() uint64 {
	return MarshalID(in.Height, in.Index, in.N)
}

// WriteTo TODO
func (in *TxIn) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := util.WriteUint64(w, in.ID()); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom TODO
func (in *TxIn) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if id, n, err := util.ReadUint64(r); err != nil {
		return read, err
	} else {
		read += n
		height, index, N := UnmarshalID(id)
		in.Height = height
		in.Index = index
		in.N = N
	}
	return read, nil
}

// UnmarshalID TODO
func UnmarshalID(id uint64) (uint32, uint16, uint16) {
	return uint32(id >> 32), uint16(id >> 16), uint16(id)
}

// MarshalID TODO
func MarshalID(height uint32, index uint16, N uint16) uint64 {
	return uint64(height)<<32 | uint64(index)<<16 | uint64(N)
}
