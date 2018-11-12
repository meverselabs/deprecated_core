package kernel

import (
	"bytes"
	"runtime"
	"sync"

	"git.fleta.io/fleta/common/hash"

	"git.fleta.io/fleta/core/observer_connector"
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

// Kernel processes the block chain using its components and stores state of the block chain
// It based on Proof-of-Formulation and Account/UTXO hybrid model
// All kinds of accounts and transactions processed the out side of kernel
type Kernel struct {
	ChainCoord        *common.Coordinate
	Consensus         *consensus.Consensus
	Store             *store.Store
	TxPool            *txpool.TransactionPool
	Generator         *generator.Generator
	ObserverConnector observer_connector.ObserverConnector
	closeLock         sync.RWMutex
	isClose           bool
}

// Init builds the genesis if it is not generated and stored yet.
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

// Close terminate and clean kernel
func (kn *Kernel) Close() {
	kn.closeLock.Lock()
	defer kn.closeLock.Unlock()

	kn.Store.Close()
	kn.isClose = true
}

// IsClose returns the close status of kernel
func (kn *Kernel) IsClose() bool {
	kn.closeLock.RLock()
	defer kn.closeLock.RUnlock()

	return kn.isClose
}

// Reward returns the mining reward
func (kn *Kernel) Reward(Height uint32) *amount.Amount {
	// TEMP
	return amount.COIN.Clone()
}

// RecvBlock proccesses and connect the new block
func (kn *Kernel) RecvBlock(b *block.Block, s *block.ObserverSigned) error {
	kn.closeLock.RLock()
	defer kn.closeLock.RUnlock()
	if kn.isClose {
		return ErrClosed
	}

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
				if err := kn.Store.Transactor().Validate(kn.Store, tx, signers); err != nil {
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
		if _, err := kn.Store.Transactor().Execute(ctx, tx, &common.Coordinate{Height: b.Header.Height, Index: uint16(i)}); err != nil {
			return err
		}
	}
	return nil
}

// GenerateBlock generates the next block (*only for a formulator)
func (kn *Kernel) GenerateBlock(TimeoutCount uint32) (*block.Block, *block.ObserverSigned, error) {
	kn.closeLock.RLock()
	defer kn.closeLock.RUnlock()
	if kn.isClose {
		return nil, nil, ErrClosed
	}

	if kn.Generator == nil {
		return nil, nil, ErrNotFormulator
	}

	ctx := data.NewContext(kn.Store)
	PrevHeight := kn.Store.Height()
	PrevHash, err := kn.Store.BlockHash(PrevHeight)
	if err != nil {
		return nil, nil, err
	}
	if is, err := kn.Consensus.IsMinable(kn.Generator.Address(), TimeoutCount); err != nil {
		return nil, nil, err
	} else if !is {
		return nil, nil, ErrInvalidGenerateRequest
	}
	nb, ns, err := kn.Generator.GenerateBlock(kn.Store.Transactor(), kn.TxPool, ctx, TimeoutCount, kn.ChainCoord, PrevHeight, PrevHash)
	if err != nil {
		return nil, nil, err
	}
	// TODO : EventBlockGenerated
	nos, err := kn.ObserverConnector.RequestSign(nb, ns)
	if err != nil {
		return nil, nil, err
	}
	// TODO : EventBlockObserverSigned

	CustomHash := map[string][]byte{}
	if ctx.StackSize() > 1 {
		return nil, nil, ErrDirtyContext
	}
	top := ctx.Top()
	if SaveData, err := kn.Consensus.ApplyBlock(top, nb); err != nil {
		return nil, nil, err
	} else {
		CustomHash["consensus"] = SaveData
	}
	if err := kn.Store.StoreBlock(top, nb, nos, CustomHash); err != nil {
		return nil, nil, err
	}
	// TODO : save consensus
	// TODO : EventBlockConnected
	return nb, nos, nil
}

// RecvTransaction validate the transaction and push it to the pool
func (kn *Kernel) RecvTransaction(tx transaction.Transaction, sigs []common.Signature) error {
	kn.closeLock.RLock()
	defer kn.closeLock.RUnlock()
	if kn.isClose {
		return ErrClosed
	}

	if !tx.ChainCoord().Equal(kn.ChainCoord) {
		return ErrInvalidChainCoordinate
	}
	TxHash := tx.Hash()
	if kn.TxPool.IsExist(TxHash) {
		return txpool.ErrExistTransaction
	}
	signers := make([]common.PublicHash, 0, len(sigs))
	for _, sig := range sigs {
		pubkey, err := common.RecoverPubkey(TxHash, sig)
		if err != nil {
			return err
		}
		signers = append(signers, common.NewPublicHash(pubkey))
	}
	if err := kn.Store.Transactor().Validate(kn.Store, tx, signers); err != nil {
		return err
	}
	if err := kn.TxPool.Push(tx, sigs); err != nil {
		return err
	}
	// TODO : EventTransactionAdded
	return nil
}
