package kernel

import (
	"git.fleta.io/fleta/common"
)

// Config is the configuration of the kernel
type Config struct {
	ChainCoord     *common.Coordinate
	ObserverKeyMap map[common.PublicHash]bool
}
