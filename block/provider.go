package block

import (
	"git.fleta.io/fleta/common/hash"
)

// Provider TODO
type Provider interface {
	BlockHash(height uint32) (hash.Hash256, error)
	Block(height uint32) (*Block, error)
	ObserverSigned(height uint32) (*ObserverSigned, error)
	Height() uint32
}
