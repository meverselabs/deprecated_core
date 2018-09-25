package chain

import (
	"git.fleta.io/fleta/core/amount"
)

// Config TODO
type Config struct {
	Version             uint16
	FormulationCost     *amount.Amount
	SingleAccountCost   *amount.Amount
	MultiSigAccountCost *amount.Amount
	DustAmount          *amount.Amount
	AccountBaseFee      *amount.Amount
	UTXOBaseFee         *amount.Amount
	OpenAccountCost     *amount.Amount
}

// Clone TODO
func (c *Config) Clone() *Config {
	cc := &Config{
		Version:             c.Version,
		FormulationCost:     c.FormulationCost.Clone(),
		SingleAccountCost:   c.SingleAccountCost.Clone(),
		MultiSigAccountCost: c.MultiSigAccountCost.Clone(),
		DustAmount:          c.DustAmount.Clone(),
		AccountBaseFee:      c.AccountBaseFee.Clone(),
		UTXOBaseFee:         c.UTXOBaseFee.Clone(),
		OpenAccountCost:     c.OpenAccountCost.Clone(),
	}
	return cc
}
