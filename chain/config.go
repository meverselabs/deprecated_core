package chain

import (
	"time"

	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/core/amount"
)

// Config TODO
type Config struct {
	Version               uint16
	GenesisHash           hash.Hash256
	UTXOCacheWriteOutTime time.Duration
	FormulationAmount     amount.Amount
}
