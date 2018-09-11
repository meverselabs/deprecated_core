package chain

import (
	"time"

	"git.fleta.io/fleta/common/hash"
)

// Config TODO
type Config struct {
	Version               uint16
	GenesisHash           hash.Hash256
	UTXOCacheWriteOutTime time.Duration
}
