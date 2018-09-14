package chain

import (
	"git.fleta.io/fleta/core/amount"
)

// Config TODO
type Config struct {
	Version             uint16
	FormulationCost     *amount.Amount
	MultiSigAccountCost *amount.Amount
	DustAmount          *amount.Amount
}

// Clone TODO
func (c *Config) Clone() *Config {
	cc := &Config{
		Version:             c.Version,
		FormulationCost:     c.FormulationCost.Clone(),
		MultiSigAccountCost: c.MultiSigAccountCost.Clone(),
		DustAmount:          c.DustAmount.Clone(),
	}
	return cc
}
