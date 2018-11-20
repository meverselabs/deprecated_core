package data

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/core/account"
	"git.fleta.io/fleta/core/transaction"
)

type cache struct {
	ctx                *Context
	SeqHash            map[common.Address]uint64
	AccountHash        map[common.Address]account.Account
	AccountBalanceHash map[common.Address]*account.Balance
	AccountDataHash    map[string][]byte
	UTXOHash           map[uint64]*transaction.UTXO
}

// NewCache is used for generating genesis state
func newCache(ctx *Context) *cache {
	return &cache{
		ctx:                ctx,
		SeqHash:            map[common.Address]uint64{},
		AccountHash:        map[common.Address]account.Account{},
		AccountBalanceHash: map[common.Address]*account.Balance{},
		AccountDataHash:    map[string][]byte{},
		UTXOHash:           map[uint64]*transaction.UTXO{},
	}
}

// ChainCoord returns the coordinate of the target chain
func (cc *cache) ChainCoord() *common.Coordinate {
	return cc.ctx.ChainCoord()
}

// Accounter returns the accounter of the target chain
func (cc *cache) Accounter() *Accounter {
	return cc.ctx.Accounter()
}

// Transactor returns the transactor of the target chain
func (cc *cache) Transactor() *Transactor {
	return cc.ctx.Transactor()
}

// TargetHeight returns 0
func (cc *cache) TargetHeight() uint32 {
	return cc.ctx.TargetHeight()
}

// LastBlockHash returns hash.Hash256{}
func (cc *cache) LastBlockHash() hash.Hash256 {
	return cc.ctx.LastBlockHash()
}

// Seq returns the sequence of the account
func (cc *cache) Seq(addr common.Address) uint64 {
	if seq, has := cc.SeqHash[addr]; has {
		return seq
	} else {
		seq := cc.ctx.loader.Seq(addr)
		cc.SeqHash[addr] = seq
		return seq
	}
}

// Account returns the account instance of the address
func (cc *cache) Account(addr common.Address) (account.Account, error) {
	if acc, has := cc.AccountHash[addr]; has {
		return acc, nil
	} else {
		if acc, err := cc.ctx.loader.Account(addr); err != nil {
			return nil, err
		} else {
			cc.AccountHash[addr] = acc
			return acc, nil
		}
	}
}

// IsExistAccount checks that the account of the address is exist or not
func (cc *cache) IsExistAccount(addr common.Address) (bool, error) {
	if _, has := cc.AccountHash[addr]; has {
		return true, nil
	} else {
		return cc.ctx.loader.IsExistAccount(addr)
	}
}

// AccountBalance returns the account balance
func (cc *cache) AccountBalance(addr common.Address) (*account.Balance, error) {
	if bc, has := cc.AccountBalanceHash[addr]; has {
		return bc, nil
	} else {
		if bc, err := cc.ctx.loader.AccountBalance(addr); err != nil {
			return nil, err
		} else {
			cc.AccountBalanceHash[addr] = bc
			return bc, nil
		}
	}
}

// AccountData returns the account data
func (cc *cache) AccountData(addr common.Address, name []byte) []byte {
	key := string(addr[:]) + string(name)
	if value, has := cc.AccountDataHash[key]; has {
		return value
	} else {
		value := cc.ctx.loader.AccountData(addr, name)
		cc.AccountDataHash[key] = value
		return value
	}
}

// UTXO returns the UTXO
func (cc *cache) UTXO(id uint64) (*transaction.UTXO, error) {
	if utxo, has := cc.UTXOHash[id]; has {
		return utxo, nil
	} else {
		if utxo, err := cc.ctx.loader.UTXO(id); err != nil {
			return nil, err
		} else {
			cc.UTXOHash[id] = utxo
			return utxo, nil
		}
	}
}
