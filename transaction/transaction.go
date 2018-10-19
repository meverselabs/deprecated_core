package transaction

import (
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
)

// Transaction TODO
type Transaction interface {
	io.WriterTo
	io.ReaderFrom
	ChainCoord() *common.Coordinate
	Timestamp() uint64
	SetType(t Type)
	Type() Type
	Hash() hash.Hash256
	IsUTXO() bool
}

// Base TODO
type Base struct {
	ChainCoord_ *common.Coordinate
	Timestamp_  uint64
	Type_       Type
}

// ChainCoord TODO
func (tx *Base) ChainCoord() *common.Coordinate {
	return tx.ChainCoord_.Clone()
}

// Timestamp TODO
func (tx *Base) Timestamp() uint64 {
	return tx.Timestamp_
}

// SetType TODO
func (tx *Base) SetType(t Type) {
	tx.Type_ = t
}

// Type TODO
func (tx *Base) Type() Type {
	return tx.Type_
}

// WriteTo TODO
func (tx *Base) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := tx.ChainCoord_.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint64(w, tx.Timestamp_); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint8(w, uint8(tx.Type_)); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom TODO
func (tx *Base) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := tx.ChainCoord_.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if v, n, err := util.ReadUint64(r); err != nil {
		return read, err
	} else {
		read += n
		tx.Timestamp_ = v
	}
	if v, n, err := util.ReadUint8(r); err != nil {
		return read, err
	} else {
		read += n
		tx.Type_ = Type(v)
	}
	return read, nil
}
