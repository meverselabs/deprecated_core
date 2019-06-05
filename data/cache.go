package data

import (
	"github.com/fletaio/common"
	"github.com/fletaio/common/hash"
	"github.com/fletaio/core/account"
	"github.com/fletaio/core/transaction"
)

type cache struct {
	ctx                *Context
	SeqMap             map[common.Address]uint64
	AccountMap         map[common.Address]account.Account
	AccountNameMap     map[string]common.Address
	AccountDataMap     map[string][]byte
	AccountDataKeysMap map[common.Address][][]byte
	UTXOMap            map[uint64]*transaction.UTXO
}

// NewCache is used for generating genesis state
func newCache(ctx *Context) *cache {
	return &cache{
		ctx:                ctx,
		SeqMap:             map[common.Address]uint64{},
		AccountMap:         map[common.Address]account.Account{},
		AccountNameMap:     map[string]common.Address{},
		AccountDataMap:     map[string][]byte{},
		AccountDataKeysMap: map[common.Address][][]byte{},
		UTXOMap:            map[uint64]*transaction.UTXO{},
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

// Eventer returns the eventer of the target chain
func (cc *cache) Eventer() *Eventer {
	return cc.ctx.Eventer()
}

// TargetHeight returns cached target height when context generation
func (cc *cache) TargetHeight() uint32 {
	return cc.ctx.TargetHeight()
}

// LastHash returns the recorded prev hash when context generation
func (cc *cache) LastHash() hash.Hash256 {
	return cc.ctx.LastHash()
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

// AddressByName returns the account address of the name
func (cc *cache) AddressByName(Name string) (common.Address, error) {
	if addr, has := cc.AccountNameMap[Name]; has {
		return addr, nil
	} else {
		if addr, err := cc.ctx.loader.AddressByName(Name); err != nil {
			return common.Address{}, err
		} else {
			cc.AccountNameMap[Name] = addr
			return addr, nil
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

// IsExistAccountName checks that the account of the name is exist or not
func (cc *cache) IsExistAccountName(Name string) (bool, error) {
	if _, has := cc.AccountNameMap[Name]; has {
		return true, nil
	} else {
		return cc.ctx.loader.IsExistAccountName(Name)
	}
}

// AccountDataKeys returns all data keys of the account in the context
func (cc *cache) AccountDataKeys(addr common.Address) ([][]byte, error) {
	if keys, has := cc.AccountDataKeysMap[addr]; has {
		return keys, nil
	} else {
		keys, err := cc.ctx.loader.AccountDataKeys(addr)
		if err != nil {
			return nil, err
		}
		cc.AccountDataKeysMap[addr] = keys
		return keys, nil
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

// IsExistUTXO checks that the utxo of the id is exist or not
func (cc *cache) IsExistUTXO(id uint64) (bool, error) {
	if _, has := cc.UTXOMap[id]; has {
		return true, nil
	} else {
		return false, nil
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
