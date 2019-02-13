package kernel

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/data"
	"git.fleta.io/fleta/core/transaction"
)

// EventHandler provides callback abilities to the kernel
type EventHandler interface {
	// OnCreateContext called when a context creation (error prevent using context)
	OnCreateContext(kn *Kernel, ctx *data.Context) error
	// OnProcessBlock called when processing a block to the chain (error prevent processing block)
	OnProcessBlock(kn *Kernel, b *block.Block, s *block.ObserverSigned, ctx *data.Context) error
	// OnPushTransaction called when pushing a transaction to the transaction pool (error prevent push transaction)
	OnPushTransaction(kn *Kernel, tx transaction.Transaction, sigs []common.Signature) error
	// AfterProcessBlock called when processed block to the chain
	AfterProcessBlock(kn *Kernel, b *block.Block, s *block.ObserverSigned)
}
