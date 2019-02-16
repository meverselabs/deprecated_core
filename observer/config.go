package observer

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/key"
)

// Config TODO
type Config struct {
	ChainCoord    *common.Coordinate
	ObserverKeys  []string
	NetAddressMap map[common.PublicHash]string
	Key           key.Key
}
