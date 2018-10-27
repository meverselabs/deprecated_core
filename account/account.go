package account

import (
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/amount"
)

// Type is using when serealization and deserialization account
type Type uint8

// Account is a interface that define the basic account functions
type Account interface {
	Address() common.Address
	SetType(t Type)
	Type() Type
	Balance(coord *common.Coordinate) *amount.Amount
	SetBalance(coord *common.Coordinate, a *amount.Amount)
	TokenCoords() []*common.Coordinate
	Clone() Account
	io.WriterTo
	io.ReaderFrom
}

// Base is the parts of account functions that are not changed by derived one
type Base struct {
	Address_    common.Address
	Type_       Type
	BalanceHash map[uint64]*amount.Amount
}

// NewBase returns a account Base
func NewBase() *Base {
	acc := &Base{
		BalanceHash: map[uint64]*amount.Amount{},
	}
	return acc
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

// Balance returns the amount of the target chain's token(or coin)
func (acc *Base) Balance(coord *common.Coordinate) *amount.Amount {
	if a, has := acc.BalanceHash[coord.ID()]; has {
		return a
	} else {
		return amount.NewCoinAmount(0, 0)
	}
}

// SetBalance set the amount of the target chain's token(or coin)
func (acc *Base) SetBalance(coord *common.Coordinate, a *amount.Amount) {
	if a.IsZero() {
		delete(acc.BalanceHash, coord.ID())
	} else {
		acc.BalanceHash[coord.ID()] = a
	}
}

// TokenCoords returns chain's coordinates of usable tokens
func (acc *Base) TokenCoords() []*common.Coordinate {
	list := make([]*common.Coordinate, 0, len(acc.BalanceHash))
	for k := range acc.BalanceHash {
		list = append(list, common.NewCoordinateByID(k))
	}
	return list
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

	if n, err := util.WriteUint32(w, uint32(len(acc.BalanceHash))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for k, a := range acc.BalanceHash {
			if n, err := util.WriteUint64(w, k); err != nil {
				return wrote, err
			} else {
				wrote += n
			}
			if n, err := a.WriteTo(w); err != nil {
				return wrote, err
			} else {
				wrote += n
			}
		}
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

	if Len, n, err := util.ReadUint32(r); err != nil {
		return read, err
	} else {
		read += n
		acc.BalanceHash = map[uint64]*amount.Amount{}
		for i := 0; i < int(Len); i++ {
			if k, n, err := util.ReadUint64(r); err != nil {
				return read, err
			} else {
				read += n
				a := amount.NewCoinAmount(0, 0)
				if n, err := a.ReadFrom(r); err != nil {
					return read, err
				} else {
					read += n
					acc.BalanceHash[k] = a
				}
			}
		}
	}
	return read, nil
}
