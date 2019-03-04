package formulator

import (
	"github.com/fletaio/common"
	"github.com/fletaio/core/key"
	"github.com/fletaio/framework/peer"
	"github.com/fletaio/framework/router"
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
