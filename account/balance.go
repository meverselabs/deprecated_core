package account

import (
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/amount"
)

// Balance is a interface that defines common balance functions
type Balance struct {
	address    common.Address
	amountHash map[uint64]*amount.Amount
}

// NewBalance returns a Balance
func NewBalance() *Balance {
	bc := &Balance{
		amountHash: map[uint64]*amount.Amount{},
	}
	return bc
}

// Address returns the account address of the balance
func (bc *Balance) Address() common.Address {
	return bc.address
}

// Balance returns the amount of the target chain's token(or coin)
func (bc *Balance) Balance(coord *common.Coordinate) *amount.Amount {
	if a, has := bc.amountHash[coord.ID()]; has {
		return a
	} else {
		return amount.NewCoinAmount(0, 0)
	}
}

// ClearBalance removes balance and returns the amount of the target chain's token(or coin)
func (bc *Balance) ClearBalance(coord *common.Coordinate) *amount.Amount {
	a := bc.Balance(coord)
	delete(bc.amountHash, coord.ID())
	return a
}

// SubBalance sub a amount from the amount of the target chain's token(or coin)
func (bc *Balance) SubBalance(coord *common.Coordinate, a *amount.Amount) error {
	balance := bc.Balance(coord)
	if balance.Less(a) {
		return ErrInsufficientBalance
	}
	bc.amountHash[coord.ID()] = balance.Sub(a)
	return nil
}

// AddBalance add a amount to the amount of the target chain's token(or coin)
func (bc *Balance) AddBalance(coord *common.Coordinate, a *amount.Amount) {
	bc.amountHash[coord.ID()] = bc.Balance(coord).Add(a)
}

// TokenCoords returns chain's coordinates of usable tokens
func (bc *Balance) TokenCoords() []*common.Coordinate {
	list := make([]*common.Coordinate, 0, len(bc.amountHash))
	for k := range bc.amountHash {
		list = append(list, common.NewCoordinateByID(k))
	}
	return list
}

// Clone returns the clonend value of it
func (bc *Balance) Clone() *Balance {
	amountHash := map[uint64]*amount.Amount{}
	for k, v := range bc.amountHash {
		amountHash[k] = v.Clone()
	}
	c := &Balance{
		address:    bc.address.Clone(),
		amountHash: amountHash,
	}
	return c
}

// WriteTo is a serialization function
func (bc *Balance) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := bc.address.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint32(w, uint32(len(bc.amountHash))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for k, a := range bc.amountHash {
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
func (bc *Balance) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := bc.address.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if Len, n, err := util.ReadUint32(r); err != nil {
		return read, err
	} else {
		read += n
		bc.amountHash = map[uint64]*amount.Amount{}
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
					bc.amountHash[k] = a
				}
			}
		}
	}
	return read, nil
}
