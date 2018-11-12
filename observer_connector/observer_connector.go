package observer_connector

import (
	"git.fleta.io/fleta/core/block"
)

// ObserverConnector provides a sign operation
type ObserverConnector interface {
	RequestSign(b *block.Block, s *block.Signed) (*block.ObserverSigned, error)
}
