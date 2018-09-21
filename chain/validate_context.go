package chain

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/store"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/chain/account"
	"git.fleta.io/fleta/core/transaction"
)

// ValidationContext TODO
type ValidationContext struct {
	CurrentTxHash     hash.Hash256
	AccountHash       map[string]*account.Account
	DeleteAccountHash map[string]*account.Account
	AccountDataHash   map[string][]byte
	SpentUTXOHash     map[uint64]bool
	UTXOHash          map[uint64]*transaction.TxOut
}

// NewValidationContext TODO
func NewValidationContext() *ValidationContext {
	ctx := &ValidationContext{
		AccountHash:       map[string]*account.Account{},
		DeleteAccountHash: map[string]*account.Account{},
		AccountDataHash:   map[string][]byte{},
		SpentUTXOHash:     map[uint64]bool{},
		UTXOHash:          map[uint64]*transaction.TxOut{},
	}
	return ctx
}

// IsExistAccount TODO
func (ctx *ValidationContext) IsExistAccount(cn Provider, addr common.Address) (bool, error) {
	if _, err := ctx.loadAccount(cn, addr, false); err != nil {
		if err != store.ErrNotExistKey {
			return false, err
		} else {
			return false, nil
		}
	} else {
		return true, nil
	}
}

// LoadAccount TODO
func (ctx *ValidationContext) LoadAccount(cn Provider, addr common.Address) (*account.Account, error) {
	return ctx.loadAccount(cn, addr, true)
}

func (ctx *ValidationContext) loadAccount(cn Provider, addr common.Address, checkLock bool) (*account.Account, error) {
	if _, has := ctx.DeleteAccountHash[string(addr[:])]; has {
		return nil, ErrDeletedAccount
	}

	targetAcc, has := ctx.AccountHash[string(addr[:])]
	if !has {
		acc, err := cn.Account(addr)
		if err != nil {
			return nil, err
		}
		if checkLock {
			switch acc.Address.Type() {
			case LockedAddressType:
				bs, err := ctx.AccountData(cn, acc.Address, "UnlockHeight")
				if err != nil {
					return nil, err
				}
				if cn.Height() < util.BytesToUint32(bs) {
					return nil, ErrLockedAccount
				}
			}
		}
		targetAcc = acc
		ctx.AccountHash[string(addr[:])] = targetAcc
	}
	return targetAcc, nil
}

// AccountData TODO
func (ctx *ValidationContext) AccountData(cn Provider, addr common.Address, name string) ([]byte, error) {
	key := toAccountDataKey(addr, name)
	data, has := ctx.AccountDataHash[string(key)]
	if !has {
		bs, err := cn.AccountData(addr, name)
		if err != nil {
			return nil, err
		}
		data = bs
	}
	return data, nil
}

// Unspent TODO
func (ctx *ValidationContext) Unspent(cn Provider, height uint32, index uint16, n uint16) (*UTXO, error) {
	id := transaction.MarshalID(height, index, n)
	if _, has := ctx.SpentUTXOHash[id]; has {
		return nil, ErrDoubleSpent
	}
	if utxo, err := cn.Unspent(height, index, n); err != nil {
		return nil, err
	} else {
		return utxo, nil
	}
}
