package kernel

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/data"
	"git.fleta.io/fleta/core/transaction"
)

// EventHandler provides callback abilities to the kernel
type EventHandler interface {
	OnCreateContext(ctx *data.Context) error
	BeforeProcessBlock(b *block.Block, s *block.ObserverSigned, ctx *data.Context) error
	AfterProcessBlock(b *block.Block, s *block.ObserverSigned, ctx *data.Context) error
	BeforePushTransaction(tx transaction.Transaction, sigs []common.Signature) error
	AfterPushTransaction(tx transaction.Transaction, sigs []common.Signature) error
}

// EventHandlerBase provides empty handler
type EventHandlerBase struct {
}

// OnCreateContext called when a context creation (error prevent using context)
func (eh *EventHandlerBase) OnCreateContext(ctx *data.Context) error {
	return nil
}

// BeforeProcessBlock called before processingblock to the chain (error prevent processing block)
func (eh *EventHandlerBase) BeforeProcessBlock(b *block.Block, s *block.ObserverSigned, ctx *data.Context) error {
	return nil
}

// AfterProcessBlock called after processing block to the chain
func (eh *EventHandlerBase) AfterProcessBlock(b *block.Block, s *block.ObserverSigned, ctx *data.Context) error {
	return nil
}

// BeforePushTransaction called before push a transaction to the transaction pool (error prevent push transaction)
func (eh *EventHandlerBase) BeforePushTransaction(tx transaction.Transaction, sigs []common.Signature) error {
	return nil
}

// AfterPushTransaction called after push a transaction to the transaction pool
func (eh *EventHandlerBase) AfterPushTransaction(tx transaction.Transaction, sigs []common.Signature) error {
	return nil
}
