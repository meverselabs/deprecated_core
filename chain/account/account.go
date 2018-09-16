package account

import (
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/amount"
)

// Account TODO
type Account struct {
	Address    common.Address
	ChainCoord *common.Coordinate
	Balance    *amount.Amount
	Seq        uint64
	KeyHashes  []common.PublicHash
}

// WriteTo TODO
func (acc *Account) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := acc.Address.WriteTo(w); err != nil {
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

	if n, err := util.WriteUint16(w, uint16(len(acc.KeyHashes))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, ka := range acc.KeyHashes {
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
		acc.KeyHashes = make([]common.PublicHash, 0, Len)
		for i := 0; i < int(Len); i++ {
			var ka common.PublicHash
			if n, err := ka.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				acc.KeyHashes = append(acc.KeyHashes, ka)
			}
		}
	}
	return read, nil
}
