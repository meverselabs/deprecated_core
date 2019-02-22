package formulator

import (
	"bytes"
	"io"
	"sync"
	"time"

	"git.fleta.io/fleta/common/queue"

	"git.fleta.io/fleta/framework/router"

	"git.fleta.io/fleta/core/data"
	"git.fleta.io/fleta/core/kernel"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/message_def"
	"git.fleta.io/fleta/framework/chain"
	"git.fleta.io/fleta/framework/chain/mesh"
	"git.fleta.io/fleta/framework/message"
	"git.fleta.io/fleta/framework/peer"
)

// Formulator procudes a block by the consensus
type Formulator struct {
	sync.Mutex
	Config         *Config
	ms             *Mesh
	cm             *chain.Manager
	kn             *kernel.Kernel
	pm             peer.Manager
	mm             *message.Manager
	lastGenMessage *message_def.BlockGenMessage
	lastReqMessage *message_def.BlockReqMessage
	lastContext    *data.Context
	txQueue        *queue.Queue
	txCastMap      map[string]bool
	statusMap      map[string]*chain.Status
	requestTimer   *chain.RequestTimer
	requestLock    sync.RWMutex
	isProcessing   bool
	isRunning      bool
	closeLock      sync.RWMutex
	runEnd         chan struct{}
	isClose        bool
}

// NewFormulator returns a Formulator
func NewFormulator(Config *Config, kn *kernel.Kernel) (*Formulator, error) {
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

	fr := &Formulator{
		Config:       Config,
		cm:           chain.NewManager(kn),
		pm:           pm,
		kn:           kn,
		mm:           message.NewManager(),
		txQueue:      queue.NewQueue(),
		txCastMap:    map[string]bool{},
		statusMap:    map[string]*chain.Status{},
		requestTimer: chain.NewRequestTimer(nil),
		runEnd:       make(chan struct{}, 1),
	}
	fr.mm.SetCreator(message_def.BlockReqMessageType, fr.messageCreator)
	fr.mm.SetCreator(message_def.BlockObSignMessageType, fr.messageCreator)
	fr.mm.SetCreator(message_def.TransactionMessageType, fr.messageCreator)
	fr.mm.SetCreator(chain.DataMessageType, fr.messageCreator)
	fr.mm.SetCreator(chain.StatusMessageType, fr.messageCreator)

	fr.ms = NewMesh(Config.Key, Config.Formulator, Config.ObserverKeyMap, fr)
	fr.cm.Mesh = pm
	fr.pm.RegisterEventHandler(fr.cm)
	return fr, nil
}

// Close terminates the formulator
func (fr *Formulator) Close() {
	fr.closeLock.Lock()
	defer fr.closeLock.Unlock()

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
	fr.pm.EnforceConnect()
	go fr.cm.Run()
	go fr.ms.Run()

	timer := time.NewTimer(time.Minute)
	for !fr.isClose {
		select {
		case <-timer.C:
			txCastTargetMap := map[string]bool{}
			fr.Lock()
			for _, id := range fr.pm.ConnectedList() {
				if !fr.txCastMap[id] {
					txCastTargetMap[id] = true
				}
			}
			fr.txCastMap = map[string]bool{}
			fr.Unlock()

			msgs := []*message_def.TransactionMessage{}
			item := fr.txQueue.Pop()
			for item != nil {
				msgs = append(msgs, item.(*message_def.TransactionMessage))
				item = fr.txQueue.Pop()
			}

			for _, msg := range msgs {
				if fr.kn.HasTransaction(msg.Tx.Hash()) {
					for id := range txCastTargetMap {
						fr.pm.TargetCast(id, msg)
					}
				}
			}
			timer.Reset(time.Minute)
		case <-fr.runEnd:
		}
	}
}

// OnDisconnected is called when the peer is disconnected
func (fr *Formulator) OnDisconnected(p mesh.Peer) {
	fr.Lock()
	delete(fr.statusMap, p.ID())
	fr.Unlock()
}

// OnConnected is called after a new peer is connected
func (fr *Formulator) OnConnected(p mesh.Peer) {
	fr.Lock()
	fr.statusMap[p.ID()] = &chain.Status{}
	fr.Unlock()
}

// OnRecv is called when a message is received from the peer
func (fr *Formulator) OnRecv(p mesh.Peer, r io.Reader, t message.Type) error {
	m, err := fr.mm.ParseMessage(r, t)
	if err != nil {
		return err
	}
	if err := fr.handleMessage(p, m); err != nil {
		//log.Println(err)
		return nil
	}
	return nil
}

func (fr *Formulator) handleMessage(p mesh.Peer, m message.Message) error {
	switch msg := m.(type) {
	case *message_def.BlockReqMessage:
		fr.Lock()
		defer fr.Unlock()

		//log.Println(fr.Config.Formulator, fr.kn.Provider().Height(), msg.TargetHeight, "BlockReqMessage")
		cp := fr.kn.Provider()
		if msg.TargetHeight <= cp.Height() {
			return nil
		}
		if fr.lastGenMessage != nil {
			if fr.lastGenMessage.RoundHash.Equal(msg.RoundHash) {
				if fr.lastGenMessage.Block.Header.TimeoutCount == msg.TimeoutCount {
					if err := p.Send(fr.lastGenMessage); err != nil {
						return err
					}
				}
				return nil
			}
		}

		nextRoundHash := fr.nextRoundHash()
		if !msg.RoundHash.Equal(nextRoundHash) {
			return ErrInvalidRequest
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

		if !msg.PrevHash.Equal(cp.PrevHash()) {
			return ErrInvalidRequest
		}
		if msg.TargetHeight != cp.Height()+1 {
			return ErrInvalidRequest
		}

		ctx, b, err := fr.kn.GenerateBlock(msg.TimeoutCount, fr.Config.Formulator)
		if err != nil {
			return err
		}

		nm := &message_def.BlockGenMessage{
			RoundHash: msg.RoundHash,
			Block:     b,
			Tran:      fr.kn.Transactor(),
		}

		if sig, err := fr.Config.Key.Sign(b.Header.Hash()); err != nil {
			return err
		} else {
			nm.GeneratorSignature = sig
		}

		if err := p.Send(nm); err != nil {
			return err
		}

		fr.lastGenMessage = nm
		fr.lastReqMessage = msg
		fr.lastContext = ctx

		return nil
	case *message_def.BlockObSignMessage:
		fr.Lock()
		defer fr.Unlock()

		//log.Println(fr.Config.Formulator, fr.kn.Provider().Height(), "BlockObSignMessage")
		if fr.lastGenMessage == nil {
			return nil
		}
		if msg.TargetHeight <= fr.kn.Provider().Height() {
			return nil
		}
		if !msg.ObserverSigned.HeaderHash.Equal(fr.lastGenMessage.Block.Header.Hash()) {
			return ErrInvalidRequest
		}

		cd := &chain.Data{
			Header:     fr.lastGenMessage.Block.Header,
			Body:       fr.lastGenMessage.Block.Body,
			Signatures: append([]common.Signature{msg.ObserverSigned.GeneratorSignature}, msg.ObserverSigned.ObserverSignatures...),
		}
		if err := fr.cm.ProcessWithCallback(cd, fr.lastContext, func() {
			fr.isProcessing = true
		}, func() {
			fr.isProcessing = false
		}); err != nil {
			return err
		}
		fr.cm.BroadcastHeader(cd.Header)

		if status, has := fr.statusMap[p.ID()]; has {
			if status.Height < fr.lastGenMessage.Block.Header.Height() {
				status.Height = fr.lastGenMessage.Block.Header.Height()
			}
		}

		go fr.tryRequestNext()
		return nil
	case *chain.DataMessage:
		//log.Println(fr.Config.Formulator, fr.kn.Provider().Height(), "chain.DataMessage")
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
		//log.Println(fr.Config.Formulator, fr.kn.Provider().Height(), "chain.StatusMessage")
		fr.Lock()
		if status, has := fr.statusMap[p.ID()]; has {
			if status.Height < msg.Height {
				status.Version = msg.Version
				status.Height = msg.Height
				status.PrevHash = msg.PrevHash
			}
		}
		fr.Unlock()

		TargetHeight := fr.kn.Provider().Height() + 1
		for TargetHeight <= msg.Height {
			if !fr.requestTimer.Exist(TargetHeight) {
				sm := &chain.RequestMessage{
					Height: TargetHeight,
				}
				if err := p.Send(sm); err != nil {
					return err
				}
				fr.requestTimer.Add(TargetHeight, 10*time.Second, p.ID())
			}
			TargetHeight++
		}
		return nil
	case *message_def.TransactionMessage:
		if err := fr.kn.AddTransaction(msg.Tx, msg.Sigs); err != nil {
			if err != kernel.ErrPastSeq {
				return err
			}
			return nil
		}
		fr.pm.ExceptCast(p.ID(), msg)
		fr.Lock()
		for _, id := range fr.pm.ConnectedList() {
			fr.txCastMap[id] = true
		}
		fr.Unlock()
		fr.txQueue.Push(msg)
		return nil
	default:
		return message.ErrUnhandledMessage
	}
}

func (fr *Formulator) tryRequestNext() {
	fr.requestLock.Lock()
	defer fr.requestLock.Unlock()

	TargetHeight := fr.kn.Provider().Height()
	if !fr.requestTimer.Exist(TargetHeight) {
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

func (fr *Formulator) nextRoundHash() hash.Hash256 {
	cp := fr.kn.Provider()
	var buffer bytes.Buffer
	if _, err := fr.kn.ChainCoord().WriteTo(&buffer); err != nil {
		panic(err)
	}
	buffer.WriteString(",")
	PrevHash := cp.PrevHash()
	if _, err := PrevHash.WriteTo(&buffer); err != nil {
		panic(err)
	}
	buffer.WriteString(",")
	if _, err := util.WriteUint32(&buffer, cp.Height()+1); err != nil {
		panic(err)
	}
	return hash.DoubleHash(buffer.Bytes())
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
