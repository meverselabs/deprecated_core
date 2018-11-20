package chain

import (
	"bytes"
	"runtime"
	"sync"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/consensus"
	"git.fleta.io/fleta/core/data"
	"git.fleta.io/fleta/core/db"
	"git.fleta.io/fleta/core/level"
	"git.fleta.io/fleta/core/reward"
	"git.fleta.io/fleta/core/store"
	"git.fleta.io/fleta/core/transaction"
)

// Chain validates and stores the blockchain data
type Chain struct {
	Config     *Config
	store      *store.Store
	consensus  *consensus.Consensus
	appendLock sync.Mutex
	closeLock  sync.RWMutex
	isClose    bool
}

// NewChain returns a Chain
func NewChain(Config *Config, st *store.Store) (*Chain, error) {
	ObserverSignatureHash := map[common.PublicHash]bool{}
	for _, str := range Config.ObserverSignatures {
		if pubhash, err := common.ParsePublicHash(str); err != nil {
			return nil, err
		} else {
			ObserverSignatureHash[pubhash] = true
		}
	}

	FormulationAccountType, err := st.Accounter().TypeByName("formulation.Account")
	if err != nil {
		return nil, err
	}
	cn := &Chain{
		Config:    Config,
		consensus: consensus.NewConsensus(ObserverSignatureHash, FormulationAccountType),
		store:     st,
	}
	return cn, nil
}

// Init builds the genesis if it is not generated and stored yet.
func (cn *Chain) Init(GenesisContextData *data.ContextData) error {
	if bs := cn.store.CustomData("chaincoord"); bs != nil {
		var coord common.Coordinate
		if _, err := coord.ReadFrom(bytes.NewReader(bs)); err != nil {
			return err
		}
		if !coord.Equal(cn.store.ChainCoord()) {
			return ErrInvalidChainCoordinate
		}
	} else {
		var buffer bytes.Buffer
		if _, err := cn.store.ChainCoord().WriteTo(&buffer); err != nil {
			return err
		}
		if err := cn.store.SetCustomData("chaincoord", buffer.Bytes()); err != nil {
			return err
		}
	}

	var buffer bytes.Buffer
	for _, str := range cn.Config.ObserverSignatures {
		buffer.WriteString(str)
		buffer.WriteString(":")
	}
	GenesisHash := hash.TwoHash(hash.Hash(buffer.Bytes()), GenesisContextData.Hash())
	if h, err := cn.store.BlockHash(0); err != nil {
		if err != db.ErrNotExistKey {
			return err
		}
		CustomHash := map[string][]byte{}
		if SaveData, err := cn.consensus.ApplyGenesis(GenesisContextData); err != nil {
			return err
		} else {
			CustomHash["consensus"] = SaveData
		}
		if err := cn.store.StoreGenesis(GenesisContextData, GenesisHash, CustomHash); err != nil {
			return err
		}
		// TODO : EventInitGenesis
	} else {
		if !h.Equal(GenesisHash) {
			return ErrInvalidGenesisHash
		}
		if SaveData := cn.store.CustomData("consensus"); SaveData == nil {
			return ErrNotExistConsensusSaveData
		} else if err := cn.consensus.LoadFromSaveData(SaveData); err != nil {
			return err
		}
	}
	return nil
}

// Close terminates and cleans the chain
func (cn *Chain) Close() {
	cn.closeLock.Lock()
	defer cn.closeLock.Unlock()

	cn.isClose = true
	cn.store.Close()
}

// IsClose returns the close status of the chain
func (cn *Chain) IsClose() bool {
	cn.closeLock.RLock()
	defer cn.closeLock.RUnlock()

	return cn.isClose
}

// Loader returns the loader of the chain
func (cn *Chain) Loader() data.Loader {
	return cn.store
}

// Provider returns the provider of the chain
func (cn *Chain) Provider() block.Provider {
	return cn.store
}

// IsMinable returns true when the target address is the top rank's address
func (cn *Chain) IsMinable(addr common.Address, TimeoutCount uint32) (bool, error) {
	return cn.consensus.IsMinable(addr, TimeoutCount)
}

// ValidateObserverSigned validates observer signatures
func (cn *Chain) ValidateObserverSigned(b *block.Block, s *block.ObserverSigned) error {
	cn.closeLock.RLock()
	defer cn.closeLock.RUnlock()
	if cn.isClose {
		return ErrClosedChain
	}

	if err := cn.consensus.ValidateBlock(b, s); err != nil {
		return err
	}
	return nil
}

// ValidateContext validates the context using the block
func (cn *Chain) ValidateContext(b *block.Block, ctx *data.Context) error {
	if !b.Header.ChainCoord.Equal(ctx.ChainCoord()) {
		return ErrInvalidChainCoordinate
	}
	if b.Header.Height != ctx.TargetHeight() {
		return ErrExpiredContextHeight
	}
	if !b.Header.HashPrevBlock.Equal(ctx.LastBlockHash()) {
		return ErrExpiredContextBlockHash
	}
	if ctx.TargetHeight() != cn.store.TargetHeight() {
		return ErrInvalidAppendBlockHeight
	}
	if !cn.store.LastBlockHash().Equal(ctx.LastBlockHash()) {
		return ErrInvalidAppendBlockHash
	}
	if ctx.StackSize() > 1 {
		return ErrDirtyContext
	}
	if !b.Header.HashContext.Equal(ctx.Hash()) {
		return ErrInvalidAppendContextHash
	}
	return nil
}

func (cn *Chain) ProcessBlock(b *block.Block, Rewarder reward.Rewarder) (*data.Context, error) {
	cn.closeLock.RLock()
	defer cn.closeLock.RUnlock()
	if cn.isClose {
		return nil, ErrClosedChain
	}

	if !b.Header.ChainCoord.Equal(cn.store.ChainCoord()) {
		return nil, ErrInvalidChainCoordinate
	}
	if b.Header.Height != cn.store.TargetHeight() {
		return nil, ErrInvalidAppendBlockHeight
	}
	if !b.Header.HashPrevBlock.Equal(cn.store.LastBlockHash()) {
		return nil, ErrInvalidAppendBlockHash
	}
	if err := cn.validateBlockBody(b); err != nil {
		return nil, err
	}
	ctx := data.NewContext(cn.store)
	for i, tx := range b.Transactions {
		if _, err := cn.store.Transactor().Execute(ctx, tx, &common.Coordinate{Height: b.Header.Height, Index: uint16(i)}); err != nil {
			return nil, err
		}
	}
	if err := Rewarder.ProcessReward(b.Header.FormulationAddress, ctx); err != nil {
		return nil, err
	}
	if err := cn.ValidateContext(b, ctx); err != nil {
		return nil, err
	}
	return ctx, nil
}

func (cn *Chain) validateBlockBody(b *block.Block) error {
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
	defer close(errs)
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
				if err := cn.store.Transactor().Validate(cn.store, tx, signers); err != nil {
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
	return nil
}

// AppendBlock stores the block data
func (cn *Chain) AppendBlock(b *block.Block, s *block.ObserverSigned, ctx *data.Context) error {
	cn.closeLock.RLock()
	defer cn.closeLock.RUnlock()
	if cn.isClose {
		return ErrClosedChain
	}

	cn.appendLock.Lock()
	defer cn.appendLock.Unlock()

	if err := cn.ValidateContext(b, ctx); err != nil {
		return err
	}
	if err := cn.ValidateObserverSigned(b, s); err != nil {
		return err
	}
	top := ctx.Top()
	CustomHash := map[string][]byte{}
	if SaveData, err := cn.consensus.ApplyBlock(top, b); err != nil {
		return err
	} else {
		CustomHash["consensus"] = SaveData
	}
	if err := cn.store.StoreBlock(top, b, s, CustomHash); err != nil {
		return err
	}
	// TODO : EventBlockConnected
	return nil
}
