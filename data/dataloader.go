package data

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/account"
	"git.fleta.io/fleta/core/transaction"
)

// Loader TODO
type Loader interface {
	ChainCoord() *common.Coordinate
	Accounter() *Accounter
	TargetHeight() uint32
	Seq(addr common.Address) uint64
	Account(addr common.Address) (account.Account, error)
	IsExistAccount(addr common.Address) (bool, error)
	AccountData(addr common.Address, name []byte) []byte
	UTXO(id uint64) (*transaction.UTXO, error)
}

type emptyLoader struct {
	coord *common.Coordinate
	act   *Accounter
}

// NewEmptyLoader TODO
func NewEmptyLoader(coord *common.Coordinate, act *Accounter) Loader {
	return &emptyLoader{
		coord: coord,
		act:   act,
	}
}

func (st *emptyLoader) ChainCoord() *common.Coordinate {
	return st.coord
}

func (st *emptyLoader) Accounter() *Accounter {
	return st.act
}
func (st *emptyLoader) TargetHeight() uint32 {
	return 0
}

func (st *emptyLoader) Seq(addr common.Address) uint64 {
	return 0
}

func (st *emptyLoader) Account(addr common.Address) (account.Account, error) {
	return nil, ErrNotExistAccount
}

func (st *emptyLoader) IsExistAccount(addr common.Address) (bool, error) {
	return false, nil
}

func (st *emptyLoader) AccountData(addr common.Address, name []byte) []byte {
	return nil
}

func (st *emptyLoader) UTXO(id uint64) (*transaction.UTXO, error) {
	return nil, ErrNotExistUTXO
}
