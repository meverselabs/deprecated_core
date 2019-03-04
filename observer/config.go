package observer

import (
	"github.com/fletaio/common"
	"github.com/fletaio/core/key"
)

// Config is the configuration for the observer
type Config struct {
	ChainCoord     *common.Coordinate
	ObserverKeyMap map[common.PublicHash]string
	Key            key.Key
}
