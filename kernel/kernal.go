package kernel

import (
	"bytes"
	"errors"
	"runtime"
	"sync"

	"git.fleta.io/fleta/common/hash"

	"git.fleta.io/fleta/core/observer_proxy"
	"git.fleta.io/fleta/core/txpool"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/amount"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/consensus"
	"git.fleta.io/fleta/core/data"
	"git.fleta.io/fleta/core/db"
	"git.fleta.io/fleta/core/generator"
	"git.fleta.io/fleta/core/level"
	"git.fleta.io/fleta/core/store"
	"git.fleta.io/fleta/core/transaction"
)

// kernel errors
var (
	ErrInvalidChainCoordinate    = errors.New("invalid chain coordinate")
	ErrInvalidHashLevelRoot      = errors.New("invalid hash level root")
	ErrInvalidGenesisHash        = errors.New("invalid genesis hash")
	ErrNotExistConsensusSaveData = errors.New("invalid consensus save data")
	ErrDirtyContext              = errors.New("dirty context")
	ErrInvalidGenerateRequest    = errors.New("invalid generate request")
)

// Kernel TODO
type Kernel struct {
	ChainCoord    *common.Coordinate
	Consensus     *consensus.Consensus
	Transactor    *data.Transactor
	Store         *store.Store
	TxPool        *txpool.TransactionPool
	Generator     *generator.Generator
	ObserverProxy observer_proxy.ObserverProxy
}

// Init TODO
func (kn *Kernel) Init(ObserverSignatures []string, GenesisContextData *data.ContextData) error {
	if bs := kn.Store.CustomData("chaincoord"); bs != nil {
		var coord common.Coordinate
		if _, err := coord.ReadFrom(bytes.NewReader(bs)); err != nil {
			return err
		}
		if !coord.Equal(kn.ChainCoord) {
			return ErrInvalidChainCoordinate
		}
	} else {
		var buffer bytes.Buffer
		if _, err := kn.ChainCoord.WriteTo(&buffer); err != nil {
			return err
		}
		if err := kn.Store.SetCustomData("chaincoord", buffer.Bytes()); err != nil {
			return err
		}
	}

	var buffer bytes.Buffer
	for _, str := range ObserverSignatures {
		buffer.WriteString(str)
		buffer.WriteString(":")
	}
	GenesisHash := hash.TwoHash(hash.Hash(buffer.Bytes()), GenesisContextData.Hash())
	if h, err := kn.Store.BlockHash(0); err != nil {
		if err != db.ErrNotExistKey {
			return err
		}
		CustomHash := map[string][]byte{}
		if SaveData, err := kn.Consensus.ApplyGenesis(GenesisContextData); err != nil {
			return err
		} else {
			CustomHash["consensus"] = SaveData
		}
		if err := kn.Store.StoreGenesis(GenesisContextData, GenesisHash, CustomHash); err != nil {
			return err
		}
		// TODO : EventInitGenesis
	} else {
		if !h.Equal(GenesisHash) {
			return ErrInvalidGenesisHash
		}
		if SaveData := kn.Store.CustomData("consensus"); SaveData == nil {
			return ErrNotExistConsensusSaveData
		} else if err := kn.Consensus.LoadFromSaveData(SaveData); err != nil {
			return err
		}
	}
	return nil
}

// Reward TODO
func (kn *Kernel) Reward(Height uint32) *amount.Amount {
	// TEMP
	return amount.COIN.Clone()
}

// RecvBlock TODO
func (kn *Kernel) RecvBlock(b *block.Block, s *block.ObserverSigned, IsNewBlock bool) error {
	if !b.Header.ChainCoord.Equal(kn.ChainCoord) {
		return ErrInvalidChainCoordinate
	}
	PrevHeight := kn.Store.Height()
	PrevHash, err := kn.Store.BlockHash(PrevHeight)
	if err != nil {
		return err
	}
	if err := kn.Consensus.ValidateBlock(b, s, PrevHeight, PrevHash); err != nil {
		return err
	}
	ctx := data.NewContext(kn.Store)
	if err := kn.processBlock(ctx, b); err != nil {
		return err
	}
	if ctx.StackSize() > 1 {
		return ErrDirtyContext
	}
	top := ctx.Top()
	if acc, err := top.Account(b.Header.FormulationAddress); err != nil {
		return err
	} else {
		acc.SetBalance(kn.ChainCoord, acc.Balance(kn.ChainCoord).Add(kn.Reward(b.Header.Height)))
	}
	CustomHash := map[string][]byte{}
	if SaveData, err := kn.Consensus.ApplyBlock(top, b); err != nil {
		return err
	} else {
		CustomHash["consensus"] = SaveData
	}
	if err := kn.Store.StoreBlock(top, b, s, CustomHash); err != nil {
		return err
	}
	// TODO : EventBlockConnected

	for _, tx := range b.Transactions {
		kn.TxPool.Remove(tx)
	}

	if kn.Generator != nil {
		if IsNewBlock {
			if is, err := kn.Consensus.IsMinable(kn.Generator.Address(), 0); err != nil {
				return err
			} else if is {
				if err := kn.GenerateBlock(0); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (kn *Kernel) processBlock(ctx *data.Context, b *block.Block) error {
	var wg sync.WaitGroup
	cpuCnt := runtime.NumCPU()
	if len(b.Transactions) < 1000 {
		cpuCnt = 1
	}
	txCnt := len(b.Transactions) / cpuCnt
	TxHashes := make([]hash.Hash256, len(b.Transactions))
	if len(b.Transactions)%cpuCnt != 0 {
		txCnt++
	}
	errs := make(chan error, cpuCnt)
	for i := 0; i < cpuCnt; i++ {
		lastCnt := (i + 1) * txCnt
		if lastCnt > len(b.Transactions) {
			lastCnt = len(b.Transactions)
		}
		wg.Add(1)
		go func(sidx int, txs []transaction.Transaction) {
			defer wg.Done()
			for q, tx := range txs {
				sigs := b.TransactionSignatures[sidx+q]
				TxHash := tx.Hash()
				TxHashes[sidx+q] = TxHash

				signers := make([]common.PublicHash, 0, len(sigs))
				for _, sig := range sigs {
					pubkey, err := common.RecoverPubkey(TxHash, sig)
					if err != nil {
						errs <- err
						return
					}
					signers = append(signers, common.NewPublicHash(pubkey))
				}
				if err := kn.Transactor.Validate(kn.Store, tx, signers); err != nil {
					errs <- err
					return
				}
			}
		}(i*txCnt, b.Transactions[i*txCnt:lastCnt])
	}
	wg.Wait()
	if len(errs) > 0 {
		err := <-errs
		return err
	}
	if h, err := level.BuildLevelRoot(TxHashes); err != nil {
		return err
	} else if !b.Header.HashLevelRoot.Equal(h) {
		return ErrInvalidHashLevelRoot
	}
	for i, tx := range b.Transactions {
		if _, err := kn.Transactor.Execute(ctx, tx, &common.Coordinate{Height: b.Header.Height, Index: uint16(i)}); err != nil {
			return err
		}
	}
	return nil
}

// GenerateBlock TODO
func (kn *Kernel) GenerateBlock(TimeoutCount uint32) error {
	ctx := data.NewContext(kn.Store)
	PrevHeight := kn.Store.Height()
	PrevHash, err := kn.Store.BlockHash(PrevHeight)
	if err != nil {
		return err
	}
	if is, err := kn.Consensus.IsMinable(kn.Generator.Address(), TimeoutCount); err != nil {
		return err
	} else if !is {
		return ErrInvalidGenerateRequest
	}
	nb, ns, err := kn.Generator.GenerateBlock(kn.Transactor, kn.TxPool, ctx, TimeoutCount, kn.ChainCoord, PrevHeight, PrevHash)
	if err != nil {
		return err
	}
	// TODO : EventBlockGenerated
	nos, err := kn.ObserverProxy.RequestSign(nb, ns)
	if err != nil {
		return err
	}
	// TODO : EventBlockObserverSigned

	CustomHash := map[string][]byte{}
	if ctx.StackSize() > 1 {
		return ErrDirtyContext
	}
	top := ctx.Top()
	if SaveData, err := kn.Consensus.ApplyBlock(top, nb); err != nil {
		return err
	} else {
		CustomHash["consensus"] = SaveData
	}
	if err := kn.Store.StoreBlock(top, nb, nos, CustomHash); err != nil {
		return err
	}
	// TODO : save consensus
	// TODO : EventBlockConnected
	return nil
}

// RecvTransaction TODO
func (kn *Kernel) RecvTransaction(tx transaction.Transaction, sigs []common.Signature) error {
	if !tx.ChainCoord().Equal(kn.ChainCoord) {
		return ErrInvalidChainCoordinate
	}
	//TODO : check IsExist in pool
	signers := make([]common.PublicHash, 0, len(sigs))
	TxHash := tx.Hash()
	for _, sig := range sigs {
		pubkey, err := common.RecoverPubkey(TxHash, sig)
		if err != nil {
			return err
		}
		signers = append(signers, common.NewPublicHash(pubkey))
	}
	if err := kn.Transactor.Validate(kn.Store, tx, signers); err != nil {
		return err
	}
	if err := kn.TxPool.Push(tx, sigs); err != nil {
		return err
	}
	// TODO : EventTransactionAdded
	return nil
}
