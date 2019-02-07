package account

import (
	"io"
	"sort"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/amount"
)

// Balance is a interface that defines common balance functions
type Balance struct {
	address   common.Address
	amountMap map[uint64]*amount.Amount
}

// NewBalance returns a Balance
func NewBalance() *Balance {
	bc := &Balance{
		amountMap: map[uint64]*amount.Amount{},
	}
	return bc
}

// Address returns the account address of the balance
func (bc *Balance) Address() common.Address {
	return bc.address
}

// Balance returns the amount of the target chain's token(or coin)
func (bc *Balance) Balance(coord *common.Coordinate) *amount.Amount {
	if a, has := bc.amountMap[coord.ID()]; has {
		return a
	} else {
		return amount.NewCoinAmount(0, 0)
	}
}

// ClearBalance removes balance and returns the amount of the target chain's token(or coin)
func (bc *Balance) ClearBalance(coord *common.Coordinate) *amount.Amount {
	a := bc.Balance(coord)
	delete(bc.amountMap, coord.ID())
	return a
}

// SubBalance sub a amount from the amount of the target chain's token(or coin)
func (bc *Balance) SubBalance(coord *common.Coordinate, a *amount.Amount) error {
	balance := bc.Balance(coord)
	if balance.Less(a) {
		return ErrInsufficientBalance
	}
	bc.amountMap[coord.ID()] = balance.Sub(a)
	return nil
}

// AddBalance add a amount to the amount of the target chain's token(or coin)
func (bc *Balance) AddBalance(coord *common.Coordinate, a *amount.Amount) {
	bc.amountMap[coord.ID()] = bc.Balance(coord).Add(a)
}

// TokenCoords returns chain's coordinates of usable tokens
func (bc *Balance) TokenCoords() []*common.Coordinate {
	list := make([]*common.Coordinate, 0, len(bc.amountMap))
	for k := range bc.amountMap {
		list = append(list, common.NewCoordinateByID(k))
	}
	return list
}

// Clone returns the clonend value of it
func (bc *Balance) Clone() *Balance {
	amountMap := map[uint64]*amount.Amount{}
	for k, v := range bc.amountMap {
		amountMap[k] = v.Clone()
	}
	c := &Balance{
		address:   bc.address.Clone(),
		amountMap: amountMap,
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
	if n, err := util.WriteUint32(w, uint32(len(bc.amountMap))); err != nil {
		return wrote, err
	} else {
		wrote += n
		keys := make([]uint64, 0, len(bc.amountMap))
		for k := range bc.amountMap {
			keys = append(keys, k)
		}
		sort.Sort(uint64Slice(keys))
		for _, k := range keys {
			a := bc.amountMap[k]
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
		bc.amountMap = map[uint64]*amount.Amount{}
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
					bc.amountMap[k] = a
				}
			}
		}
	}
	return read, nil
}

type uint64Slice []uint64

func (p uint64Slice) Len() int           { return len(p) }
func (p uint64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p uint64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
