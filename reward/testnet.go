package reward

import (
	"github.com/fletaio/common"
	"github.com/fletaio/core/amount"
	"github.com/fletaio/core/data"
)

type TestNetRewarder struct {
}

// ProcessReward gives a reward to the block generator address
func (rd *TestNetRewarder) ProcessReward(addr common.Address, ctx *data.Context) error {
	acc, err := ctx.Account(addr)
	if err != nil {
		return err
	}
	acc.AddBalance(amount.NewCoinAmount(1, 0))
	return nil
}
