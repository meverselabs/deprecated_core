package data

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/core/account"
	"git.fleta.io/fleta/core/transaction"
)

// Loader is an interface to provide data from the target chain
type Loader interface {
	ChainCoord() *common.Coordinate
	Accounter() *Accounter
	Transactor() *Transactor
	TargetHeight() uint32
	LastHash() hash.Hash256
	Seq(addr common.Address) uint64
	Account(addr common.Address) (account.Account, error)
	IsExistAccount(addr common.Address) (bool, error)
	AccountBalance(addr common.Address) (*account.Balance, error)
	AccountData(addr common.Address, name []byte) []byte
	UTXO(id uint64) (*transaction.UTXO, error)
}

type emptyLoader struct {
	coord *common.Coordinate
	act   *Accounter
	tran  *Transactor
}

// NewEmptyLoader is used for generating genesis state
func NewEmptyLoader(coord *common.Coordinate, act *Accounter, tran *Transactor) Loader {
	return &emptyLoader{
		coord: coord,
		act:   act,
		tran:  tran,
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

// IsExistAccount returns false
func (st *emptyLoader) IsExistAccount(addr common.Address) (bool, error) {
	return false, nil
}

// AccountBalance returns ErrNotExistAccount
func (st *emptyLoader) AccountBalance(addr common.Address) (*account.Balance, error) {
	return nil, ErrNotExistAccount
}

// AccountData returns nil
func (st *emptyLoader) AccountData(addr common.Address, name []byte) []byte {
	return nil
}

// UTXO returns ErrNotExistUTXO
func (st *emptyLoader) UTXO(id uint64) (*transaction.UTXO, error) {
	return nil, ErrNotExistUTXO
}
