package chain

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/amount"
	"git.fleta.io/fleta/core/chain/account"
)

// CreateAccount TODO
func CreateAccount(cn Provider, addr common.Address, keyHashes []common.PublicHash) *account.Account {
	return &account.Account{
		Address:    addr,
		ChainCoord: cn.Coordinate(),
		Balance:    amount.NewCoinAmount(0, 0),
		KeyHashes:  keyHashes,
	}
}
