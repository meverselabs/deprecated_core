package transaction

import (
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
)

// Transaction is an interface that defines common transaction functions
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

// Base is the parts of transaction functions that are not changed by derived one
type Base struct {
	ChainCoord_ *common.Coordinate
	Timestamp_  uint64
	Type_       Type
}

// ChainCoord returns the coordinate of the target chain
func (tx *Base) ChainCoord() *common.Coordinate {
	return tx.ChainCoord_.Clone()
}

// Timestamp returns the timestamp
func (tx *Base) Timestamp() uint64 {
	return tx.Timestamp_
}

// SetType updates the type of the transaction
func (tx *Base) SetType(t Type) {
	tx.Type_ = t
}

// Type returns the type of the transaction
func (tx *Base) Type() Type {
	return tx.Type_
}

// WriteTo is a serialization function
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

// ReadFrom is a deserialization function
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
