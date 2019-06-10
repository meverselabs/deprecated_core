package consensus

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/fletaio/common"
	"github.com/fletaio/core/account"
	"github.com/fletaio/core/amount"
	"github.com/fletaio/core/data"
)

func init() {
	data.RegisterAccount("consensus.CommunityFormulationAccount", func(t account.Type) account.Account {
		return &CommunityFormulationAccount{
			Base: account.Base{
				Type_:    t,
				Balance_: amount.NewCoinAmount(0, 0),
			},
			Policy: &CommunityPolicy{
				MinimumStaking: amount.NewCoinAmount(0, 0),
				MaximumStaking: amount.NewCoinAmount(0, 0),
			},
			Amount:        amount.NewCoinAmount(0, 0),
			StakingAmount: amount.NewCoinAmount(0, 0),
		}
	}, func(loader data.Loader, a account.Account, signers []common.PublicHash) error {
		acc := a.(*CommunityFormulationAccount)
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

// CommunityFormulationAccount is a consensus.CommunityFormulationAccount
// It is used to indentify community formulator that supports the staking system
type CommunityFormulationAccount struct {
	account.Base
	KeyHash       common.PublicHash
	Policy        *CommunityPolicy
	Amount        *amount.Amount
	StakingAmount *amount.Amount
}

// Clone returns the clonend value of it
func (acc *CommunityFormulationAccount) Clone() account.Account {
	return &CommunityFormulationAccount{
		Base: account.Base{
			Type_:    acc.Type_,
			Address_: acc.Address_,
			Balance_: acc.Balance(),
		},
		KeyHash:       acc.KeyHash.Clone(),
		Policy:        acc.Policy.Clone(),
		Amount:        acc.Amount.Clone(),
		StakingAmount: acc.StakingAmount.Clone(),
	}
}

// WriteTo is a serialization function
func (acc *CommunityFormulationAccount) WriteTo(w io.Writer) (int64, error) {
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
	if n, err := acc.Policy.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := acc.Amount.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := acc.StakingAmount.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom is a deserialization function
func (acc *CommunityFormulationAccount) ReadFrom(r io.Reader) (int64, error) {
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
	if n, err := acc.Policy.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := acc.Amount.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := acc.StakingAmount.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	return read, nil
}

// MarshalJSON is a marshaler function
func (acc *CommunityFormulationAccount) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString(`{`)
	buffer.WriteString(`"address":`)
	if bs, err := acc.Address_.MarshalJSON(); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`,`)
	buffer.WriteString(`"type":`)
	if bs, err := json.Marshal(acc.Type_); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`,`)
	buffer.WriteString(`"key_hash":`)
	if bs, err := acc.KeyHash.MarshalJSON(); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`,`)
	buffer.WriteString(`"policy":`)
	if bs, err := acc.Policy.MarshalJSON(); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`,`)
	buffer.WriteString(`"amount":`)
	if bs, err := acc.StakingAmount.MarshalJSON(); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`,`)
	buffer.WriteString(`"staking_amount":`)
	if bs, err := acc.StakingAmount.MarshalJSON(); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`}`)
	return buffer.Bytes(), nil
}
