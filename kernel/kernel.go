package kernel

import (
	"bytes"
	"io"
	"log"
	"runtime"
	"sync"
	"time"

	"git.fleta.io/fleta/core/message_def"
	"git.fleta.io/fleta/core/observer"
	"git.fleta.io/fleta/core/reward"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/consensus"
	"git.fleta.io/fleta/core/data"
	"git.fleta.io/fleta/core/db"
	"git.fleta.io/fleta/core/level"
	"git.fleta.io/fleta/core/transaction"
	"git.fleta.io/fleta/core/txpool"
	"git.fleta.io/fleta/framework/chain"
	"git.fleta.io/fleta/framework/chain/mesh"
	"git.fleta.io/fleta/framework/message"
)

// Kernel processes the block chain using its components and stores state of the block chain
// It based on Proof-of-Formulation and Account/UTXO hybrid model
// All kinds of accounts and transactions processed the out side of kernel
type Kernel struct {
	Config             *Config
	store              *Store
	consensus          *consensus.Consensus
	txPool             *txpool.TransactionPool
	genesisContextData *data.ContextData
	rewarder           reward.Rewarder

	manager          *message.Manager
	processBlockLock sync.Mutex
	closeLock        sync.RWMutex
	eventHandlers    []EventHandler
	isClose          bool
	// formualtor
	observerConnector *observer.Connector
	genBlockLock      sync.Mutex
}

// NewKernel returns a Kernel
func NewKernel(Config *Config, st *Store, rewarder reward.Rewarder, genesisContextData *data.ContextData) (*Kernel, error) {
	ObserverKeyHash := map[common.PublicHash]bool{}
	for _, str := range Config.ObserverKeys {
		if pubhash, err := common.ParsePublicHash(str); err != nil {
			return nil, err
		} else {
			ObserverKeyHash[pubhash] = true
		}
	}

	FormulationAccountType, err := st.Accounter().TypeByName("consensus.FormulationAccount")
	if err != nil {
		return nil, err
	}

	kn := &Kernel{
		Config:             Config,
		store:              st,
		genesisContextData: genesisContextData,
		rewarder:           rewarder,
		consensus:          consensus.NewConsensus(ObserverKeyHash, FormulationAccountType),
		txPool:             txpool.NewTransactionPool(),
		manager:            message.NewManager(),
		eventHandlers:      []EventHandler{},
	}
	kn.manager.SetCreator(message_def.TransactionMessageType, kn.messageCreator)
	return kn, nil
}

// OnClose terminates and cleans the kernel
func (kn *Kernel) Close() {
	kn.closeLock.Lock()
	defer kn.closeLock.Unlock()

	kn.isClose = true
	kn.store.Close()
}

// Provider returns the provider of the chain
func (kn *Kernel) Provider() chain.Provider {
	return kn.store
}

// Version returns the version of the target chain
func (kn *Kernel) Version() uint16 {
	return kn.store.Version()
}

// ChainCoord returns the coordinate of the target chain
func (kn *Kernel) ChainCoord() *common.Coordinate {
	return kn.store.ChainCoord()
}

// Accounter returns the accounter of the target chain
func (kn *Kernel) Accounter() *data.Accounter {
	return kn.store.Accounter()
}

// Transactor returns the transactor of the target chain
func (kn *Kernel) Transactor() *data.Transactor {
	return kn.store.Transactor()
}

// Block returns the block of the height
func (kn *Kernel) Block(height uint32) (*block.Block, error) {
	cd, err := kn.store.Data(height)
	if err != nil {
		return nil, err
	}
	b := &block.Block{}
	if _, err := b.Header.ReadFrom(bytes.NewReader(cd.Body)); err != nil {
		return nil, err
	}
	if _, err := b.Body.ReadFromWith(bytes.NewReader(cd.Extra), kn.store.Transactor()); err != nil {
		return nil, err
	}
	return b, nil
}

// Init is called when the chain is going to init
func (kn *Kernel) Init() error {
	kn.closeLock.RLock()
	defer kn.closeLock.RUnlock()
	if kn.isClose {
		return ErrKernelClosed
	}

	log.Println("Kernel", "Init")

	if bs := kn.store.CustomData("chaincoord"); bs != nil {
		var coord common.Coordinate
		if _, err := coord.ReadFrom(bytes.NewReader(bs)); err != nil {
			return err
		}
		if !coord.Equal(kn.store.ChainCoord()) {
			return ErrInvalidChainCoord
		}
	} else {
		var buffer bytes.Buffer
		if _, err := kn.store.ChainCoord().WriteTo(&buffer); err != nil {
			return err
		}
		if err := kn.store.SetCustomData("chaincoord", buffer.Bytes()); err != nil {
			return err
		}
	}

	var buffer bytes.Buffer
	if _, err := kn.Config.ChainCoord.WriteTo(&buffer); err != nil {
		return err
	}
	for _, str := range kn.Config.ObserverKeys {
		buffer.WriteString(str)
		buffer.WriteString(":")
	}
	GenesisHash := hash.TwoHash(hash.DoubleHash(buffer.Bytes()), kn.genesisContextData.Hash())
	if h, err := kn.store.Hash(0); err != nil {
		if err != db.ErrNotExistKey {
			return err
		} else {
			CustomData := map[string][]byte{}
			if SaveData, err := kn.consensus.ApplyGenesis(kn.genesisContextData); err != nil {
				return err
			} else {
				CustomData["consensus"] = SaveData
			}
			if err := kn.store.StoreGenesis(GenesisHash, kn.genesisContextData, CustomData); err != nil {
				return err
			}
		}
	} else {
		if !GenesisHash.Equal(h) {
			return chain.ErrInvalidGenesisHash
		}
		if SaveData := kn.store.CustomData("consensus"); SaveData == nil {
			return ErrNotExistConsensusSaveData
		} else if err := kn.consensus.LoadFromSaveData(SaveData); err != nil {
			return err
		}
	}
	kn.genesisContextData = nil // to reduce memory usagse
	return nil
}

// Screening determines the acceptance of the chain data
func (kn *Kernel) Screening(cd *chain.Data) error {
	kn.closeLock.RLock()
	defer kn.closeLock.RUnlock()
	if kn.isClose {
		return ErrKernelClosed
	}

	log.Println("Kernel", "OnScreening", cd)
	var bh block.Header
	if _, err := bh.ReadFrom(bytes.NewReader(cd.Body)); err != nil {
		return err
	}
	if !bh.ChainCoord.Equal(kn.Config.ChainCoord) {
		return ErrInvalidChainCoord
	}
	if len(cd.Signatures) != len(kn.Config.ObserverKeys)/2+2 {
		return ErrInvalidSignatureCount
	}
	s := &block.ObserverSigned{
		Signed: block.Signed{
			HeaderHash:         cd.Header.Hash(),
			GeneratorSignature: cd.Signatures[0],
		},
		ObserverSignatures: cd.Signatures[1:],
	}
	if err := kn.consensus.ValidateObserverSignatures(s.Signed.Hash(), s.ObserverSignatures); err != nil {
		return err
	}
	return nil
}

// CheckFork returns chain.ErrForkDetected if the given data is valid and collapse with stored one
func (kn *Kernel) CheckFork(ch *chain.Header, sigs []common.Signature) error {
	kn.closeLock.RLock()
	defer kn.closeLock.RUnlock()
	if kn.isClose {
		return ErrKernelClosed
	}

	if len(sigs) != len(kn.Config.ObserverKeys)/2+2 {
		return nil
	}
	s := &block.ObserverSigned{
		Signed: block.Signed{
			HeaderHash:         ch.Hash(),
			GeneratorSignature: sigs[0],
		},
		ObserverSignatures: sigs[1:],
	}
	if err := kn.consensus.ValidateObserverSignatures(s.Signed.Hash(), s.ObserverSignatures); err != nil {
		return nil
	}
	return chain.ErrForkDetected
}

// Process resolves the chain data and updates the context
func (kn *Kernel) Process(cd *chain.Data, UserData interface{}) error {
	kn.closeLock.RLock()
	defer kn.closeLock.RUnlock()
	if kn.isClose {
		return ErrKernelClosed
	}

	log.Println("Kernel", "OnProcess", cd, UserData)
	b := &block.Block{}
	if _, err := b.Header.ReadFrom(bytes.NewReader(cd.Body)); err != nil {
		return err
	}
	if !b.Header.ChainCoord.Equal(kn.Config.ChainCoord) {
		return ErrInvalidChainCoord
	}
	if len(cd.Signatures) != len(kn.Config.ObserverKeys)/2+2 {
		return ErrInvalidSignatureCount
	}
	s := &block.ObserverSigned{
		Signed: block.Signed{
			HeaderHash:         cd.Header.Hash(),
			GeneratorSignature: cd.Signatures[0],
		},
		ObserverSignatures: cd.Signatures[1:],
	}
	if err := kn.consensus.ValidateBlockHeader(&b.Header, s); err != nil {
		return err
	}
	if _, err := b.Body.ReadFromWith(bytes.NewReader(cd.Extra), kn.store.Transactor()); err != nil {
		return err
	}
	ctx, is := UserData.(*data.Context)
	if !is {
		v, err := kn.ContextByBlock(b)
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
	CustomHash := map[string][]byte{}
	if SaveData, err := kn.consensus.ProcessContext(top, s.HeaderHash, &b.Header); err != nil {
		return err
	} else {
		CustomHash["consensus"] = SaveData
	}
	if err := kn.store.StoreData(cd, top, CustomHash); err != nil {
		return err
	}
	return nil
}

// IsGenerator returns the kernel is generator or not
func (kn *Kernel) IsGenerator() bool {
	return kn.Config.FormulatorKey != nil
}

// Generate generate the next block when this formulator is the top rank
func (kn *Kernel) Generate() (*chain.Data, interface{}, error) {
	kn.closeLock.RLock()
	defer kn.closeLock.RUnlock()
	if kn.isClose {
		return nil, nil, ErrKernelClosed
	}

	if kn.Config.FormulatorKey == nil {
		return nil, nil, ErrNotFormulator
	}

	kn.genBlockLock.Lock()
	defer kn.genBlockLock.Unlock()

	if is, err := kn.consensus.IsMinable(kn.Config.FormulatorAddress, 0); err != nil {
		return nil, nil, err
	} else if !is {
		return nil, nil, nil
	}

	ctx := data.NewContext(kn.store)
	for _, eh := range kn.eventHandlers {
		if err := eh.OnCreateContext(kn, ctx); err != nil {
			return nil, nil, err
		}
	}
	cd, ns, err := kn.GenerateBlock(ctx, 0)
	if err != nil {
		return nil, nil, err
	}
	nos, err := kn.observerConnector.RequestSign(cd, ns)
	if err != nil {
		return nil, nil, err
	}
	cd.Signatures = append(cd.Signatures, nos.ObserverSignatures...)
	return cd, ctx, nil
}

// OnRecv is called when a message is received from the peer
func (kn *Kernel) OnRecv(p mesh.Peer, t message.Type, r io.Reader) error {
	m, err := kn.manager.ParseMessage(r, t)
	if err != nil {
		return err
	}
	switch msg := m.(type) {
	case *message_def.TransactionMessage:
		if err := kn.AddTransaction(msg.Tx, msg.Sigs); err != nil {
			return err
		}
		return nil
	default:
		return message.ErrUnhandledMessage
	}
}

// AddTransaction validate the transaction and push it to the transaction pool
func (kn *Kernel) AddTransaction(tx transaction.Transaction, sigs []common.Signature) error {
	kn.closeLock.RLock()
	defer kn.closeLock.RUnlock()
	if kn.isClose {
		return ErrKernelClosed
	}

	loader := kn.store
	if !tx.ChainCoord().Equal(loader.ChainCoord()) {
		return ErrInvalidChainCoord
	}
	TxHash := tx.Hash()
	if kn.txPool.IsExist(TxHash) {
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
	return nil
}

// ContextByBlock creates context using the target block
func (kn *Kernel) ContextByBlock(b *block.Block) (*data.Context, error) {
	kn.closeLock.RLock()
	defer kn.closeLock.RUnlock()
	if kn.isClose {
		return nil, ErrKernelClosed
	}

	ctx := data.NewContext(kn.store)
	for _, eh := range kn.eventHandlers {
		if err := eh.OnCreateContext(kn, ctx); err != nil {
			return nil, err
		}
	}
	if !b.Header.ChainCoord.Equal(ctx.ChainCoord()) {
		return nil, ErrInvalidChainCoord
	}
	if err := kn.validateBlockBody(ctx, b); err != nil {
		return nil, err
	}
	for i, tx := range b.Body.Transactions {
		if _, err := ctx.Transactor().Execute(ctx, tx, &common.Coordinate{Height: ctx.TargetHeight(), Index: uint16(i)}); err != nil {
			return nil, err
		}
	}
	if err := kn.rewarder.ProcessReward(b.Header.FormulationAddress, ctx); err != nil {
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
func (kn *Kernel) GenerateBlock(ctx *data.Context, TimeoutCount uint32) (*chain.Data, *block.Signed, error) {
	kn.closeLock.RLock()
	defer kn.closeLock.RUnlock()
	if kn.isClose {
		return nil, nil, ErrKernelClosed
	}

	if kn.Config.FormulatorKey == nil {
		return nil, nil, ErrNotFormulator
	}

	b := &block.Block{
		Header: block.Header{
			ChainCoord:         *ctx.ChainCoord(),
			FormulationAddress: kn.Config.FormulatorAddress,
			TimeoutCount:       TimeoutCount,
		},
		Body: block.Body{
			Transactions:          []transaction.Transaction{},
			TransactionSignatures: [][]common.Signature{},
		},
	}

	timer := time.NewTimer(kn.Config.GenTimeThreshold)
	TxHashes := make([]hash.Hash256, 0, 65535)

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
				//TODO : EventTransactionPendingFail
				break
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

	if err := kn.rewarder.ProcessReward(b.Header.FormulationAddress, ctx); err != nil {
		return nil, nil, err
	}

	b.Header.ContextHash = ctx.Hash()

	if h, err := level.BuildLevelRoot(TxHashes); err != nil {
		return nil, nil, err
	} else {
		b.Header.LevelRootHash = h
	}

	var bodyBuffer bytes.Buffer
	if _, err := b.Header.WriteTo(&bodyBuffer); err != nil {
		panic(err)
	}
	var extraBuffer bytes.Buffer
	if _, err := b.Body.WriteTo(&extraBuffer); err != nil {
		panic(err)
	}
	body := bodyBuffer.Bytes()
	cd := &chain.Data{
		Header: chain.Header{
			Version:   kn.Provider().Version(),
			Height:    ctx.TargetHeight(),
			PrevHash:  ctx.PrevHash(),
			BodyHash:  hash.DoubleHash(body),
			Timestamp: uint64(time.Now().UnixNano()),
		},
		Body:       body,
		Extra:      extraBuffer.Bytes(),
		Signatures: []common.Signature{},
	}

	HeaderHash := cd.Header.Hash()
	if sig, err := kn.Config.FormulatorKey.Sign(HeaderHash); err != nil {
		return nil, nil, err
	} else {
		s := &block.Signed{
			HeaderHash:         HeaderHash,
			GeneratorSignature: sig,
		}
		cd.Signatures = append(cd.Signatures, sig)
		return cd, s, nil
	}
}

func (kn *Kernel) validateBlockBody(loader data.Loader, b *block.Block) error {
	var wg sync.WaitGroup
	cpuCnt := runtime.NumCPU()
	if len(b.Body.Transactions) < 1000 {
		cpuCnt = 1
	}
	txCnt := len(b.Body.Transactions) / cpuCnt
	TxHashes := make([]hash.Hash256, len(b.Body.Transactions))
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

func (kn *Kernel) messageCreator(r io.Reader, t message.Type) (message.Message, error) {
	switch t {
	case message_def.TransactionMessageType:
		p := &message_def.TransactionMessage{}
		p.Tran = kn.store.Transactor()
		if _, err := p.ReadFrom(r); err != nil {
			return nil, err
		}
		return p, nil
	default:
		return nil, message.ErrUnknownMessage
	}
}
