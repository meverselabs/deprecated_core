package account

import (
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/util"
)

// Type is using when serealization and deserialization account
type Type uint8

// Account is a interface that defines common account functions
type Account interface {
	Address() common.Address
	SetType(t Type)
	Type() Type
	Clone() Account
	io.WriterTo
	io.ReaderFrom
}

// Base is the parts of account functions that are not changed by derived one
type Base struct {
	Address_ common.Address
	Type_    Type
}

// Address returns the account address
func (acc *Base) Address() common.Address {
	return acc.Address_
}

// SetType set the account type
func (acc *Base) SetType(t Type) {
	acc.Type_ = t
}

// Type returns the account type
func (acc *Base) Type() Type {
	return acc.Type_
}

// WriteTo is a serialization function
func (acc *Base) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := acc.Address_.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint8(w, uint8(acc.Type_)); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom is a deserialization function
func (acc *Base) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := acc.Address_.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if v, n, err := util.ReadUint8(r); err != nil {
		return read, err
	} else {
		read += n
		acc.Type_ = Type(v)
	}
	return read, nil
}
