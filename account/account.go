package account

import (
	"encoding/json"
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/amount"
)

// Type is using when serealization and deserialization account
type Type uint8

// Account is a interface that defines common account functions
type Account interface {
	io.WriterTo
	io.ReaderFrom
	json.Marshaler
	Type() Type
	Address() common.Address
	Name() string
	Balance() *amount.Amount
	AddBalance(a *amount.Amount)
	SubBalance(a *amount.Amount) error
	Clone() Account
}

// Base is the parts of account functions that are not changed by derived one
type Base struct {
	Type_    Type
	Address_ common.Address
	Name_    string
	Balance_ *amount.Amount
}

// Type returns the account type
func (acc *Base) Type() Type {
	return acc.Type_
}

// Address returns the account address
func (acc *Base) Address() common.Address {
	return acc.Address_
}

// Name returns the account name
func (acc *Base) Name() string {
	return acc.Name_
}

// Balance returns the balance of the account
func (acc *Base) Balance() *amount.Amount {
	return acc.Balance_
}

// AddBalance adds the balance to the account
func (acc *Base) AddBalance(a *amount.Amount) {
	acc.Balance_ = acc.Balance_.Add(a)
}

// SubBalance subs the balance from the account
func (acc *Base) SubBalance(a *amount.Amount) error {
	if acc.Balance_.Less(a) {
		return ErrInsufficientBalance
	}
	acc.Balance_ = acc.Balance_.Sub(a)
	return nil
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
