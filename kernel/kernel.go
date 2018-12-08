package kernel

import (
	"io"
	"log"
	"sync"

	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/message_def"
	"git.fleta.io/fleta/core/store"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/blockpool"
	"git.fleta.io/fleta/core/chain"
	"git.fleta.io/fleta/core/data"
	"git.fleta.io/fleta/core/generator"
	"git.fleta.io/fleta/core/observer"
	"git.fleta.io/fleta/core/reward"
	"git.fleta.io/fleta/core/transaction"
	"git.fleta.io/fleta/core/txpool"
	"git.fleta.io/fleta/framework/message"
	"git.fleta.io/fleta/framework/peer"
	"git.fleta.io/fleta/framework/router"
)

// Kernel processes the block chain using its components and stores state of the block chain
// It based on Proof-of-Formulation and Account/UTXO hybrid model
// All kinds of accounts and transactions processed the out side of kernel
type Kernel struct {
	peer.BaseEventHandler
	Config           *Config
	chain            *chain.Chain
	Rewarder         reward.Rewarder
	TxPool           *txpool.TransactionPool
	BlockPool        *blockpool.BlockPool
	peerMsgHandler   *message.Manager
	Router           router.Router
	PeerManager      peer.Manager
	processBlockLock sync.Mutex
	closeLock        sync.RWMutex
	eventHandlers    []EventHandler
	isClose          bool
	// Formulator
	generator         *generator.Generator
	observerConnector *observer.Connector
	genBlockLock      sync.Mutex
}

// Config TODO
type Config struct {
	ChainCoord *common.Coordinate
	SeedNodes  []string
	Chain      chain.Config
	Peer       peer.Config
	Router     router.Config
	Generator  generator.Config
	Observer   observer.Config
	StorePath  string
}

// NewKernel returns a Kernel
func NewKernel(Config *Config, st *store.Store, Rewarder reward.Rewarder, GenesisContextData *data.ContextData) (*Kernel, error) {
	cn, err := chain.NewChain(&Config.Chain, st)
	if err != nil {
		return nil, err
	}
	mm := message.NewManager()
	r, err := router.NewRouter(&Config.Router)
	if err != nil {
		return nil, err
	}
	pm, err := peer.NewManager(Config.ChainCoord, r, mm, &Config.Peer)
	if err != nil {
		return nil, err
	}
	for _, v := range Config.SeedNodes {
		pm.AddNode(v)
	}
	kn := &Kernel{
		Config:         Config,
		chain:          cn,
		Rewarder:       Rewarder,
		TxPool:         txpool.NewTransactionPool(),
		BlockPool:      blockpool.NewBlockPool(),
		peerMsgHandler: mm,
		Router:         r,
		PeerManager:    pm,
	}
	if err := cn.Init(GenesisContextData); err != nil {
		return nil, err
	}
	kn.peerMsgHandler.ApplyMessage(message_def.BlockMessageType, kn.blockMessageCreator, kn.blockMessageHandler)
	kn.peerMsgHandler.ApplyMessage(message_def.TransactionMessageType, kn.transactionMessageCreator, kn.transactionMessageHandler)
	kn.peerMsgHandler.ApplyMessage(message_def.StatusMessageType, kn.statusMessageCreator, kn.statusMessageHandler)
	kn.PeerManager.RegisterEventHandler(kn)
	return kn, nil
}

// AddEventHandler adds a event handler to kernel
func (kn *Kernel) AddEventHandler(eh EventHandler) {
	kn.eventHandlers = append(kn.eventHandlers, eh)
}

// Loader returns the loader of the chain
func (kn *Kernel) Loader() data.Loader {
	return kn.chain.Loader()
}

// Provider returns the provider of the chain
func (kn *Kernel) Provider() block.Provider {
	return kn.chain.Provider()
}

// IsMinable returns true when the target address is the top rank's address
func (kn *Kernel) IsMinable(addr common.Address, TimeoutCount uint32) (bool, error) {
	return kn.chain.IsMinable(addr, TimeoutCount)
}

// ValidateObserverSigned validates observer signatures
func (kn *Kernel) ValidateObserverSigned(b *block.Block, s *block.ObserverSigned) error {
	return kn.chain.ValidateObserverSigned(b, s)
}

// ValidateContext validates the context using the block
func (kn *Kernel) ValidateContext(b *block.Block, ctx *data.Context) error {
	return kn.chain.ValidateContext(b, ctx)
}

// AppendBlock stores the block data
func (kn *Kernel) AppendBlock(b *block.Block, s *block.ObserverSigned, ctx *data.Context) error {
	for _, eh := range kn.eventHandlers {
		if err := eh.BeforeAppendBlock(b, s, ctx); err != nil {
			return err
		}
	}
	if err := kn.chain.AppendBlock(b, s, ctx); err != nil {
		return err
	}
	for _, eh := range kn.eventHandlers {
		eh.AfterAppendBlock(b, s, ctx)
	}
	return nil
}

// InitFormulator updates the node as a formulator
func (kn *Kernel) InitFormulator(Generator *generator.Generator, ObserverConnector *observer.Connector) error {
	kn.generator = Generator
	kn.observerConnector = ObserverConnector
	kn.observerConnector.AddMessageHandler(message_def.BlockMessageType, kn.blockMessageCreator, kn.observerBlockMessageHandler)
	return nil
}

// Start runs kernel
func (kn *Kernel) Start() {
	kn.PeerManager.StartManage()
	kn.PeerManager.EnforceConnect()
	if kn.observerConnector != nil {
		kn.observerConnector.Start()
	}
	// TODO : start tx cast manager
	// TODO : start block syncer
}

// Close terminates and cleans the kernel
func (kn *Kernel) Close() {
	kn.closeLock.Lock()
	defer kn.closeLock.Unlock()

	kn.isClose = true
	kn.chain.Close()
}

// IsClose returns the close status of the kernel
func (kn *Kernel) IsClose() bool {
	kn.closeLock.RLock()
	defer kn.closeLock.RUnlock()

	return kn.isClose
}

// TryGenerateBlock generate the next block when this formulator is the top rank
func (kn *Kernel) TryGenerateBlock() error {
	kn.closeLock.RLock()
	defer kn.closeLock.RUnlock()
	if kn.isClose {
		return ErrClosedKernel
	}

	if kn.generator == nil {
		return ErrNotFormulator
	}

	kn.genBlockLock.Lock()
	defer kn.genBlockLock.Unlock()

	if is, err := kn.chain.IsMinable(kn.generator.Address(), 0); err != nil {
		return err
	} else if !is {
		return nil
	}

	ctx := data.NewContext(kn.chain.Loader())
	for _, eh := range kn.eventHandlers {
		if err := eh.OnCreateContext(ctx); err != nil {
			return err
		}
	}
	nb, ns, err := kn.generator.GenerateBlock(kn.TxPool, ctx, 0, kn.Rewarder)
	if err != nil {
		return err
	}
	nos, err := kn.observerConnector.RequestSign(nb, ns)
	if err != nil {
		return err
	}
	cb := func() error {
		go kn.TryGenerateBlock()
		return nil
	}
	if err := kn.BlockPool.Append(nb, nos, ctx, cb); err != nil {
		return err
	}
	if err := kn.tryProcessBlock(); err != nil {
		return err
	}
	return nil
}

// TryProcessBlock pops the block from blockpool and processes it when it is the next block of the chain
func (kn *Kernel) TryProcessBlock() error {
	kn.closeLock.RLock()
	defer kn.closeLock.RUnlock()
	if kn.isClose {
		return ErrClosedKernel
	}
	return kn.tryProcessBlock()
}

func (kn *Kernel) tryProcessBlock() error {
	kn.processBlockLock.Lock()
	defer kn.processBlockLock.Unlock()

	item := kn.BlockPool.Pop(kn.chain.Loader().TargetHeight())
	for item != nil {
		if err := kn.chain.ValidateObserverSigned(item.Block, item.ObserverSigned); err != nil {
			return err
		}
		if item.Context == nil {
			ctx, err := kn.ContextByBlock(item.Block)
			if err != nil {
				return err
			}
			item.Context = ctx
		}
		if err := kn.AppendBlock(item.Block, item.ObserverSigned, item.Context); err != nil {
			return err
		}
		for _, tx := range item.Block.Transactions {
			kn.TxPool.Remove(tx)
		}
		if item.Callback != nil {
			if err := item.Callback(); err != nil {
				return err
			}
		}
		item = kn.BlockPool.Pop(kn.chain.Loader().TargetHeight())
	}
	return nil
}

// ContextByBlock TODO
func (kn *Kernel) ContextByBlock(b *block.Block) (*data.Context, error) {
	kn.closeLock.RLock()
	defer kn.closeLock.RUnlock()
	if kn.isClose {
		return nil, ErrClosedKernel
	}

	ctx := data.NewContext(kn.chain.Loader())
	for _, eh := range kn.eventHandlers {
		if err := eh.OnCreateContext(ctx); err != nil {
			return nil, err
		}
	}
	if err := kn.chain.ProcessBlock(ctx, b, kn.Rewarder); err != nil {
		return nil, err
	}
	return ctx, nil
}

// AddTransaction validate the transaction and push it to the transaction pool
func (kn *Kernel) AddTransaction(tx transaction.Transaction, sigs []common.Signature) error {
	kn.closeLock.RLock()
	defer kn.closeLock.RUnlock()
	if kn.isClose {
		return ErrClosedKernel
	}

	loader := kn.chain.Loader()
	if !tx.ChainCoord().Equal(loader.ChainCoord()) {
		return chain.ErrInvalidChainCoordinate
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
	if err := loader.Transactor().Validate(loader, tx, signers); err != nil {
		return err
	}
	if err := kn.TxPool.Push(tx, sigs); err != nil {
		return err
	}
	return nil
}

// PeerConnected is the callback function to be called when peer connected
func (kn *Kernel) PeerConnected(p peer.Peer) {
	log.Println("PeerConnected ", p.LocalAddr().String(), " : ", p.RemoteAddr().String())
	kn.sendStatusMessage(p)
}

// PeerDisconnected is the callback function to be called when peer disconnected
func (kn *Kernel) PeerDisconnected(p peer.Peer) {
	log.Println("PeerDisconnected ", p.LocalAddr().String(), " : ", p.RemoteAddr().String())
}

func (kn *Kernel) sendStatusMessage(p peer.Peer) {
	loader := kn.chain.Loader()
	msg := &message_def.StatusMessage{
		Version:       kn.chain.Config.Version,
		Height:        loader.TargetHeight() - 1,
		LastBlockHash: loader.LastBlockHash(),
	}
	p.Send(msg)
}

func (kn *Kernel) transactionMessageCreator(r io.Reader) message.Message {
	p := &message_def.TransactionMessage{}
	p.Tran = kn.chain.Loader().Transactor()
	p.ReadFrom(r)
	return p
}

func (kn *Kernel) transactionMessageHandler(m message.Message) error {
	msg := m.(*message_def.TransactionMessage)
	if err := kn.AddTransaction(msg.Tx, msg.Sigs); err != nil {
		return err
	}
	return nil
}

func (kn *Kernel) blockMessageCreator(r io.Reader) message.Message {
	p := message_def.NewBlockMessage(kn.chain.Loader().Transactor())
	p.ReadFrom(r)
	return p
}

func (kn *Kernel) blockMessageHandler(m message.Message) error {
	msg := m.(*message_def.BlockMessage)
	if err := kn.BlockPool.Append(msg.Block, msg.ObserverSigned, nil, nil); err != nil {
		return err
	}
	return nil
}

func (kn *Kernel) observerBlockMessageHandler(m message.Message) error {
	msg := m.(*message_def.BlockMessage)
	cb := func() error {
		go kn.TryGenerateBlock()
		return nil
	}
	if err := kn.BlockPool.Append(msg.Block, msg.ObserverSigned, nil, cb); err != nil {
		return err
	}
	if err := kn.TryProcessBlock(); err != nil {
		return err
	}
	return nil
}

func (kn *Kernel) statusMessageCreator(r io.Reader) message.Message {
	p := &message_def.StatusMessage{}
	p.ReadFrom(r)
	return p
}

func (kn *Kernel) statusMessageHandler(m message.Message) error {
	msg := m.(*message_def.StatusMessage)
	//TODO
	log.Println(msg)
	return nil
}
