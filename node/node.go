package node

import (
	"io"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/fletaio/core/block"
	"github.com/fletaio/core/data"
	"github.com/fletaio/core/transaction"

	"github.com/fletaio/common"

	"github.com/fletaio/framework/router"

	"github.com/fletaio/core/kernel"

	"github.com/fletaio/core/message_def"
	"github.com/fletaio/framework/chain"
	"github.com/fletaio/framework/chain/mesh"
	"github.com/fletaio/framework/message"
	"github.com/fletaio/framework/peer"
)

// Node validates and shares the block chain
type Node struct {
	sync.Mutex
	Config     *Config
	cm         *chain.Manager
	kn         *kernel.Kernel
	pm         peer.Manager
	mm         *message.Manager
	txMsgChans []*chan *txMsgItem
	txMsgIdx   uint64
	isRunning  bool
	closeLock  sync.RWMutex
	runEnd     chan struct{}
	isClose    bool
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

	nd := &Node{
		Config: Config,
		cm:     chain.NewManager(kn),
		pm:     pm,
		kn:     kn,
		mm:     message.NewManager(),
		runEnd: make(chan struct{}),
	}
	nd.mm.SetCreator(message_def.TransactionMessageType, nd.messageCreator)
	nd.cm.Mesh = pm
	nd.pm.RegisterEventHandler(nd.cm)
	nd.pm.RegisterEventHandler(nd)
	return nd, nil
}

// Close terminates the formulator
func (nd *Node) Close() {
	nd.closeLock.Lock()
	defer nd.closeLock.Unlock()

	nd.Lock()
	defer nd.Unlock()

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
	go func() {
		for _, v := range nd.Config.SeedNodes {
			nd.pm.AddNode(v)
		}
		nd.pm.EnforceConnect()
	}()
	go nd.cm.Run()

	WorkerCount := runtime.NumCPU() - 1
	if WorkerCount < 1 {
		WorkerCount = 1
	}
	workerEnd := make([]*chan struct{}, WorkerCount)
	nd.txMsgChans = make([]*chan *txMsgItem, WorkerCount)
	for i := 0; i < WorkerCount; i++ {
		mch := make(chan *txMsgItem)
		nd.txMsgChans[i] = &mch
		ch := make(chan struct{})
		workerEnd[i] = &ch
		go func(pMsgCh *chan *txMsgItem, pEndCh *chan struct{}) {
			for {
				select {
				case item := <-(*pMsgCh):
					if err := nd.kn.AddTransaction(item.Message.Tx, item.Message.Sigs); err != nil {
						if err != kernel.ErrProcessingTransaction && err != kernel.ErrPastSeq {
							(*item.pErrCh) <- err
						} else {
							(*item.pErrCh) <- nil
						}
						break
					}
					(*item.pErrCh) <- nil
					if len(item.PeerID) > 0 {
						//nd.pm.ExceptCast(item.PeerID, item.Message)
						nd.pm.ExceptCastLimit(item.PeerID, item.Message, 7)
					} else {
						//nd.pm.BroadCast(item.Message)
						nd.pm.BroadCastLimit(item.Message, 7)
					}
				case <-(*pEndCh):
					return
				}
			}
		}(&mch, &ch)
	}

	select {
	case <-nd.runEnd:
		for i := 0; i < WorkerCount; i++ {
			(*workerEnd[i]) <- struct{}{}
		}
	}
}

// CommitTransaction adds and broadcasts transaction
func (nd *Node) CommitTransaction(tx transaction.Transaction, sigs []common.Signature) error {
	msg := &message_def.TransactionMessage{
		Tx:   tx,
		Sigs: sigs,
		Tran: nd.kn.Transactor(),
	}
	errCh := make(chan error)
	idx := atomic.AddUint64(&nd.txMsgIdx, 1) % uint64(len(nd.txMsgChans))
	(*nd.txMsgChans[idx]) <- &txMsgItem{
		Message: msg,
		PeerID:  "",
		pErrCh:  &errCh,
	}
	return <-errCh
}

// OnConnected is called after a new peer is connected
func (nd *Node) OnConnected(p mesh.Peer) {
}

// OnDisconnected is called when the peer is disconnected
func (nd *Node) OnDisconnected(p mesh.Peer) {
}

// OnRecv is called when a message is received from the peer
func (nd *Node) OnRecv(p mesh.Peer, r io.Reader, t message.Type) error {
	m, err := nd.mm.ParseMessage(r, t)
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
		errCh := make(chan error)
		idx := atomic.AddUint64(&nd.txMsgIdx, 1) % uint64(len(nd.txMsgChans))
		(*nd.txMsgChans[idx]) <- &txMsgItem{
			Message: msg,
			PeerID:  "",
			pErrCh:  &errCh,
		}
		return <-errCh
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

type txMsgItem struct {
	Message *message_def.TransactionMessage
	PeerID  string
	pErrCh  *chan error
}
