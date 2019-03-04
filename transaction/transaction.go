package transaction

import (
	"encoding/json"
	"io"

	"github.com/fletaio/common/hash"
	"github.com/fletaio/common/util"
)

// Transaction is an interface that defines common transaction functions
type Transaction interface {
	io.WriterTo
	io.ReaderFrom
	json.Marshaler
	Type() Type
	Timestamp() uint64
	Hash() hash.Hash256
	IsUTXO() bool
}

// Base is the parts of transaction functions that are not changed by derived one
type Base struct {
	Type_      Type
	Timestamp_ uint64
}

// Type returns the type of the transaction
func (tx *Base) Type() Type {
	return tx.Type_
}

// Timestamp returns the timestamp
func (tx *Base) Timestamp() uint64 {
	return tx.Timestamp_
}

// WriteTo is a serialization function
func (tx *Base) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := util.WriteUint8(w, uint8(tx.Type_)); err != nil {
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

// ReadFrom is a deserialization function
func (tx *Base) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if v, n, err := util.ReadUint8(r); err != nil {
		return read, err
	} else {
		read += n
		tx.Type_ = Type(v)
	}
	if v, n, err := util.ReadUint64(r); err != nil {
		return read, err
	} else {
		read += n
		tx.Timestamp_ = v
	}
	return read, nil
}
