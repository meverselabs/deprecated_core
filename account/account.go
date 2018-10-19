package account

import (
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/amount"
)

// Type TODO
type Type uint8

// Account TODO
type Account interface {
	Address() common.Address
	SetType(t Type)
	Type() Type
	Seq() uint64
	AddSeq()
	Balance(coord *common.Coordinate) *amount.Amount
	SetBalance(coord *common.Coordinate, a *amount.Amount)
	Clone() Account
	io.WriterTo
	io.ReaderFrom
}

// Base TODO
type Base struct {
	Address_    common.Address
	Type_       Type
	Seq_        uint64
	BalanceHash map[uint64]*amount.Amount
}

// NewBase TODO
func NewBase() *Base {
	acc := &Base{
		BalanceHash: map[uint64]*amount.Amount{},
	}
	return acc
}

// Address TODO
func (acc *Base) Address() common.Address {
	return acc.Address_
}

// SetType TODO
func (acc *Base) SetType(t Type) {
	acc.Type_ = t
}

// Type TODO
func (acc *Base) Type() Type {
	return acc.Type_
}

// Seq TODO
func (acc *Base) Seq() uint64 {
	return acc.Seq_
}

// AddSeq TODO
func (acc *Base) AddSeq() {
	acc.Seq_++
}

// Balance TODO
func (acc *Base) Balance(coord *common.Coordinate) *amount.Amount {
	if a, has := acc.BalanceHash[coord.ID()]; has {
		return a
	} else {
		return amount.NewCoinAmount(0, 0)
	}
}

// SetBalance TODO
func (acc *Base) SetBalance(coord *common.Coordinate, a *amount.Amount) {
	acc.BalanceHash[coord.ID()] = a
}

// WriteTo TODO
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
	if n, err := util.WriteUint64(w, acc.Seq_); err != nil {
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

// ReadFrom TODO
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
	if v, n, err := util.ReadUint64(r); err != nil {
		return read, err
	} else {
		read += n
		acc.Seq_ = v
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
