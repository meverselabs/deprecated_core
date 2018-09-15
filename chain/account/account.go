package account

import (
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/amount"
)

// Account TODO
type Account struct {
	Address      common.Address
	ChainCoord   *common.Coordinate
	Type         common.AddressType
	Balance      *amount.Amount
	Seq          uint64
	KeyAddresses []common.Address
}

// WriteTo TODO
func (acc *Account) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := acc.Address.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint8(w, uint8(acc.Type)); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := acc.Balance.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint64(w, acc.Seq); err != nil {
		return wrote, err
	} else {
		wrote += n
	}

	if n, err := util.WriteUint16(w, uint16(len(acc.KeyAddresses))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, ka := range acc.KeyAddresses {
			if n, err := ka.WriteTo(w); err != nil {
				return wrote, err
			} else {
				wrote += n
			}
		}
	}
	return wrote, nil
}

// ReadFrom TODO
func (acc *Account) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := acc.Address.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if v, n, err := util.ReadUint8(r); err != nil {
		return read, err
	} else {
		read += n
		acc.Type = common.AddressType(v)
	}
	if n, err := acc.Balance.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if v, n, err := util.ReadUint64(r); err != nil {
		return read, err
	} else {
		read += n
		acc.Seq = v
	}

	if Len, n, err := util.ReadUint16(r); err != nil {
		return read, err
	} else {
		read += n
		acc.KeyAddresses = make([]common.Address, 0, Len)
		for i := 0; i < int(Len); i++ {
			var ka common.Address
			if n, err := ka.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				acc.KeyAddresses = append(acc.KeyAddresses, ka)
			}
		}
	}
	return read, nil
}
