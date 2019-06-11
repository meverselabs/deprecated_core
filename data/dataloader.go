package data

import (
	"github.com/fletaio/common"
	"github.com/fletaio/common/hash"
	"github.com/fletaio/core/account"
	"github.com/fletaio/core/transaction"
	"github.com/fletaio/framework/chain"
)

// Loader is an interface to provide data from the target chain
type Loader interface {
	ChainCoord() *common.Coordinate
	Accounter() *Accounter
	Transactor() *Transactor
	Provider() chain.Provider
	Eventer() *Eventer
	TargetHeight() uint32
	LastHash() hash.Hash256
	Seq(addr common.Address) uint64
	Account(addr common.Address) (account.Account, error)
	AddressByName(Name string) (common.Address, error)
	IsExistAccount(addr common.Address) (bool, error)
	IsExistAccountName(Name string) (bool, error)
	AccountData(addr common.Address, name []byte) []byte
	AccountDataKeys(addr common.Address) ([][]byte, error)
	IsExistUTXO(id uint64) (bool, error)
	UTXO(id uint64) (*transaction.UTXO, error)
}

type emptyLoader struct {
	coord *common.Coordinate
	act   *Accounter
	tran  *Transactor
	evt   *Eventer
}

// NewEmptyLoader is used for generating genesis state
func NewEmptyLoader(coord *common.Coordinate, act *Accounter, tran *Transactor, evt *Eventer) Loader {
	return &emptyLoader{
		coord: coord,
		act:   act,
		tran:  tran,
		evt:   evt,
	}
}

// ChainCoord returns the coordinate of the target chain
func (st *emptyLoader) ChainCoord() *common.Coordinate {
	return st.coord
}

// Accounter returns the accounter of the target chain
func (st *emptyLoader) Accounter() *Accounter {
	return st.act
}

// Transactor returns the transactor of the target chain
func (st *emptyLoader) Transactor() *Transactor {
	return st.tran
}

// Provider returns nil
func (st *emptyLoader) Provider() chain.Provider {
	return nil
}

// Eventer returns the eventer of the target chain
func (st *emptyLoader) Eventer() *Eventer {
	return st.evt
}

// TargetHeight returns 0
func (st *emptyLoader) TargetHeight() uint32 {
	return 0
}

// LastHash returns hash.Hash256{}
func (st *emptyLoader) LastHash() hash.Hash256 {
	return hash.Hash256{}
}

// Seq returns 0
func (st *emptyLoader) Seq(addr common.Address) uint64 {
	return 0
}

// Account returns ErrNotExistAccount
func (st *emptyLoader) Account(addr common.Address) (account.Account, error) {
	return nil, ErrNotExistAccount
}

// AddressByName returns ErrNotExistAccount
func (st *emptyLoader) AddressByName(Name string) (common.Address, error) {
	return common.Address{}, ErrNotExistAccount
}

// IsExistAccount returns false
func (st *emptyLoader) IsExistAccount(addr common.Address) (bool, error) {
	return false, nil
}

// IsExistAccountName returns false
func (st *emptyLoader) IsExistAccountName(Name string) (bool, error) {
	return false, nil
}

// AccountDataKeys returns nil
func (st *emptyLoader) AccountDataKeys(addr common.Address) ([][]byte, error) {
	return nil, nil
}

// AccountData returns nil
func (st *emptyLoader) AccountData(addr common.Address, name []byte) []byte {
	return nil
}

// IsExistUTXO returns false
func (st *emptyLoader) IsExistUTXO(id uint64) (bool, error) {
	return false, nil
}

// UTXO returns ErrNotExistUTXO
func (st *emptyLoader) UTXO(id uint64) (*transaction.UTXO, error) {
	return nil, ErrNotExistUTXO
}
