package reward

import (
	"github.com/fletaio/common"
	"github.com/fletaio/common/util"
	"github.com/fletaio/core/amount"
	"github.com/fletaio/core/consensus"
	"github.com/fletaio/core/data"
)

type TestNetRewarder struct {
}

// ProcessReward gives a reward to the block generator address
func (rd *TestNetRewarder) ProcessReward(addr common.Address, ctx *data.Context) error {
	var zeroAddr common.Address
	policy, err := consensus.GetConsensusPolicy(ctx.ChainCoord())
	if err != nil {
		return err
	}

	if true {
		acc, err := ctx.Account(addr)
		if err != nil {
			return err
		}

		frAcc, is := acc.(*consensus.FormulationAccount)
		if !is {
			return consensus.ErrInvalidAccountType
		}
		switch frAcc.FormulationType {
		case consensus.AlphaFormulatorType:
			rd.addRewardPower(addr, ctx, frAcc.Amount.MulC(int64(policy.AlphaEfficiency1000)).DivC(1000))
		case consensus.SigmaFormulatorType:
			rd.addRewardPower(addr, ctx, frAcc.Amount.MulC(int64(policy.SigmaEfficiency1000)).DivC(1000))
		case consensus.OmegaFormulatorType:
			rd.addRewardPower(addr, ctx, frAcc.Amount.MulC(int64(policy.OmegaEfficiency1000)).DivC(1000))
		case consensus.HyperFormulatorType:
			rd.addRewardPower(addr, ctx, frAcc.Amount.MulC(int64(policy.HyperEfficiency1000)).DivC(1000))

			keys, err := ctx.AccountDataKeys(addr)
			if err != nil {
				return err
			}
			for _, k := range keys {
				if addr, is := consensus.FromStakingKey(k); is {
					bs := ctx.AccountData(addr, k)
					if len(bs) == 0 {
						return consensus.ErrInvalidStakingAddress
					}
					StakingAmount := amount.NewAmountFromBytes(bs)

					if _, err := ctx.Account(addr); err != nil {
						if err != data.ErrNotExistAccount {
							return err
						}
						rd.removeRewardPower(addr, ctx)
					} else {
						rd.addRewardPower(addr, ctx, StakingAmount.MulC(int64(policy.StakingEfficiency1000)).DivC(1000))
					}
				}
			}
		default:
			return consensus.ErrInvalidAccountType
		}
	}

	var LastPaidHeight uint32
	if bs := ctx.AccountData(zeroAddr, []byte("consensus:last_paid_height")); len(bs) > 0 {
		LastPaidHeight = util.BytesToUint32(bs)
	}

	if ctx.TargetHeight() >= LastPaidHeight+policy.PayRewardEveryBlocks {
		keys, err := ctx.AccountDataKeys(zeroAddr)
		if err != nil {
			return err
		}

		TotalPower := amount.NewCoinAmount(0, 0)
		PowerMap := map[common.Address]*amount.Amount{}
		for _, k := range keys {
			if addr, is := FromPowerSumKey(k); is {
				PowerSum := rd.getRewardPower(addr, ctx)
				if !PowerSum.IsZero() {
					TotalPower = TotalPower.Add(PowerSum)
					PowerMap[addr] = PowerSum
				}
			}
		}

		TotalReward := policy.RewardPerBlock.MulC(int64(ctx.TargetHeight() - LastPaidHeight))
		Ratio := TotalReward.Mul(amount.COIN).Div(TotalPower)
		for addr, PowerSum := range PowerMap {
			acc, err := ctx.Account(addr)
			if err != nil {
				if err != data.ErrNotExistAccount {
					return err
				}
			} else {
				frAcc := acc.(*consensus.FormulationAccount)
				frAcc.AddBalance(PowerSum.Mul(Ratio).Div(amount.COIN))
			}
			rd.removeRewardPower(addr, ctx)
		}

		ctx.SetAccountData(zeroAddr, []byte("consensus:last_paid_height"), util.Uint32ToBytes(ctx.TargetHeight()))
	}
	return nil
}

func (rd *TestNetRewarder) addRewardPower(addr common.Address, ctx *data.Context, Power *amount.Amount) {
	var zeroAddr common.Address
	ctx.SetAccountData(zeroAddr, toPowerSumKey(addr), rd.getRewardPower(addr, ctx).Add(Power).Bytes())
}

func (rd *TestNetRewarder) removeRewardPower(addr common.Address, ctx *data.Context) {
	var zeroAddr common.Address
	ctx.SetAccountData(zeroAddr, toPowerSumKey(addr), nil)
}

func (rd *TestNetRewarder) getRewardPower(addr common.Address, ctx *data.Context) *amount.Amount {
	var zeroAddr common.Address
	var PowerSum *amount.Amount
	if bs := ctx.AccountData(zeroAddr, toPowerSumKey(addr)); len(bs) > 0 {
		PowerSum = amount.NewAmountFromBytes(bs)
	} else {
		PowerSum = amount.NewCoinAmount(0, 0)
	}
	return PowerSum
}
