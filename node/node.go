package node

import (
	"io"
	"sync"

	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/data"
	"git.fleta.io/fleta/core/transaction"

	"git.fleta.io/fleta/common"

	"git.fleta.io/fleta/framework/router"

	"git.fleta.io/fleta/core/kernel"

	"git.fleta.io/fleta/core/message_def"
	"git.fleta.io/fleta/framework/chain"
	"git.fleta.io/fleta/framework/chain/mesh"
	"git.fleta.io/fleta/framework/message"
	"git.fleta.io/fleta/framework/peer"
)

// Node validates and shares the block chain
type Node struct {
	sync.Mutex
	Config    *Config
	cm        *chain.Manager
	kn        *kernel.Kernel
	pm        peer.Manager
	manager   *message.Manager
	txCastMap map[string]bool
	isRunning bool
	closeLock sync.RWMutex
	runEnd    chan struct{}
	isClose   bool
}

// NewNode returns a Node
func NewNode(Config *Config, kn *kernel.Kernel) (*Node, error) {
	r, err := router.NewRouter(&Config.Router)
	if err != nil {
		return nil, err
	}

	pm, err := peer.NewManager(kn.ChainCoord(), r, &Config.Peer)
	if err != nil {
		return nil, err
	}
	for _, v := range Config.SeedNodes {
		pm.AddNode(v)
	}

	nd := &Node{
		Config:  Config,
		cm:      chain.NewManager(kn),
		pm:      pm,
		kn:      kn,
		manager: message.NewManager(),
		runEnd:  make(chan struct{}),
	}
	nd.manager.SetCreator(message_def.TransactionMessageType, nd.messageCreator)
	nd.cm.Mesh = pm
	nd.pm.RegisterEventHandler(nd.cm)
	return nd, nil
}

// Close terminates the formulator
func (nd *Node) Close() {
	nd.closeLock.Lock()
	defer nd.closeLock.Unlock()

	nd.isClose = true
	nd.kn.Close()
	nd.runEnd <- struct{}{}
}

// Run runs the node
func (nd *Node) Run() {
	nd.Lock()
	if nd.isRunning {
		nd.Unlock()
		return
	}
	nd.isRunning = true
	nd.Unlock()

	nd.pm.StartManage()
	nd.pm.EnforceConnect()
	go nd.cm.Run()

	<-nd.runEnd
}

// CommitTransaction adds and broadcasts transaction
func (nd *Node) CommitTransaction(tx transaction.Transaction, sigs []common.Signature) error {
	if err := nd.kn.AddTransaction(tx, sigs); err != nil {
		return err
	}
	msg := &message_def.TransactionMessage{
		Tx:   tx,
		Sigs: sigs,
		Tran: nd.kn.Transactor(),
	}
	nd.pm.BroadCast(msg)
	return nil
}

// OnRecv is called when a message is received from the peer
func (nd *Node) OnRecv(p mesh.Peer, r io.Reader, t message.Type) error {
	m, err := nd.manager.ParseMessage(r, t)
	if err != nil {
		return err
	}
	if err := nd.handleMessage(p, m); err != nil {
		return err
	}
	return nil
}

func (nd *Node) handleMessage(p mesh.Peer, m message.Message) error {
	switch msg := m.(type) {
	case *message_def.TransactionMessage:
		if err := nd.kn.AddTransaction(msg.Tx, msg.Sigs); err != nil {
			if err != kernel.ErrPastSeq {
				return err
			}
			return nil
		}
		nd.pm.ExceptCast(p.ID(), msg)
		return nil
	default:
		return message.ErrUnhandledMessage
	}
}

func (nd *Node) messageCreator(r io.Reader, t message.Type) (message.Message, error) {
	switch t {
	case message_def.TransactionMessageType:
		p := &message_def.TransactionMessage{
			Tran: nd.kn.Transactor(),
		}
		if _, err := p.ReadFrom(r); err != nil {
			return nil, err
		}
		return p, nil
	default:
		return nil, message.ErrUnknownMessage
	}
}

// OnProcessBlock called when processing block to the chain (error prevent processing block)
func (nd *Node) OnProcessBlock(kn *kernel.Kernel, b *block.Block, s *block.ObserverSigned, ctx *data.Context) error {
	return nil
}

// AfterProcessBlock called when processed block to the chain
func (nd *Node) AfterProcessBlock(kn *kernel.Kernel, b *block.Block, s *block.ObserverSigned, ctx *data.Context) {
}

// OnPushTransaction called when pushing a transaction to the transaction pool (error prevent push transaction)
func (nd *Node) OnPushTransaction(kn *kernel.Kernel, tx transaction.Transaction, sigs []common.Signature) error {
	return nil
}

// AfterPushTransaction called when pushed a transaction to the transaction pool
func (nd *Node) AfterPushTransaction(kn *kernel.Kernel, tx transaction.Transaction, sigs []common.Signature) {
}

// DoTransactionBroadcast called when a transaction need to be broadcast
func (nd *Node) DoTransactionBroadcast(kn *kernel.Kernel, msg *message_def.TransactionMessage) {
	nd.pm.BroadCast(msg)
}

// DebugLog TEMP
func (nd *Node) DebugLog(kn *kernel.Kernel, args ...interface{}) {
}
