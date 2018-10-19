package observer_proxy

import (
	"git.fleta.io/fleta/core/block"
)

// ObserverProxy TODO
type ObserverProxy interface {
	RequestSign(b *block.Block, s *block.Signed) (*block.ObserverSigned, error)
}
