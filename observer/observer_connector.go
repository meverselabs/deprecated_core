package observer

import (
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/data"
)

// Connector provides a sign operation
type Connector interface {
	RequestSign(b *block.Block, s *block.Signed, tran *data.Transactor) (*block.ObserverSigned, error)
	WaitBlock() error
}
