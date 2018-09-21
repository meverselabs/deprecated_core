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
	Coordinate() *common.Coordinate
	Timestamp() uint64
	Hash() (hash.Hash256, error)
}

// Base TODO
type Base struct {
	Coordinate_ *common.Coordinate
	Timestamp_  uint64
}

// Coordinate TODO
func (tx *Base) Coordinate() *common.Coordinate {
	return tx.Coordinate_.Clone()
}

// Timestamp TODO
func (tx *Base) Timestamp() uint64 {
	return tx.Timestamp_
}

// WriteTo TODO
func (tx *Base) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := tx.Coordinate_.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint64(w, tx.Timestamp_); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom TODO
func (tx *Base) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := tx.Coordinate_.ReadFrom(r); err != nil {
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
	return read, nil
}
