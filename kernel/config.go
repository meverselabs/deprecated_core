package kernel

import (
	"github.com/fletaio/common"
)

// Config is the configuration of the kernel
type Config struct {
	ChainCoord              *common.Coordinate
	ObserverKeyMap          map[common.PublicHash]bool
	MaxBlocksPerFormulator  uint32
	MaxTransactionsPerBlock int
}
