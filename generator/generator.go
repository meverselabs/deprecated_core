package generator

import (
	"log"
	"time"

	"git.fleta.io/fleta/core/data"
	"git.fleta.io/fleta/core/level"
	"git.fleta.io/fleta/core/transactor"
	"git.fleta.io/fleta/core/txpool"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/transaction"
	"git.fleta.io/fleta/core/wallet/key"
)

// Config TODO
type Config struct {
	Address          common.Address
	BlockVersion     uint16
	GenTimeThreshold time.Duration
	Signer           key.Key
}

// Generator TODO
type Generator struct {
	config *Config
}

// NewGenerator TODO
func NewGenerator(config *Config) *Generator {
	gn := &Generator{
		config: config,
	}
	return gn
}

// Address TODO
func (gn *Generator) Address() common.Address {
	return gn.config.Address.Clone()
}

// GenerateBlock TODO
func (gn *Generator) GenerateBlock(Transactor *transactor.Transactor, TxPool *txpool.TransactionPool, ctx *data.Context, TimeoutCount uint32, ChainCoord *common.Coordinate, PrevHeight uint32, PrevHash hash.Hash256) (*block.Block, *block.Signed, error) {
	b := &block.Block{
		Header: block.Header{
			ChainCoord:         *ChainCoord,
			Height:             PrevHeight + 1,
			Version:            gn.config.BlockVersion,
			HashPrevBlock:      PrevHash,
			Timestamp:          uint64(time.Now().UnixNano()),
			FormulationAddress: gn.Address(),
			TimeoutCount:       TimeoutCount,
		},
		Transactions:          []transaction.Transaction{},
		TransactionSignatures: [][]common.Signature{},
	}

	timer := time.NewTimer(gn.config.GenTimeThreshold)
	TxHashes := make([]hash.Hash256, 0, 65535)

TxLoop:
	for {
		select {
		case <-timer.C:
			break TxLoop
		default:
			item := TxPool.Pop(ctx)
			if item == nil {
				break TxLoop
			}
			idx := uint16(len(b.Transactions))
			if err := Transactor.Execute(ctx, item.Transaction, &common.Coordinate{Height: b.Header.Height, Index: idx}); err != nil {
				log.Println(err)
				//TODO : EventTransactionPendingFail
				break
			}
			b.Transactions = append(b.Transactions, item.Transaction)
			b.TransactionSignatures = append(b.TransactionSignatures, item.Signatures)

			TxHashes = append(TxHashes, item.TxHash)
		}
	}

	if h, err := level.BuildLevelRoot(TxHashes); err != nil {
		return nil, nil, err
	} else {
		b.Header.HashLevelRoot = h
	}

	blockHash := b.Header.Hash()
	if sig, err := gn.config.Signer.Sign(blockHash); err != nil {
		return nil, nil, err
	} else {
		s := &block.Signed{
			BlockHash:          blockHash,
			GeneratorSignature: sig,
		}
		return b, s, nil
	}
}
