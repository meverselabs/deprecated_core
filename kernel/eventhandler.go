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
	BeforeAppendBlock(b *block.Block, s *block.ObserverSigned, ctx *data.Context) error
	AfterAppendBlock(b *block.Block, s *block.ObserverSigned, ctx *data.Context)
	BeforePushTransaction(tx transaction.Transaction, sigs []common.Signature) error
	AfterPushTransaction(tx transaction.Transaction, sigs []common.Signature)
}

// EventHandlerBase provides empty handler
type EventHandlerBase struct {
}

// OnCreateContext called when a context creation (error prevent using context)
func (eh *EventHandlerBase) OnCreateContext(ctx *data.Context) error {
	return nil
}

// BeforeAppendBlock called before append block to the chain (error prevent append block)
func (eh *EventHandlerBase) BeforeAppendBlock(b *block.Block, s *block.ObserverSigned, ctx *data.Context) error {
	return nil
}

// AfterAppendBlock called after append block to the chain
func (eh *EventHandlerBase) AfterAppendBlock(b *block.Block, s *block.ObserverSigned, ctx *data.Context) error {
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
