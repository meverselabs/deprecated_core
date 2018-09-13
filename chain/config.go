package chain

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/amount"
)

// Config TODO
type Config struct {
	Version             uint16
	Coordinate          common.Coordinate
	FormulationCost     *amount.Amount
	MultiSigAccountCost *amount.Amount
	DustAmount          *amount.Amount
}
