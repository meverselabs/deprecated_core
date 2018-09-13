package chain

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/amount"
	"git.fleta.io/fleta/core/chain/account"
	"git.fleta.io/fleta/core/chain/account/data"
)

// CreateAccount TODO
func CreateAccount(cn Provider, addr common.Address, keyAddresses []common.Address, data data.Data) *account.Account {
	return &account.Account{
		Address:      addr,
		ChainCoord:   cn.Coordinate(),
		Type:         addr.Type(),
		Balance:      amount.NewCoinAmount(0, 0),
		KeyAddresses: keyAddresses,
		Data:         data,
	}
}
