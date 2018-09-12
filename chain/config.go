package chain

import (
	"time"

	"git.fleta.io/fleta/core/amount"
)

// Config TODO
type Config struct {
	Version               uint16
	UTXOCacheWriteOutTime time.Duration
	FormulationCost       *amount.Amount
	MultiSigAccountCost   *amount.Amount
	DustAmount            *amount.Amount
}
