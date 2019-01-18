package consensus

import (
	"errors"
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/account"
	"git.fleta.io/fleta/core/data"
)

// Account errors
var (
	ErrInvalidSignerCount   = errors.New("invalid signer count")
	ErrInvalidAccountSigner = errors.New("invalid account signer")
)

func init() {
	data.RegisterAccount("consensus.FormulationAccount", func(t account.Type) account.Account {
		return &FormulationAccount{
			Base: account.Base{
				Type_: t,
			},
		}
	}, func(loader data.Loader, a account.Account, signers []common.PublicHash) error {
		acc := a.(*FormulationAccount)
		if len(signers) != 1 {
			return ErrInvalidSignerCount
		}
		signer := signers[0]
		if !acc.KeyHash.Equal(signer) {
			return ErrInvalidAccountSigner
		}
		return nil
	})
}

// FormulationAccount is a consensus.FormulationAccount
// It is used to indentify formulator
type FormulationAccount struct {
	account.Base
	KeyHash common.PublicHash
}

// Clone returns the clonend value of it
func (acc *FormulationAccount) Clone() account.Account {
	return &FormulationAccount{
		Base: account.Base{
			Address_: acc.Address_,
			Type_:    acc.Type_,
		},
		KeyHash: acc.KeyHash.Clone(),
	}
}

// WriteTo is a serialization function
func (acc *FormulationAccount) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := acc.Base.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := acc.KeyHash.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom is a deserialization function
func (acc *FormulationAccount) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := acc.Base.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := acc.KeyHash.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	return read, nil
}