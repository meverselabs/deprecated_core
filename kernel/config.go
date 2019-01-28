package kernel

import (
	"time"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/key"
)

// Config TODO
type Config struct {
	ChainCoord        *common.Coordinate
	ObserverKeys      []string
	FormulatorAddress common.Address
	GenTimeThreshold  time.Duration
	FormulatorKey     key.Key
}
