package generator

import (
	"log"
	"time"

	"git.fleta.io/fleta/core/data"
	"git.fleta.io/fleta/core/key"
	"git.fleta.io/fleta/core/level"
	"git.fleta.io/fleta/core/txpool"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/transaction"
)

// Config is a generator's config
type Config struct {
	Address          string
	BlockVersion     uint16
	GenTimeThreshold time.Duration
}

// Generator makes block using the config and chain informations
type Generator struct {
	config  *Config
	address common.Address
	signer  key.Key
}

// NewGenerator returns a Generator
func NewGenerator(config *Config, Signer key.Key) (*Generator, error) {
	addr, err := common.ParseAddress(config.Address)
	if err != nil {
		return nil, err
	}
	gn := &Generator{
		config:  config,
		address: addr,
		signer:  Signer,
	}
	return gn, nil
}

// Address returns the address of the formulator
func (gn *Generator) Address() common.Address {
	return gn.address.Clone()
}

// GenerateBlock generate a next block and its signature using transactions in the pool
func (gn *Generator) GenerateBlock(Transactor *data.Transactor, TxPool *txpool.TransactionPool, ctx *data.Context, TimeoutCount uint32, ChainCoord *common.Coordinate, PrevHeight uint32, PrevHash hash.Hash256) (*block.Block, *block.Signed, error) {
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

	TxPool.Lock() // Prevent delaying from TxPool.Push
TxLoop:
	for {
		select {
		case <-timer.C:
			break TxLoop
		default:
			item := TxPool.UnsafePop(ctx)
			if item == nil {
				break TxLoop
			}
			idx := uint16(len(b.Transactions))
			if _, err := Transactor.Execute(ctx, item.Transaction, &common.Coordinate{Height: b.Header.Height, Index: idx}); err != nil {
				log.Println(err)
				//TODO : EventTransactionPendingFail
				break
			}
			b.Transactions = append(b.Transactions, item.Transaction)
			b.TransactionSignatures = append(b.TransactionSignatures, item.Signatures)

			TxHashes = append(TxHashes, item.TxHash)
		}
	}
	TxPool.Unlock() // Prevent delaying from TxPool.Push

	if h, err := level.BuildLevelRoot(TxHashes); err != nil {
		return nil, nil, err
	} else {
		b.Header.HashLevelRoot = h
	}

	blockHash := b.Header.Hash()
	if sig, err := gn.signer.Sign(blockHash); err != nil {
		return nil, nil, err
	} else {
		s := &block.Signed{
			BlockHash:          blockHash,
			GeneratorSignature: sig,
		}
		return b, s, nil
	}
}
