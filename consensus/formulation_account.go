package consensus

import (
	"errors"
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/account"
	"git.fleta.io/fleta/core/accounter"
	"git.fleta.io/fleta/core/amount"
)

// Account errors
var (
	ErrInvalidSignerCount   = errors.New("invalid signer count")
	ErrInvalidAccountSigner = errors.New("invalid account signer")
)

func init() {
	accounter.RegisterHandler("formulation.Account", func(t account.Type) account.Account {
		return &Account{
			Base: account.Base{
				Type_:       t,
				BalanceHash: map[uint64]*amount.Amount{},
			},
		}
	}, func(a account.Account, signers []common.PublicHash) error {
		acc := a.(*Account)
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

// Account TODO
type Account struct {
	account.Base
	KeyHash common.PublicHash
}

// Clone TODO
func (acc *Account) Clone() account.Account {
	balanceHash := map[uint64]*amount.Amount{}
	for k, v := range acc.BalanceHash {
		balanceHash[k] = v.Clone()
	}
	return &Account{
		Base: account.Base{
			Address_:    acc.Address_,
			Type_:       acc.Type_,
			BalanceHash: balanceHash,
		},
		KeyHash: acc.KeyHash.Clone(),
	}
}

// WriteTo TODO
func (acc *Account) WriteTo(w io.Writer) (int64, error) {
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

// ReadFrom TODO
func (acc *Account) ReadFrom(r io.Reader) (int64, error) {
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
