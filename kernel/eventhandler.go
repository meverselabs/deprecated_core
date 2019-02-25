package kernel

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/data"
	"git.fleta.io/fleta/core/message_def"
	"git.fleta.io/fleta/core/transaction"
)

// EventHandler provides callback abilities to the kernel
type EventHandler interface {
	// OnProcessBlock called when processing a block to the chain (error prevent processing block)
	OnProcessBlock(kn *Kernel, b *block.Block, s *block.ObserverSigned, ctx *data.Context) error
	// AfterProcessBlock called when processed block to the chain
	AfterProcessBlock(kn *Kernel, b *block.Block, s *block.ObserverSigned, ctx *data.Context)
	// OnPushTransaction called when pushing a transaction to the transaction pool (error prevent push transaction)
	OnPushTransaction(kn *Kernel, tx transaction.Transaction, sigs []common.Signature) error
	// AfterPushTransaction called when pushed a transaction to the transaction pool
	AfterPushTransaction(kn *Kernel, tx transaction.Transaction, sigs []common.Signature)
	// DoTransactionBroadcast called when a transaction need to be broadcast
	DoTransactionBroadcast(kn *Kernel, msg *message_def.TransactionMessage)
	// DebugLog TEMP
	DebugLog(kn *Kernel, args ...interface{})
}
