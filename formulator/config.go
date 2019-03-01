package formulator

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/key"
	"git.fleta.io/fleta/framework/peer"
	"git.fleta.io/fleta/framework/router"
)

// Config is a configuration of the formulator
type Config struct {
	SeedNodes      []string
	ObserverKeyMap map[common.PublicHash]string
	Key            key.Key
	Formulator     common.Address
	Router         router.Config
	Peer           peer.Config
}
