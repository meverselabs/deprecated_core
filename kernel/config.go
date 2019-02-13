package kernel

import (
	"time"

	"git.fleta.io/fleta/common"
)

// Config TODO
type Config struct {
	ChainCoord       *common.Coordinate
	ObserverKeys     []string
	GenTimeThreshold time.Duration
}
