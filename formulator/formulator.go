package formulator

import (
	"io"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fletaio/framework/router"

	"github.com/fletaio/core/data"
	"github.com/fletaio/core/kernel"
	"github.com/fletaio/core/transaction"
	"github.com/fletaio/core/txpool"

	"github.com/fletaio/common"
	"github.com/fletaio/core/block"
	"github.com/fletaio/core/message_def"
	"github.com/fletaio/framework/chain"
	"github.com/fletaio/framework/chain/mesh"
	"github.com/fletaio/framework/message"
	"github.com/fletaio/framework/peer"
)

// Formulator procudes a block by the consensus
type Formulator struct {
	sync.Mutex
	Config               *Config
	ms                   *Mesh
	cm                   *chain.Manager
	kn                   *kernel.Kernel
	pm                   peer.Manager
	mm                   *message.Manager
	lastGenMessages      []*message_def.BlockGenMessage
	lastObSignMessageMap map[uint32]*message_def.BlockObSignMessage
	lastContextes        []*data.Context
	lastReqMessage       *message_def.BlockReqMessage
	txMsgChans           []*chan *txMsgItem
	txMsgIdx             uint64
	statusMap            map[string]*chain.Status
	requestTimer         *chain.RequestTimer
	requestLock          sync.RWMutex
	isRunning            bool
	closeLock            sync.RWMutex
	runEnd               chan struct{}
	isClose              bool
}

// NewFormulator returns a Formulator
func NewFormulator(Config *Config, kn *kernel.Kernel) (*Formulator, error) {
	r, err := router.NewRouter(&Config.Router, kn.ChainCoord())
	if err != nil {
		return nil, err
	}

	pm, err := peer.NewManager(kn.ChainCoord(), r, &Config.Peer)
	if err != nil {
		return nil, err
	}

	fr := &Formulator{
		Config:               Config,
		cm:                   chain.NewManager(kn),
		pm:                   pm,
		kn:                   kn,
		mm:                   message.NewManager(),
		lastGenMessages:      []*message_def.BlockGenMessage{},
		lastObSignMessageMap: map[uint32]*message_def.BlockObSignMessage{},
		lastContextes:        []*data.Context{},
		statusMap:            map[string]*chain.Status{},
		requestTimer:         chain.NewRequestTimer(nil),
		runEnd:               make(chan struct{}),
	}
	fr.mm.SetCreator(message_def.BlockReqMessageType, fr.messageCreator)
	fr.mm.SetCreator(message_def.BlockObSignMessageType, fr.messageCreator)
	fr.mm.SetCreator(message_def.TransactionMessageType, fr.messageCreator)
	fr.mm.SetCreator(chain.DataMessageType, fr.messageCreator)
	fr.mm.SetCreator(chain.StatusMessageType, fr.messageCreator)

	fr.ms = NewMesh(Config.Key, Config.Formulator, Config.ObserverKeyMap, fr)
	fr.cm.Mesh = pm
	fr.pm.RegisterEventHandler(fr.cm)
	fr.pm.RegisterEventHandler(fr)
	return fr, nil
}

// Close terminates the formulator
func (fr *Formulator) Close() {
	fr.closeLock.Lock()
	defer fr.closeLock.Unlock()

	fr.Lock()
	defer fr.Unlock()

	fr.isClose = true
	fr.kn.Close()
	fr.runEnd <- struct{}{}
}

// Run runs the formulator
func (fr *Formulator) Run() {
	fr.Lock()
	if fr.isRunning {
		fr.Unlock()
		return
	}
	fr.isRunning = true
	fr.Unlock()

	fr.pm.StartManage()
	go func() {
		for _, v := range fr.Config.SeedNodes {
			fr.pm.AddNode(v)
		}
	}()
	go fr.cm.Run()
	go fr.ms.Run()

	WorkerCount := runtime.NumCPU() - 1
	if WorkerCount < 1 {
		WorkerCount = 1
	}
	workerEnd := make([]*chan struct{}, WorkerCount)
	fr.txMsgChans = make([]*chan *txMsgItem, WorkerCount)
	for i := 0; i < WorkerCount; i++ {
		mch := make(chan *txMsgItem)
		fr.txMsgChans[i] = &mch
		ch := make(chan struct{})
		workerEnd[i] = &ch
		go func(pMsgCh *chan *txMsgItem, pEndCh *chan struct{}) {
			for {
				select {
				case item := <-(*pMsgCh):
					if err := fr.kn.AddTransaction(item.Message.Tx, item.Message.Sigs); err != nil {
						if err != kernel.ErrProcessingTransaction && err != kernel.ErrPastSeq {
							(*item.pErrCh) <- err
						} else {
							(*item.pErrCh) <- nil
						}
						break
					}
					(*item.pErrCh) <- nil
					if len(item.PeerID) > 0 {
						//fr.pm.ExceptCast(item.PeerID, item.Message)
						fr.pm.ExceptCastLimit(item.PeerID, item.Message, 7)
					} else {
						//fr.pm.BroadCast(item.Message)
						fr.pm.BroadCastLimit(item.Message, 7)
					}
				case <-(*pEndCh):
					return
				}
			}
		}(&mch, &ch)
	}

	select {
	case <-fr.runEnd:
		for i := 0; i < WorkerCount; i++ {
			(*workerEnd[i]) <- struct{}{}
		}
	}
}

// OnConnected is called after a new peer is connected
func (fr *Formulator) OnConnected(p mesh.Peer) {
}

// OnDisconnected is called when the peer is disconnected
func (fr *Formulator) OnDisconnected(p mesh.Peer) {
}

// OnObserverConnected is called after a new observer peer is connected
func (fr *Formulator) OnObserverConnected(p *Peer) {
	fr.Lock()
	fr.statusMap[p.ID()] = &chain.Status{}
	fr.Unlock()
}

// OnObserverDisconnected is called when the observer peer is disconnected
func (fr *Formulator) OnObserverDisconnected(p *Peer) {
	fr.Lock()
	delete(fr.statusMap, p.ID())
	fr.Unlock()
}

// OnRecv is called when a message is received from the peer
func (fr *Formulator) OnRecv(p mesh.Peer, r io.Reader, t message.Type) error {
	m, err := fr.mm.ParseMessage(r, t)
	if err != nil {
		return err
	}
	if err := fr.handleMessage(p, m, 0); err != nil {
		//log.Println(err)
		return nil
	}
	return nil
}

func (fr *Formulator) handleMessage(p mesh.Peer, m message.Message, RetryCount int) error {
	switch msg := m.(type) {
	case *message_def.BlockReqMessage:
		fr.Lock()
		defer fr.Unlock()

		//fr.kn.DebugLog("Formulator", fr.kn.Provider().Height(), "BlockReqMessage", msg.TargetHeight, RetryCount)
		cp := fr.kn.Provider()
		Height := cp.Height()
		if msg.TargetHeight <= Height {
			return nil
		}
		if msg.TargetHeight > Height+1 {
			if RetryCount >= 40 {
				return nil
			}
			go func() {
				fr.tryRequestNext()
				time.Sleep(50 * time.Millisecond)
				fr.handleMessage(p, m, RetryCount+1)
			}()
			return nil
		}

		Top, err := fr.kn.TopRank(int(msg.TimeoutCount))
		if err != nil {
			return err
		}
		if !msg.Formulator.Equal(Top.Address) {
			return ErrInvalidRequest
		}
		if !msg.Formulator.Equal(fr.Config.Formulator) {
			return ErrInvalidRequest
		}
		if !msg.FormulatorPublicHash.Equal(common.NewPublicHash(fr.Config.Key.PublicKey())) {
			return ErrInvalidRequest
		}

		if !msg.PrevHash.Equal(cp.LastHash()) {
			return ErrInvalidRequest
		}
		if msg.TargetHeight != Height+1 {
			return ErrInvalidRequest
		}

		fr.lastGenMessages = []*message_def.BlockGenMessage{}
		fr.lastObSignMessageMap = map[uint32]*message_def.BlockObSignMessage{}
		fr.lastContextes = []*data.Context{}
		fr.lastReqMessage = msg

		var ctx *data.Context
		start := time.Now().UnixNano()
		StartBlockTime := uint64(time.Now().UnixNano())
		bNoDelay := false
		if Height > 0 {
			LastHeader, err := cp.Header(Height)
			if err != nil {
				return err
			}
			if StartBlockTime < LastHeader.Timestamp() {
				StartBlockTime = LastHeader.Timestamp() + uint64(time.Millisecond)
			} else if LastHeader.Timestamp() < uint64(fr.kn.Config.MaxBlocksPerFormulator)*uint64(500*time.Millisecond) {
				bNoDelay = true
			}
		}
		for i := uint32(0); i < fr.kn.Config.MaxBlocksPerFormulator; i++ {
			var TimeoutCount uint32
			if i == 0 {
				ctx = data.NewContext(fr.kn.Loader())
				TimeoutCount = msg.TimeoutCount
			} else {
				ctx = ctx.NextContext(fr.lastGenMessages[len(fr.lastGenMessages)-1].Block.Header.Hash())
			}
			Timestamp := StartBlockTime
			if bNoDelay {
				Timestamp += uint64(i) * uint64(time.Millisecond)
			} else {
				Timestamp += uint64(i) * uint64(500*time.Millisecond)
			}
			b, err := fr.kn.GenerateBlock(ctx, TimeoutCount, Timestamp, fr.Config.Formulator)
			if err != nil {
				return err
			}

			nm := &message_def.BlockGenMessage{
				Block: b,
				Tran:  fr.kn.Transactor(),
			}

			if sig, err := fr.Config.Key.Sign(b.Header.Hash()); err != nil {
				return err
			} else {
				nm.GeneratorSignature = sig
			}

			//fr.kn.DebugLog("Formulator", fr.kn.Provider().Height(), "Send BlockGenMessage", nm.Block.Header.Height())
			if err := p.Send(nm); err != nil {
				return err
			}

			fr.lastGenMessages = append(fr.lastGenMessages, nm)
			fr.lastContextes = append(fr.lastContextes, ctx)

			ExpectedTime := time.Duration(i+1) * 200 * time.Millisecond
			PastTime := time.Duration(time.Now().UnixNano() - start)
			if ExpectedTime > PastTime {
				time.Sleep(ExpectedTime - PastTime)
			}
		}
		return nil
	case *message_def.BlockObSignMessage:
		fr.Lock()
		defer fr.Unlock()

		//fr.kn.DebugLog("Formulator", fr.kn.Provider().Height(), "BlockObSignMessage", msg.TargetHeight)
		if len(fr.lastGenMessages) == 0 {
			return nil
		}
		if msg.TargetHeight <= fr.kn.Provider().Height() {
			return nil
		}
		if msg.TargetHeight >= fr.lastReqMessage.TargetHeight+10 {
			return ErrInvalidRequest
		}
		fr.lastObSignMessageMap[msg.TargetHeight] = msg

		for len(fr.lastGenMessages) > 0 {
			GenMessage := fr.lastGenMessages[0]
			sm, has := fr.lastObSignMessageMap[GenMessage.Block.Header.Height()]
			if !has {
				break
			}
			if GenMessage.Block.Header.Height() == sm.TargetHeight {
				ctx := fr.lastContextes[0]

				if !sm.ObserverSigned.HeaderHash.Equal(GenMessage.Block.Header.Hash()) {
					return ErrInvalidRequest
				}

				cd := &chain.Data{
					Header:     GenMessage.Block.Header,
					Body:       GenMessage.Block.Body,
					Signatures: append([]common.Signature{sm.ObserverSigned.GeneratorSignature}, sm.ObserverSigned.ObserverSignatures...),
				}
				if err := fr.cm.Process(cd, ctx); err != nil {
					return err
				}
				//fr.kn.DebugLog("Formulator", fr.kn.Provider().Height(), "Send BroadcastHeader", cd.Header.Height())
				fr.cm.BroadcastHeader(cd.Header)

				if status, has := fr.statusMap[p.ID()]; has {
					if status.Height < GenMessage.Block.Header.Height() {
						status.Height = GenMessage.Block.Header.Height()
					}
				}

				if len(fr.lastGenMessages) > 1 {
					fr.lastGenMessages = fr.lastGenMessages[1:]
					fr.lastContextes = fr.lastContextes[1:]
				} else {
					fr.lastGenMessages = []*message_def.BlockGenMessage{}
					fr.lastContextes = []*data.Context{}
				}
			}
		}
		return nil
	case *chain.DataMessage:
		//fr.kn.DebugLog("Formulator", fr.kn.Provider().Height(), "chain.DataMessage", msg.Data.Header.Height())
		if msg.Data.Header.Height() <= fr.kn.Provider().Height() {
			return nil
		}
		if err := fr.cm.AddData(msg.Data); err != nil {
			return err
		}

		fr.requestTimer.Remove(msg.Data.Header.Height())

		fr.Lock()
		if status, has := fr.statusMap[p.ID()]; has {
			if status.Height < msg.Data.Header.Height() {
				status.Height = msg.Data.Header.Height()
			}
		}
		fr.Unlock()

		fr.tryRequestNext()
		return nil
	case *chain.StatusMessage:
		//fr.kn.DebugLog("Formulator", fr.kn.Provider().Height(), "chain.StatusMessage", msg.Height)
		fr.Lock()
		defer fr.Unlock()

		if status, has := fr.statusMap[p.ID()]; has {
			if status.Height < msg.Height {
				status.Version = msg.Version
				status.Height = msg.Height
				status.LastHash = msg.LastHash
			}
		}

		TargetHeight := fr.kn.Provider().Height() + 1
		for TargetHeight <= msg.Height {
			if !fr.requestTimer.Exist(TargetHeight) {
				if !fr.cm.IsExistData(TargetHeight) {
					sm := &chain.RequestMessage{
						Height: TargetHeight,
					}
					//fr.kn.DebugLog("Formulator", fr.kn.Provider().Height(), "Send RequestMessage by StatusMessage", sm.Height)
					if err := p.Send(sm); err != nil {
						return err
					}
					fr.requestTimer.Add(TargetHeight, 10*time.Second, p.ID())
				}
			}
			TargetHeight++
		}
		return nil
	case *message_def.TransactionMessage:
		errCh := make(chan error)
		idx := atomic.AddUint64(&fr.txMsgIdx, 1) % uint64(len(fr.txMsgChans))
		(*fr.txMsgChans[idx]) <- &txMsgItem{
			Message: msg,
			PeerID:  "",
			pErrCh:  &errCh,
		}
		err := <-errCh
		if err != kernel.ErrInvalidUTXO && err != txpool.ErrExistTransaction {
			return err
		}
		return nil
	default:
		return message.ErrUnhandledMessage
	}
}

func (fr *Formulator) tryRequestNext() {
	fr.requestLock.Lock()
	defer fr.requestLock.Unlock()

	TargetHeight := fr.kn.Provider().Height() + 1
	if !fr.requestTimer.Exist(TargetHeight) {
		if !fr.cm.IsExistData(TargetHeight) {
			fr.Lock()
			defer fr.Unlock()

			for id, status := range fr.statusMap {
				if TargetHeight <= status.Height {
					sm := &chain.RequestMessage{
						Height: TargetHeight,
					}
					if err := fr.ms.SendTo(id, sm); err != nil {
						return
					}
					fr.requestTimer.Add(TargetHeight, 10*time.Second, id)
					return
				}
			}
		}
	}
}

func (fr *Formulator) messageCreator(r io.Reader, t message.Type) (message.Message, error) {
	switch t {
	case message_def.BlockReqMessageType:
		p := &message_def.BlockReqMessage{}
		if _, err := p.ReadFrom(r); err != nil {
			return nil, err
		}
		return p, nil
	case message_def.BlockObSignMessageType:
		p := &message_def.BlockObSignMessage{
			ObserverSigned: &block.ObserverSigned{},
		}
		if _, err := p.ReadFrom(r); err != nil {
			return nil, err
		}
		return p, nil
	case message_def.TransactionMessageType:
		p := &message_def.TransactionMessage{
			Tran: fr.kn.Transactor(),
		}
		if _, err := p.ReadFrom(r); err != nil {
			return nil, err
		}
		return p, nil
	case chain.DataMessageType:
		p := &chain.DataMessage{
			Data: &chain.Data{
				Header: fr.kn.Provider().CreateHeader(),
				Body:   fr.kn.Provider().CreateBody(),
			},
		}
		if _, err := p.ReadFrom(r); err != nil {
			return nil, err
		}
		return p, nil
	case chain.StatusMessageType:
		p := &chain.StatusMessage{}
		if _, err := p.ReadFrom(r); err != nil {
			return nil, err
		}
		return p, nil
	default:
		return nil, message.ErrUnknownMessage
	}
}

// OnProcessBlock called when processing block to the chain (error prevent processing block)
func (fr *Formulator) OnProcessBlock(kn *kernel.Kernel, b *block.Block, s *block.ObserverSigned, ctx *data.Context) error {
	return nil
}

// AfterProcessBlock called when processed block to the chain
func (fr *Formulator) AfterProcessBlock(kn *kernel.Kernel, b *block.Block, s *block.ObserverSigned, ctx *data.Context) {
}

// OnPushTransaction called when pushing a transaction to the transaction pool (error prevent push transaction)
func (fr *Formulator) OnPushTransaction(kn *kernel.Kernel, tx transaction.Transaction, sigs []common.Signature) error {
	return nil
}

// AfterPushTransaction called when pushed a transaction to the transaction pool
func (fr *Formulator) AfterPushTransaction(kn *kernel.Kernel, tx transaction.Transaction, sigs []common.Signature) {
}

// DoTransactionBroadcast called when a transaction need to be broadcast
func (fr *Formulator) DoTransactionBroadcast(kn *kernel.Kernel, msg *message_def.TransactionMessage) {
	fr.pm.BroadCast(msg)
}

// DebugLog TEMP
func (fr *Formulator) DebugLog(kn *kernel.Kernel, args ...interface{}) {
}

type txMsgItem struct {
	Message *message_def.TransactionMessage
	PeerID  string
	pErrCh  *chan error
}
