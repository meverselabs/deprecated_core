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
	"git.fleta.io/fleta/core/transaction"

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
	ms             *FormulatorMesh
	cm             *chain.Manager
	kn             *kernel.Kernel
	pm             peer.Manager
	manager        *message.Manager
	lastGenMessage *message_def.BlockGenMessage
	lastReqMessage *message_def.BlockReqMessage
	lastContext    *data.Context
	txQueue        *queue.Queue
	txCastMap      map[string]bool
	statusMap      map[string]*chain.Status
	requestedMap   map[uint32]bool
	isProcessing   bool
	isRunning      bool
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
		manager:      message.NewManager(),
		txQueue:      queue.NewQueue(),
		txCastMap:    map[string]bool{},
		statusMap:    map[string]*chain.Status{},
		requestedMap: map[uint32]bool{},
	}
	fr.manager.SetCreator(message_def.BlockReqMessageType, fr.messageCreator)
	fr.manager.SetCreator(message_def.BlockObSignMessageType, fr.messageCreator)
	fr.manager.SetCreator(message_def.TransactionMessageType, fr.messageCreator)
	fr.manager.SetCreator(chain.DataMessageType, fr.messageCreator)
	fr.manager.SetCreator(chain.StatusMessageType, fr.messageCreator)
	kn.AddEventHandler(fr)

	fr.ms = NewFormulatorMesh(Config.Key, Config.Formulator, Config.ObserverKeyMap, fr)
	fr.cm.Mesh = pm
	fr.cm.Deligator = fr
	fr.pm.RegisterEventHandler(fr.cm)

	if err := fr.cm.Init(); err != nil {
		return nil, err
	}
	return fr, nil
}

// NextRoundHash provides next round hash value
func (fr *Formulator) NextRoundHash() hash.Hash256 {
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
	for {
		select {
		case <-timer.C:
			fr.Lock()
			txCastTargetMap := map[string]bool{}
			for _, id := range fr.pm.ConnectedList() {
				if !fr.txCastMap[id] {
					txCastTargetMap[id] = true
				}
			}
			fr.txCastMap = map[string]bool{}
			msgs := []*message_def.TransactionMessage{}
			item := fr.txQueue.Pop()
			for item != nil {
				msgs = append(msgs, item.(*message_def.TransactionMessage))
				item = fr.txQueue.Pop()
			}
			fr.Unlock()

			for _, msg := range msgs {
				if fr.kn.HasTransaction(msg.Tx.Hash()) {
					for id := range txCastTargetMap {
						fr.pm.TargetCast(id, msg)
					}
				}
			}
			timer.Reset(time.Minute)
		}
	}
}

// OnClosed is called when the peer is disconnected
func (fr *Formulator) OnClosed(p mesh.Peer) {
	fr.Lock()
	defer fr.Unlock()

	delete(fr.statusMap, p.ID())
}

// BeforeConnect is called before a new peer is handled
func (fr *Formulator) BeforeConnect(p mesh.Peer) error {
	return nil
}

// AfterConnect is called after a new peer is connected
func (fr *Formulator) AfterConnect(p mesh.Peer) {
	fr.Lock()
	defer fr.Unlock()

	fr.statusMap[p.ID()] = &chain.Status{}
}

// OnCreateContext called when a context creation (error prevent using context)
func (fr *Formulator) OnCreateContext(kn *kernel.Kernel, ctx *data.Context) error {
	return nil
}

// OnProcessBlock called when processing block to the chain (error prevent processing block)
func (fr *Formulator) OnProcessBlock(kn *kernel.Kernel, b *block.Block, s *block.ObserverSigned, ctx *data.Context) error {
	return nil
}

// OnPushTransaction called when pushing a transaction to the transaction pool (error prevent push transaction)
func (fr *Formulator) OnPushTransaction(kn *kernel.Kernel, tx transaction.Transaction, sigs []common.Signature) error {
	return nil
}

// AfterProcessBlock called when processed block to the chain
func (fr *Formulator) AfterProcessBlock(kn *kernel.Kernel, b *block.Block, s *block.ObserverSigned, ctx *data.Context) {
	if fr.isProcessing {
		var MaxID string
		var MaxHeight uint32
		for id, status := range fr.statusMap {
			if MaxHeight < status.Height {
				MaxHeight = status.Height
				MaxID = id
			}
		}
		TargetHeight := fr.kn.Provider().Height() + 1
		for TargetHeight < MaxHeight {
			if fr.requestedMap[TargetHeight] {
				sm := &chain.RequestMessage{
					Height: TargetHeight,
				}
				if err := fr.ms.SendTo(MaxID, sm); err != nil {
					return
				}
				fr.requestedMap[TargetHeight] = true
				return
			}
			TargetHeight++
		}
	}
}

// OnRecv is called when a message is received from the peer
func (fr *Formulator) OnRecv(p mesh.Peer, r io.Reader, t message.Type) error {
	m, err := fr.manager.ParseMessage(r, t)
	if err != nil {
		return err
	}
	if msg, is := m.(*message_def.TransactionMessage); is {
		if err := fr.kn.AddTransaction(msg.Tx, msg.Sigs); err != nil {
			if err != kernel.ErrPastSeq {
				return err
			} else {
				return nil
			}
		}
		fr.pm.ExceptCast(p.ID(), msg)

		fr.Lock()
		for _, id := range fr.pm.ConnectedList() {
			fr.txCastMap[id] = true
		}
		fr.Unlock()
		fr.txQueue.Push(msg)
	} else {
		if err := fr.handleMessage(p, m); err != nil {
			//log.Println(err)
			return nil
		}
	}
	return nil
}

func (fr *Formulator) handleMessage(p mesh.Peer, m message.Message) error {
	fr.Lock()
	defer fr.Unlock()

	switch msg := m.(type) {
	case *message_def.BlockReqMessage:
		//log.Println(fr.Config.Formulator, fr.kn.Provider().Height(), msg.TargetHeight, "BlockReqMessage")
		if msg.TargetHeight <= fr.kn.Provider().Height() {
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

		NextRoundHash := fr.NextRoundHash()
		if !msg.RoundHash.Equal(NextRoundHash) {
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

		cp := fr.kn.Provider()
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
		fr.isProcessing = true
		if err := fr.cm.Process(cd, fr.lastContext); err != nil {
			return err
		}
		fr.isProcessing = false
		fr.cm.BroadcastHeader(cd.Header)
		return nil
	case *chain.DataMessage:
		//log.Println(fr.Config.Formulator, fr.kn.Provider().Height(), "chain.DataMessage")
		if msg.Data.Header.Height() <= fr.kn.Provider().Height() {
			return nil
		}
		if err := fr.cm.AddData(msg.Data); err != nil {
			return err
		}

		status, has := fr.statusMap[p.ID()]
		if !has {
			return nil
		}
		if status.Height < msg.Data.Header.Height() {
			status.Height = msg.Data.Header.Height()
		}
		return nil
	case *chain.StatusMessage:
		//log.Println(fr.Config.Formulator, fr.kn.Provider().Height(), "chain.DataMessage")
		status, has := fr.statusMap[p.ID()]
		if !has {
			return nil
		}
		if status.Height < msg.Height {
			status.Version = msg.Version
			status.Height = msg.Height
			status.PrevHash = msg.PrevHash
		}

		TargetHeight := fr.kn.Provider().Height() + 1
		for TargetHeight < msg.Height {
			if !fr.requestedMap[TargetHeight] {
				sm := &chain.RequestMessage{
					Height: TargetHeight,
				}
				if err := p.Send(sm); err != nil {
					return err
				}
				fr.requestedMap[TargetHeight] = true
			}
			TargetHeight++
		}
		return nil
	default:
		return message.ErrUnhandledMessage
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
