package kernel

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/data"
	"git.fleta.io/fleta/core/transaction"
)

// EventHandler provides callback abilities to the kernel
type EventHandler interface {
	OnCreateContext(kn *Kernel, ctx *data.Context) error
	OnProcessBlock(kn *Kernel, b *block.Block, s *block.ObserverSigned, ctx *data.Context) error
	OnPushTransaction(kn *Kernel, tx transaction.Transaction, sigs []common.Signature) error
}

// EventHandlerBase provides empty handler
type EventHandlerBase struct {
}

// OnCreateContext called when a context creation (error prevent using context)
func (eh *EventHandlerBase) OnCreateContext(kn *Kernel, ctx *data.Context) error {
	return nil
}

// OnProcessBlock called when processing block to the chain (error prevent processing block)
func (eh *EventHandlerBase) OnProcessBlock(kn *Kernel, b *block.Block, s *block.ObserverSigned, ctx *data.Context) error {
	return nil
}

// OnPushTransaction called when pushing a transaction to the transaction pool (error prevent push transaction)
func (eh *EventHandlerBase) OnPushTransaction(kn *Kernel, tx transaction.Transaction, sigs []common.Signature) error {
	return nil
}
