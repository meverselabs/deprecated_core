package node

import (
	"github.com/fletaio/common"
	"github.com/fletaio/framework/peer"
	"github.com/fletaio/framework/router"
)

type Config struct {
	ChainCoord *common.Coordinate
	SeedNodes  []string
	Router     router.Config
	Peer       peer.Config
}
