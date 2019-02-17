package kernel

import (
	"time"

	"git.fleta.io/fleta/common"
)

// Config is the configuration of the kernel
type Config struct {
	ChainCoord       *common.Coordinate
	ObserverKeys     []common.PublicHash
	GenTimeThreshold time.Duration
}
