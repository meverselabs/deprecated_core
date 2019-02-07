package data

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/core/account"
	"git.fleta.io/fleta/core/transaction"
)

type cache struct {
	ctx               *Context
	SeqMap            map[common.Address]uint64
	AccountMap        map[common.Address]account.Account
	AccountBalanceMap map[common.Address]*account.Balance
	AccountDataMap    map[string][]byte
	UTXOMap           map[uint64]*transaction.UTXO
}

// NewCache is used for generating genesis state
func newCache(ctx *Context) *cache {
	return &cache{
		ctx:               ctx,
		SeqMap:            map[common.Address]uint64{},
		AccountMap:        map[common.Address]account.Account{},
		AccountBalanceMap: map[common.Address]*account.Balance{},
		AccountDataMap:    map[string][]byte{},
		UTXOMap:           map[uint64]*transaction.UTXO{},
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

// TargetHeight returns cached target height when context generation
func (cc *cache) TargetHeight() uint32 {
	return cc.ctx.TargetHeight()
}

// PrevHash returns the recorded prev hash when context generation
func (cc *cache) PrevHash() hash.Hash256 {
	return cc.ctx.PrevHash()
}

// Seq returns the sequence of the account
func (cc *cache) Seq(addr common.Address) uint64 {
	if seq, has := cc.SeqMap[addr]; has {
		return seq
	} else {
		seq := cc.ctx.loader.Seq(addr)
		cc.SeqMap[addr] = seq
		return seq
	}
}

// Account returns the account instance of the address
func (cc *cache) Account(addr common.Address) (account.Account, error) {
	if acc, has := cc.AccountMap[addr]; has {
		return acc, nil
	} else {
		if acc, err := cc.ctx.loader.Account(addr); err != nil {
			return nil, err
		} else {
			cc.AccountMap[addr] = acc
			return acc, nil
		}
	}
}

// IsExistAccount checks that the account of the address is exist or not
func (cc *cache) IsExistAccount(addr common.Address) (bool, error) {
	if _, has := cc.AccountMap[addr]; has {
		return true, nil
	} else {
		return cc.ctx.loader.IsExistAccount(addr)
	}
}

// AccountBalance returns the account balance
func (cc *cache) AccountBalance(addr common.Address) (*account.Balance, error) {
	if bc, has := cc.AccountBalanceMap[addr]; has {
		return bc, nil
	} else {
		if bc, err := cc.ctx.loader.AccountBalance(addr); err != nil {
			return nil, err
		} else {
			cc.AccountBalanceMap[addr] = bc
			return bc, nil
		}
	}
}

// AccountData returns the account data
func (cc *cache) AccountData(addr common.Address, name []byte) []byte {
	key := string(addr[:]) + string(name)
	if value, has := cc.AccountDataMap[key]; has {
		return value
	} else {
		value := cc.ctx.loader.AccountData(addr, name)
		cc.AccountDataMap[key] = value
		return value
	}
}

// UTXO returns the UTXO
func (cc *cache) UTXO(id uint64) (*transaction.UTXO, error) {
	if utxo, has := cc.UTXOMap[id]; has {
		return utxo, nil
	} else {
		if utxo, err := cc.ctx.loader.UTXO(id); err != nil {
			return nil, err
		} else {
			cc.UTXOMap[id] = utxo
			return utxo, nil
		}
	}
}
