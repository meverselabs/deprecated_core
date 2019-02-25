package kernel

import (
	"bytes"
	"log"
	"runtime"
	"sort"
	"sync"
	"time"

	"git.fleta.io/fleta/core/message_def"

	"git.fleta.io/fleta/core/reward"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/queue"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/consensus"
	"git.fleta.io/fleta/core/data"
	"git.fleta.io/fleta/core/db"
	"git.fleta.io/fleta/core/level"
	"git.fleta.io/fleta/core/transaction"
	"git.fleta.io/fleta/core/txpool"
	"git.fleta.io/fleta/framework/chain"
)

// Kernel processes the block chain using its components and stores state of the block chain
// It based on Proof-of-Formulation and Account/UTXO hybrid model
// All kinds of accounts and transactions processed the out side of kernel
type Kernel struct {
	sync.Mutex
	Config             *Config
	store              *Store
	consensus          *consensus.Consensus
	txPool             *txpool.TransactionPool
	txQueue            *queue.ExpireQueue
	genesisContextData *data.ContextData
	rewarder           reward.Rewarder
	eventHandlers      []EventHandler
	processBlockLock   sync.Mutex
	closeLock          sync.RWMutex
	isClose            bool
}

// NewKernel returns a Kernel
func NewKernel(Config *Config, st *Store, rewarder reward.Rewarder, genesisContextData *data.ContextData) (*Kernel, error) {
	FormulationAccountType, err := st.Accounter().TypeByName("consensus.FormulationAccount")
	if err != nil {
		return nil, err
	}

	kn := &Kernel{
		Config:             Config,
		store:              st,
		genesisContextData: genesisContextData,
		rewarder:           rewarder,
		consensus:          consensus.NewConsensus(Config.ObserverKeyMap, FormulationAccountType),
		txPool:             txpool.NewTransactionPool(),
		txQueue:            queue.NewExpireQueue(),
		eventHandlers:      []EventHandler{},
	}
	kn.txQueue.AddGroup(60 * time.Second)
	kn.txQueue.AddGroup(600 * time.Second)
	kn.txQueue.AddGroup(3600 * time.Second)
	kn.txQueue.AddHandler(kn)

	if bs := kn.store.CustomData("chaincoord"); bs != nil {
		var coord common.Coordinate
		if _, err := coord.ReadFrom(bytes.NewReader(bs)); err != nil {
			return nil, err
		}
		if !coord.Equal(kn.store.ChainCoord()) {
			return nil, ErrInvalidChainCoord
		}
	} else {
		var buffer bytes.Buffer
		if _, err := kn.store.ChainCoord().WriteTo(&buffer); err != nil {
			return nil, err
		}
		if err := kn.store.SetCustomData("chaincoord", buffer.Bytes()); err != nil {
			return nil, err
		}
	}

	var buffer bytes.Buffer
	if _, err := kn.Config.ChainCoord.WriteTo(&buffer); err != nil {
		return nil, err
	}
	keys := []string{}
	for pubhash := range Config.ObserverKeyMap {
		keys = append(keys, pubhash.String())
	}
	sort.Strings(keys)
	for _, v := range keys {
		buffer.WriteString(v)
		buffer.WriteString(":")
	}
	GenesisHash := hash.TwoHash(hash.DoubleHash(buffer.Bytes()), kn.genesisContextData.Hash())
	if h, err := kn.store.Hash(0); err != nil {
		if err != db.ErrNotExistKey {
			return nil, err
		} else {
			CustomData := map[string][]byte{}
			if SaveData, err := kn.consensus.ApplyGenesis(kn.genesisContextData); err != nil {
				return nil, err
			} else {
				CustomData["consensus"] = SaveData
			}
			if err := kn.store.StoreGenesis(GenesisHash, kn.genesisContextData, CustomData); err != nil {
				return nil, err
			}
		}
	} else {
		if !GenesisHash.Equal(h) {
			return nil, chain.ErrInvalidGenesisHash
		}
		if SaveData := kn.store.CustomData("consensus"); SaveData == nil {
			return nil, ErrNotExistConsensusSaveData
		} else if err := kn.consensus.LoadFromSaveData(SaveData); err != nil {
			return nil, err
		}
	}
	kn.genesisContextData = nil // to reduce memory usagse

	log.Println("Kernel", "Loaded with height of", kn.Provider().Height(), kn.Provider().LastHash())

	return kn, nil
}

// Close terminates and cleans the kernel
func (kn *Kernel) Close() {
	kn.closeLock.Lock()
	defer kn.closeLock.Unlock()

	kn.Lock()
	defer kn.Unlock()

	kn.isClose = true
	kn.store.Close()
}

// AddEventHandler adds a event handler to the kernel
func (kn *Kernel) AddEventHandler(eh EventHandler) {
	kn.Lock()
	defer kn.Unlock()

	kn.eventHandlers = append(kn.eventHandlers, eh)
}

// Loader returns the loader of the kernel
func (kn *Kernel) Loader() data.Loader {
	return kn.store
}

// Provider returns the provider of the kernel
func (kn *Kernel) Provider() chain.Provider {
	return kn.store
}

// Version returns the version of the target kernel
func (kn *Kernel) Version() uint16 {
	return kn.store.Version()
}

// ChainCoord returns the coordinate of the target kernel
func (kn *Kernel) ChainCoord() *common.Coordinate {
	return kn.store.ChainCoord()
}

// Accounter returns the accounter of the target kernel
func (kn *Kernel) Accounter() *data.Accounter {
	return kn.store.Accounter()
}

// Transactor returns the transactor of the target kernel
func (kn *Kernel) Transactor() *data.Transactor {
	return kn.store.Transactor()
}

// Block returns the block of the height
func (kn *Kernel) Block(height uint32) (*block.Block, error) {
	cd, err := kn.store.Data(height)
	if err != nil {
		return nil, err
	}
	b := &block.Block{
		Header: cd.Header.(*block.Header),
		Body:   cd.Body.(*block.Body),
	}
	return b, nil
}

// CandidateCount returns a count of the rank table
func (kn *Kernel) CandidateCount() int {
	return kn.consensus.CandidateCount()
}

// TopRank returns the top rank by the given timeout count
func (kn *Kernel) TopRank(TimeoutCount int) (*consensus.Rank, error) {
	return kn.consensus.TopRank(TimeoutCount)
}

// TopRankInMap returns the top rank by the given timeout count in the given map
func (kn *Kernel) TopRankInMap(FormulatorMap map[common.Address]bool) (*consensus.Rank, int, error) {
	return kn.consensus.TopRankInMap(FormulatorMap)
}

// RanksInMap returns the ranks in the map
func (kn *Kernel) RanksInMap(FormulatorMap map[common.Address]bool, Limit int) ([]*consensus.Rank, error) {
	return kn.consensus.RanksInMap(FormulatorMap, Limit)
}

// IsFormulator returns the given information is correct or not
func (kn *Kernel) IsFormulator(Formulator common.Address, Publichash common.PublicHash) bool {
	return kn.consensus.IsFormulator(Formulator, Publichash)
}

// Screening determines the acceptance of the chain data
func (kn *Kernel) Screening(cd *chain.Data) error {
	kn.closeLock.RLock()
	defer kn.closeLock.RUnlock()
	if kn.isClose {
		return ErrKernelClosed
	}

	////log.Println("Kernel", "OnScreening", cd)
	bh := cd.Header.(*block.Header)
	if !bh.ChainCoord.Equal(kn.Config.ChainCoord) {
		return ErrInvalidChainCoord
	}
	if len(cd.Signatures) != len(kn.Config.ObserverKeyMap)/2+2 {
		return ErrInvalidSignatureCount
	}
	s := &block.ObserverSigned{
		Signed: block.Signed{
			HeaderHash:         cd.Header.Hash(),
			GeneratorSignature: cd.Signatures[0],
		},
		ObserverSignatures: cd.Signatures[1:],
	}
	if err := common.ValidateSignaturesMajority(s.Signed.Hash(), s.ObserverSignatures, kn.Config.ObserverKeyMap); err != nil {
		return err
	}
	return nil
}

// CheckFork returns chain.ErrForkDetected if the given data is valid and collapse with stored one
func (kn *Kernel) CheckFork(ch chain.Header, sigs []common.Signature) error {
	kn.closeLock.RLock()
	defer kn.closeLock.RUnlock()
	if kn.isClose {
		return ErrKernelClosed
	}

	if len(sigs) != len(kn.Config.ObserverKeyMap)/2+2 {
		return nil
	}
	s := &block.ObserverSigned{
		Signed: block.Signed{
			HeaderHash:         ch.Hash(),
			GeneratorSignature: sigs[0],
		},
		ObserverSignatures: sigs[1:],
	}
	if err := common.ValidateSignaturesMajority(s.Signed.Hash(), s.ObserverSignatures, kn.Config.ObserverKeyMap); err != nil {
		return nil
	}
	return chain.ErrForkDetected
}

// Validate validates the chain header and returns the context of it
func (kn *Kernel) Validate(b *block.Block, GeneratorSignature common.Signature) (*data.Context, error) {
	kn.closeLock.RLock()
	defer kn.closeLock.RUnlock()
	if kn.isClose {
		return nil, ErrKernelClosed
	}

	kn.Lock()
	defer kn.Unlock()

	////log.Println("Kernel", "Validate", ch, b)
	height := kn.store.Height()
	if b.Header.Height() != height+1 {
		return nil, chain.ErrInvalidHeight
	}

	if b.Header.Height() == 1 {
		if b.Header.Version() <= 0 {
			return nil, chain.ErrInvalidVersion
		}
		if !b.Header.PrevHash().Equal(kn.store.LastHash()) {
			return nil, chain.ErrInvalidPrevHash
		}
	} else {
		LastHeader, err := kn.store.Header(height)
		if err != nil {
			return nil, err
		}
		if b.Header.Version() < LastHeader.Version() {
			return nil, chain.ErrInvalidVersion
		}
		if !b.Header.PrevHash().Equal(LastHeader.Hash()) {
			return nil, chain.ErrInvalidPrevHash
		}
	}

	if !b.Header.ChainCoord.Equal(kn.Config.ChainCoord) {
		return nil, ErrInvalidChainCoord
	}

	Top, err := kn.consensus.TopRank(int(b.Header.TimeoutCount))
	if err != nil {
		return nil, err
	}
	pubkey, err := common.RecoverPubkey(b.Header.Hash(), GeneratorSignature)
	if err != nil {
		return nil, err
	}
	pubhash := common.NewPublicHash(pubkey)
	if !Top.PublicHash.Equal(pubhash) {
		return nil, ErrInvalidTopSignature
	}
	ctx, err := kn.contextByBlock(b)
	if err != nil {
		return nil, err
	}
	return ctx, nil
}

// Process resolves the chain data and updates the context
func (kn *Kernel) Process(cd *chain.Data, UserData interface{}) error {
	kn.closeLock.RLock()
	defer kn.closeLock.RUnlock()
	if kn.isClose {
		return ErrKernelClosed
	}

	kn.Lock()
	defer kn.Unlock()

	////log.Println("Kernel", "Process", cd, UserData)
	b := &block.Block{
		Header: cd.Header.(*block.Header),
		Body:   cd.Body.(*block.Body),
	}
	if !b.Header.ChainCoord.Equal(kn.Config.ChainCoord) {
		return ErrInvalidChainCoord
	}
	if len(cd.Signatures) != len(kn.Config.ObserverKeyMap)/2+2 {
		return ErrInvalidSignatureCount
	}
	s := &block.ObserverSigned{
		Signed: block.Signed{
			HeaderHash:         cd.Header.Hash(),
			GeneratorSignature: cd.Signatures[0],
		},
		ObserverSignatures: cd.Signatures[1:],
	}

	Top, err := kn.consensus.TopRank(int(b.Header.TimeoutCount))
	if err != nil {
		return err
	}
	HeaderHash := b.Header.Hash()
	pubkey, err := common.RecoverPubkey(HeaderHash, s.GeneratorSignature)
	if err != nil {
		return err
	}
	pubhash := common.NewPublicHash(pubkey)
	if !Top.PublicHash.Equal(pubhash) {
		return ErrInvalidTopSignature
	}
	if err := common.ValidateSignaturesMajority(s.Signed.Hash(), s.ObserverSignatures, kn.Config.ObserverKeyMap); err != nil {
		return err
	}
	ctx, is := UserData.(*data.Context)
	if !is {
		v, err := kn.contextByBlock(b)
		if err != nil {
			return err
		}
		ctx = v
	}
	for _, eh := range kn.eventHandlers {
		if err := eh.OnProcessBlock(kn, b, s, ctx); err != nil {
			return err
		}
	}
	top := ctx.Top()
	CustomMap := map[string][]byte{}
	if SaveData, err := kn.consensus.ProcessContext(top, s.HeaderHash, b.Header); err != nil {
		return err
	} else {
		CustomMap["consensus"] = SaveData
	}
	if err := kn.store.StoreData(cd, top, CustomMap); err != nil {
		return err
	}
	for _, eh := range kn.eventHandlers {
		eh.AfterProcessBlock(kn, b, s, ctx)
	}
	for _, tx := range b.Body.Transactions {
		h := tx.Hash()
		kn.txPool.Remove(h, tx)
		kn.txQueue.Remove(string(h[:]))
	}
	log.Println("Block Connected :", kn.store.Height(), HeaderHash.String(), b.Header.Formulator.String(), len(b.Body.Transactions))
	return nil
}

// AddTransaction validate the transaction and push it to the transaction pool
func (kn *Kernel) AddTransaction(tx transaction.Transaction, sigs []common.Signature) error {
	kn.closeLock.RLock()
	defer kn.closeLock.RUnlock()
	if kn.isClose {
		return ErrKernelClosed
	}

	if kn.txQueue.Size() > 65535 {
		return ErrTxQueueOverflowed
	}

	loader := kn.store
	if !tx.ChainCoord().Equal(loader.ChainCoord()) {
		return ErrInvalidChainCoord
	}
	TxHash := tx.Hash()
	if kn.txPool.IsExist(TxHash) {
		return txpool.ErrExistTransaction
	}
	if atx, is := tx.(txpool.AccountTransaction); is {
		seq := loader.Seq(atx.From())
		if atx.Seq() <= seq {
			return ErrPastSeq
		} else if atx.Seq() > seq+100 {
			return ErrTooFarSeq
		}
	}
	signers := make([]common.PublicHash, 0, len(sigs))
	for _, sig := range sigs {
		pubkey, err := common.RecoverPubkey(TxHash, sig)
		if err != nil {
			return err
		}
		signers = append(signers, common.NewPublicHash(pubkey))
	}
	if err := loader.Transactor().Validate(loader, tx, signers); err != nil {
		return err
	}
	for _, eh := range kn.eventHandlers {
		if err := eh.OnPushTransaction(kn, tx, sigs); err != nil {
			return err
		}
	}
	if err := kn.txPool.Push(tx, sigs); err != nil {
		return err
	}
	kn.txQueue.Push(string(TxHash[:]), &message_def.TransactionMessage{
		Tx:   tx,
		Sigs: sigs,
		Tran: kn.Transactor(),
	})
	for _, eh := range kn.eventHandlers {
		eh.AfterPushTransaction(kn, tx, sigs)
	}
	return nil
}

// HasTransaction validate the transaction and push it to the transaction pool
func (kn *Kernel) HasTransaction(TxHash hash.Hash256) bool {
	return kn.txPool.IsExist(TxHash)
}

func (kn *Kernel) contextByBlock(b *block.Block) (*data.Context, error) {
	if err := kn.validateBlockBody(b); err != nil {
		return nil, err
	}

	ctx := data.NewContext(kn.store)
	if !b.Header.ChainCoord.Equal(ctx.ChainCoord()) {
		return nil, ErrInvalidChainCoord
	}
	for i, tx := range b.Body.Transactions {
		if _, err := ctx.Transactor().Execute(ctx, tx, &common.Coordinate{Height: b.Header.Height(), Index: uint16(i)}); err != nil {
			return nil, err
		}
	}
	if err := kn.rewarder.ProcessReward(b.Header.Formulator, ctx); err != nil {
		return nil, err
	}

	if ctx.StackSize() > 1 {
		return nil, ErrDirtyContext
	}

	if !b.Header.ContextHash.Equal(ctx.Hash()) {
		return nil, ErrInvalidAppendContextHash
	}
	return ctx, nil
}

// GenerateBlock generate a next block and its signature using transactions in the pool
func (kn *Kernel) GenerateBlock(TimeoutCount uint32, Formulator common.Address) (*data.Context, *block.Block, error) {
	kn.closeLock.RLock()
	defer kn.closeLock.RUnlock()
	if kn.isClose {
		return nil, nil, ErrKernelClosed
	}

	ctx := data.NewContext(kn.Loader())
	b := &block.Block{
		Header: &block.Header{
			Base: chain.Base{
				Version_:   kn.Provider().Version(),
				Height_:    ctx.TargetHeight(),
				PrevHash_:  ctx.LastHash(),
				Timestamp_: uint64(time.Now().UnixNano()),
			},
			ChainCoord:   *ctx.ChainCoord(),
			Formulator:   Formulator,
			TimeoutCount: TimeoutCount,
		},
		Body: &block.Body{
			Transactions:          []transaction.Transaction{},
			TransactionSignatures: [][]common.Signature{},
		},
	}

	timer := time.NewTimer(200 * time.Millisecond)
	TxHashes := make([]hash.Hash256, 0, 65536)
	TxHashes = append(TxHashes, b.Header.PrevHash())

	kn.txPool.Lock() // Prevent delaying from TxPool.Push
TxLoop:
	for {
		select {
		case <-timer.C:
			break TxLoop
		default:
			sn := ctx.Snapshot()
			item := kn.txPool.UnsafePop(ctx)
			ctx.Revert(sn)
			if item == nil {
				break TxLoop
			}
			idx := uint16(len(b.Body.Transactions))
			if _, err := ctx.Transactor().Execute(ctx, item.Transaction, &common.Coordinate{Height: ctx.TargetHeight(), Index: idx}); err != nil {
				log.Println(err)
				continue
			}

			b.Body.Transactions = append(b.Body.Transactions, item.Transaction)
			b.Body.TransactionSignatures = append(b.Body.TransactionSignatures, item.Signatures)

			TxHashes = append(TxHashes, item.TxHash)

			if len(TxHashes) >= 20000 {
				break TxLoop
			}
		}
	}
	kn.txPool.Unlock() // Prevent delaying from TxPool.Push

	if err := kn.rewarder.ProcessReward(b.Header.Formulator, ctx); err != nil {
		return nil, nil, err
	}
	if ctx.StackSize() > 1 {
		return nil, nil, ErrDirtyContext
	}
	b.Header.ContextHash = ctx.Hash()

	if h, err := level.BuildLevelRoot(TxHashes); err != nil {
		return nil, nil, err
	} else {
		b.Header.LevelRootHash = h
	}
	return ctx, b, nil
}

func (kn *Kernel) validateBlockBody(b *block.Block) error {
	loader := kn.Loader()

	var wg sync.WaitGroup
	cpuCnt := runtime.NumCPU()
	if len(b.Body.Transactions) < 1000 {
		cpuCnt = 1
	}
	txCnt := len(b.Body.Transactions) / cpuCnt
	TxHashes := make([]hash.Hash256, len(b.Body.Transactions)+1)
	TxHashes[0] = b.Header.PrevHash()
	if len(b.Body.Transactions)%cpuCnt != 0 {
		txCnt++
	}
	errs := make(chan error, cpuCnt)
	defer close(errs)
	for i := 0; i < cpuCnt; i++ {
		lastCnt := (i + 1) * txCnt
		if lastCnt > len(b.Body.Transactions) {
			lastCnt = len(b.Body.Transactions)
		}
		wg.Add(1)
		go func(sidx int, txs []transaction.Transaction) {
			defer wg.Done()
			for q, tx := range txs {
				sigs := b.Body.TransactionSignatures[sidx+q]
				TxHash := tx.Hash()
				TxHashes[sidx+q+1] = TxHash

				signers := make([]common.PublicHash, 0, len(sigs))
				for _, sig := range sigs {
					pubkey, err := common.RecoverPubkey(TxHash, sig)
					if err != nil {
						errs <- err
						return
					}
					signers = append(signers, common.NewPublicHash(pubkey))
				}
				if err := kn.store.Transactor().Validate(loader, tx, signers); err != nil {
					errs <- err
					return
				}
			}
		}(i*txCnt, b.Body.Transactions[i*txCnt:lastCnt])
	}
	wg.Wait()
	if len(errs) > 0 {
		err := <-errs
		return err
	}
	if h, err := level.BuildLevelRoot(TxHashes); err != nil {
		return err
	} else if !b.Header.LevelRootHash.Equal(h) {
		return ErrInvalidLevelRootHash
	}
	return nil
}

// OnItemExpired is called when the item is expired
func (kn *Kernel) OnItemExpired(Interval time.Duration, Key string, Item interface{}) {
	for _, eh := range kn.eventHandlers {
		eh.DoTransactionBroadcast(kn, Item.(*message_def.TransactionMessage))
	}
}

// DebugLog TEMP
func (kn *Kernel) DebugLog(args ...interface{}) {
	log.Println(args...)
	for _, eh := range kn.eventHandlers {
		eh.DebugLog(kn, args...)
	}
}
