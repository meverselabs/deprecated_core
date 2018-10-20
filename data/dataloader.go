package data

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/account"
	"git.fleta.io/fleta/core/db"
	"git.fleta.io/fleta/core/transaction"
)

// Loader TODO
type Loader interface {
	ChainCoord() *common.Coordinate
	Seq(addr common.Address) uint64
	Account(addr common.Address) (account.Account, error)
	AccountData(addr common.Address, name []byte) []byte
	UTXO(id uint64) (*transaction.UTXO, error)
}

type emptyLoader struct {
	coord *common.Coordinate
}

// NewEmptyLoader TODO
func NewEmptyLoader(coord *common.Coordinate) Loader {
	return &emptyLoader{
		coord: coord,
	}
}

func (st *emptyLoader) ChainCoord() *common.Coordinate {
	return st.coord
}

func (st *emptyLoader) Seq(addr common.Address) uint64 {
	return 0
}

func (st *emptyLoader) Account(addr common.Address) (account.Account, error) {
	return nil, db.ErrNotExistKey
}

func (st *emptyLoader) AccountData(addr common.Address, name []byte) []byte {
	return nil
}

func (st *emptyLoader) UTXO(id uint64) (*transaction.UTXO, error) {
	return nil, db.ErrNotExistKey
}
