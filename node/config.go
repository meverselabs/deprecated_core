package node

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/framework/peer"
	"git.fleta.io/fleta/framework/router"
)

type Config struct {
	ChainCoord *common.Coordinate
	SeedNodes  []string
	Router     router.Config
	Peer       peer.Config
}
