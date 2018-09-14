package chain

import (
	"bytes"
	"io"

	"git.fleta.io/fleta/core/amount"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
)

// Genesis TODO
type Genesis struct {
	Coordinate      common.Coordinate
	ObserverPubkeys []common.PublicKey
	Accounts        []*GenesisAccount
}

// Hash TODO
func (gn *Genesis) Hash() (hash.Hash256, error) {
	var buffer bytes.Buffer
	if _, err := gn.WriteTo(&buffer); err != nil {
		return hash.Hash256{}, err
	}
	return hash.DoubleHash(buffer.Bytes()), nil
}

// WriteTo TODO
func (gn *Genesis) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := gn.Coordinate.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}

	if n, err := util.WriteUint8(w, uint8(len(gn.ObserverPubkeys))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, ka := range gn.ObserverPubkeys {
			if n, err := ka.WriteTo(w); err != nil {
				return wrote, err
			} else {
				wrote += n
			}
		}
	}

	if n, err := util.WriteUint16(w, uint16(len(gn.Accounts))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, acc := range gn.Accounts {
			if n, err := acc.WriteTo(w); err != nil {
				return wrote, err
			} else {
				wrote += n
			}
		}
	}
	return wrote, nil
}

// ReadFrom TODO
func (gn *Genesis) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := gn.Coordinate.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}

	if Len, n, err := util.ReadUint8(r); err != nil {
		return read, err
	} else {
		read += n
		gn.ObserverPubkeys = make([]common.PublicKey, 0, Len)
		for i := 0; i < int(Len); i++ {
			var pubkey common.PublicKey
			if n, err := pubkey.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				gn.ObserverPubkeys = append(gn.ObserverPubkeys, pubkey)
			}
		}
	}

	if Len, n, err := util.ReadUint16(r); err != nil {
		return read, err
	} else {
		read += n
		gn.Accounts = make([]*GenesisAccount, 0, Len)
		for i := 0; i < int(Len); i++ {
			ga := &GenesisAccount{
				Amount:       amount.NewCoinAmount(0, 0),
				KeyAddresses: []common.Address{},
			}
			if n, err := ga.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				gn.Accounts = append(gn.Accounts, ga)
			}
		}
	}
	return read, nil
}

// GenesisAccount TODO
type GenesisAccount struct {
	Type             common.AddressType
	Amount           *amount.Amount
	PublicKey        common.PublicKey
	UnlockHeight     uint32
	Required         uint8
	KeyAddresses     []common.Address
	GeneratedAddress common.Address
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
	if n, err := util.WriteUint8(w, ga.Required); err != nil {
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

	if n, err := ga.GeneratedAddress.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
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
	if v, n, err := util.ReadUint8(r); err != nil {
		return read, err
	} else {
		read += n
		ga.Required = v
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

	if n, err := ga.GeneratedAddress.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	return read, nil
}
