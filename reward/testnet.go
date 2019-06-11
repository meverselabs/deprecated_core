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
		var PowerSum *amount.Amount
		if bs := ctx.AccountData(zeroAddr, toPowerSumKey(addr)); len(bs) > 0 {
			PowerSum = amount.NewAmountFromBytes(bs)
		} else {
			PowerSum = amount.NewCoinAmount(0, 0)
		}

		acc, err := ctx.Account(addr)
		if err != nil {
			return err
		}

		frAcc, is := acc.(*consensus.FormulationAccount)
		if !is {
			return consensus.ErrInvalidAccountType
		}
		var Power *amount.Amount
		switch frAcc.FormulationType {
		case consensus.AlphaFormulatorType:
			Power = frAcc.Amount.MulC(int64(policy.AlphaEfficiency1000)).DivC(1000)
		case consensus.SigmaFormulatorType:
			Power = frAcc.Amount.MulC(int64(policy.SigmaEfficiency1000)).DivC(1000)
		case consensus.OmegaFormulatorType:
			Power = frAcc.Amount.MulC(int64(policy.OmegaEfficiency1000)).DivC(1000)
		case consensus.HyperFormulatorType:
			OwnPower := frAcc.Amount.MulC(int64(policy.HyperEfficiency1000)).DivC(1000)
			StakingPower := frAcc.StakingAmount.MulC(int64(policy.StakingEfficiency1000)).DivC(1000)
			Power = OwnPower.Add(StakingPower)
		default:
			return consensus.ErrInvalidAccountType
		}
		PowerSum = PowerSum.Add(Power)

		ctx.SetAccountData(zeroAddr, toPowerSumKey(addr), PowerSum.Bytes())
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
			if addr, is := getPowerSumKey(k); is {
				if bs := ctx.AccountData(zeroAddr, k); len(bs) > 0 {
					PowerSum := amount.NewAmountFromBytes(bs)
					TotalPower = TotalPower.Add(PowerSum)
					PowerMap[addr] = PowerSum
				}
			}
		}

		TotalReward := policy.RewardPerBlock.MulC(int64(ctx.TargetHeight() - LastPaidHeight))
		Ratio := float64(TotalReward.Int64()) / float64(TotalPower.Int64())
		for addr, PowerSum := range PowerMap {
			acc, err := ctx.Account(addr)
			if err != nil {
				if err != data.ErrNotExistAccount {
					return err
				}
			} else {
				frAcc := acc.(*consensus.FormulationAccount)
				frAcc.AddBalance(PowerSum.MulC(int64(Ratio * 1000000)).DivC(1000000))
			}
			ctx.SetAccountData(zeroAddr, toPowerSumKey(addr), nil)
		}

		ctx.SetAccountData(zeroAddr, []byte("consensus:last_paid_height"), util.Uint32ToBytes(ctx.TargetHeight()))
	}
	return nil
}
