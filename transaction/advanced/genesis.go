package advanced

import (
	"bytes"
	"io"

	"git.fleta.io/fleta/core/amount"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/transaction"
)

// Genesis TODO
type Genesis struct {
	transaction.Base
	Coordinate      common.Coordinate
	ObserverPubkeys []common.PublicKey
	Accounts        []*GenesisAccount
}

// NewGenesis TODO
func NewGenesis(version uint16, timestamp uint64) *Genesis {
	return &Genesis{
		Base: transaction.Base{
			Version_:   version,
			Timestamp_: timestamp,
		},
	}
}

// Hash TODO
func (tx *Genesis) Hash() (hash.Hash256, error) {
	var buffer bytes.Buffer
	if _, err := tx.WriteTo(&buffer); err != nil {
		return hash.Hash256{}, err
	}
	return hash.DoubleHash(buffer.Bytes()), nil
}

// WriteTo TODO
func (tx *Genesis) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := tx.Base.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom TODO
func (tx *Genesis) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := tx.Base.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	return read, nil
}

// GenesisAccount TODO
type GenesisAccount struct {
	Type         common.AddressType
	Amount       *amount.Amount
	PublicKey    common.PublicKey
	UnlockHeight uint32
	KeyAddresses []common.Address
}

// Hash TODO
func (ga *GenesisAccount) Hash() (hash.Hash256, error) {
	var buffer bytes.Buffer
	if _, err := ga.WriteTo(&buffer); err != nil {
		return hash.Hash256{}, err
	}
	return hash.DoubleHash(buffer.Bytes()), nil
}

// WriteTo TODO
func (ga *GenesisAccount) WriteTo(w io.Writer) (int64, error) {
	if len(ga.KeyAddresses) > 255 {
		return 0, ErrExceedAddressCount
	}

	var wrote int64
	if n, err := util.WriteUint8(w, uint8(ga.Type)); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := ga.Amount.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := ga.PublicKey.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint32(w, ga.UnlockHeight); err != nil {
		return wrote, err
	} else {
		wrote += n
	}

	if n, err := util.WriteUint8(w, uint8(len(ga.KeyAddresses))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, addr := range ga.KeyAddresses {
			if n, err := addr.WriteTo(w); err != nil {
				return wrote, err
			} else {
				wrote += n
			}
		}
	}
	return wrote, nil
}

// ReadFrom TODO
func (ga *GenesisAccount) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if v, n, err := util.ReadUint8(r); err != nil {
		return read, err
	} else {
		read += n
		ga.Type = common.AddressType(v)
	}
	if n, err := ga.Amount.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := ga.PublicKey.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if v, n, err := util.ReadUint32(r); err != nil {
		return read, err
	} else {
		read += n
		ga.UnlockHeight = v
	}

	if Len, n, err := util.ReadUint16(r); err != nil {
		return read, err
	} else {
		read += n
		ga.KeyAddresses = make([]common.Address, 0, Len)
		for i := 0; i < int(Len); i++ {
			var addr common.Address
			if n, err := addr.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				ga.KeyAddresses = append(ga.KeyAddresses, addr)
			}
		}
	}
	return read, nil
}
