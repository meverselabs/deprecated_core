package advanced

import (
	"bytes"
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/core/transaction"
)

// RevokeFormulation TODO
type RevokeFormulation struct {
	transaction.Base
	From               common.Address
	FormulationAddress common.Address
}

// NewRevokeFormulation TODO
func NewRevokeFormulation(version uint16, timestamp uint64) *RevokeFormulation {
	return &RevokeFormulation{
		Base: transaction.Base{
			Version_:   version,
			Timestamp_: timestamp,
		},
	}
}

// Hash TODO
func (tx *RevokeFormulation) Hash() (hash.Hash256, error) {
	var buffer bytes.Buffer
	if _, err := tx.WriteTo(&buffer); err != nil {
		return hash.Hash256{}, err
	}
	return hash.DoubleHash(buffer.Bytes()), nil
}

// WriteTo TODO
func (tx *RevokeFormulation) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := tx.Base.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := tx.From.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := tx.FormulationAddress.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom TODO
func (tx *RevokeFormulation) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := tx.Base.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := tx.From.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := tx.FormulationAddress.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	return read, nil
}
