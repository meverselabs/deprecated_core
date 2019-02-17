package observer

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/key"
)

// Config is the configuration for the observer
type Config struct {
	ChainCoord     *common.Coordinate
	ObserverKeyMap map[common.PublicHash]string
	Key            key.Key
}
