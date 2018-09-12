package transaction

import (
	"io"

	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
)

// Transaction TODO
type Transaction interface {
	io.WriterTo
	io.ReaderFrom
	Version() uint16
	Timestamp() uint64
	Hash() (hash.Hash256, error)
}

// Base TODO
type Base struct {
	Version_   uint16
	Timestamp_ uint64
}

// NewBase TODO
func NewBase(version uint16, timestamp uint64) *Base {
	return &Base{
		Version_:   version,
		Timestamp_: timestamp,
	}
}

// Version TODO
func (tx *Base) Version() uint16 {
	return tx.Version_
}

// Timestamp TODO
func (tx *Base) Timestamp() uint64 {
	return tx.Timestamp_
}

// WriteTo TODO
func (tx *Base) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := util.WriteUint16(w, tx.Version_); err != nil {
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
	if v, n, err := util.ReadUint16(r); err != nil {
		return read, err
	} else {
		read += n
		tx.Version_ = v
	}
	if v, n, err := util.ReadUint64(r); err != nil {
		return read, err
	} else {
		read += n
		tx.Timestamp_ = v
	}
	return read, nil
}
