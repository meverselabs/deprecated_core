package observer

import (
	"bytes"
	"io"
	"log"
	"sync"
	"time"

	"git.fleta.io/fleta/core/consensus"
	"git.fleta.io/fleta/core/data"
	"git.fleta.io/fleta/core/transaction"

	"git.fleta.io/fleta/core/kernel"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/message_def"
	"git.fleta.io/fleta/framework/chain"
	"git.fleta.io/fleta/framework/chain/mesh"
	"git.fleta.io/fleta/framework/message"
)

// Observer supports the chain validation
type Observer struct {
	sync.Mutex
	Config           *Config
	round            *VoteRound
	roundState       int
	roundVoteMap     map[common.PublicHash]*RoundVote
	roundFirstTime   uint64
	roundFirstHeight uint32
	ignoreMap        map[common.Address]int64
	ms               *ObserverMesh
	cm               *chain.Manager
	kn               *kernel.Kernel
	fs               *FormulatorService
	mm               *message.Manager
	failTimer        *time.Timer
	isRunning        bool
	isProcessing     bool
	closeLock        sync.RWMutex
	runEnd           chan struct{}
	isClose          bool
}

// NewObserver returns a Observer
func NewObserver(Config *Config, kn *kernel.Kernel) (*Observer, error) {
	ob := &Observer{
		Config:       Config,
		roundVoteMap: map[common.PublicHash]*RoundVote{},
		roundState:   RoundVoteState,
		ignoreMap:    map[common.Address]int64{},
		cm:           chain.NewManager(kn),
		kn:           kn,
		mm:           message.NewManager(),
		runEnd:       make(chan struct{}, 1),
	}
	ob.mm.SetCreator(RoundVoteMessageType, ob.messageCreator)
	ob.mm.SetCreator(RoundVoteAckMessageType, ob.messageCreator)
	ob.mm.SetCreator(BlockVoteMessageType, ob.messageCreator)
	ob.mm.SetCreator(message_def.BlockGenMessageType, ob.messageCreator)
	ob.mm.SetCreator(chain.RequestMessageType, ob.messageCreator)
	kn.AddEventHandler(ob)

	ob.fs = NewFormulatorService(Config.Key, kn, ob)
	ob.ms = NewObserverMesh(Config.Key, Config.ObserverKeyMap, ob, ob.cm)
	ob.cm.Mesh = ob.ms
	return ob, nil
}

// Close terminates the observer
func (ob *Observer) Close() {
	ob.closeLock.Lock()
	defer ob.closeLock.Unlock()

	ob.isClose = true
	ob.kn.Close()
	ob.runEnd <- struct{}{}
}

// Run operates servers and a round voting
func (ob *Observer) Run(BindObserver string, BindFormulator string) {
	ob.Lock()
	if ob.isRunning {
		ob.Unlock()
		return
	}
	ob.isRunning = true
	ob.Unlock()

	go ob.ms.Run(BindObserver)
	go ob.fs.Run(BindFormulator)
	go ob.cm.Run()

	voteTimer := time.NewTimer(time.Millisecond)
	ob.failTimer = time.NewTimer(5 * time.Second)
	for !ob.isClose {
		select {
		case <-voteTimer.C:
			ob.fs.Lock()
			is := len(ob.fs.peerHash) > 0
			ob.fs.Unlock()
			log.Println(ob.kn.Provider().Height(), "Current State", ob.roundState, len(ob.adjustFormulatorMap()), len(ob.fs.peerHash), is)
			if is {
				if ob.roundState == RoundVoteState {
					ob.Lock()
					ob.sendRoundVote()
					ob.Unlock()
				}
			}
			voteTimer.Reset(500 * time.Millisecond)
		case <-ob.failTimer.C:
			ob.Lock()
			if len(ob.fs.peerHash) > 0 && ob.roundState != RoundVoteState {
				log.Println(ob.kn.Provider().Height(), "Fail State", ob.roundState, len(ob.adjustFormulatorMap()), len(ob.fs.peerHash))
				if ob.round != nil && ob.round.MinRoundVoteAck != nil {
					addr := ob.round.MinRoundVoteAck.Formulator
					_, has := ob.ignoreMap[addr]
					ob.ignoreMap[addr] = time.Now().UnixNano() + int64(30*time.Second)
					if has {
						ob.fs.RemovePeer(addr)
					}
				}
				ob.roundState = RoundVoteState
				log.Println(ob.kn.Provider().Height(), "Change State", RoundVoteState)
				ob.roundVoteMap = map[common.PublicHash]*RoundVote{}
				ob.round = nil
				ob.roundFirstTime = 0
				ob.roundFirstHeight = 0
				ob.sendRoundVote()
			}
			ob.Unlock()
			ob.failTimer.Reset(5 * time.Second)
		case <-ob.runEnd:
			return
		}
	}
}

// OnFormulatorDisconnected is called after a formulator disconnected
func (ob *Observer) OnFormulatorDisconnected(p *FormulatorPeer) {
}

// OnFormulatorConnected is called after a new formulator connected
func (ob *Observer) OnFormulatorConnected(p *FormulatorPeer) {
	//log.Println("OnFormulatorConnected", ob.round != nil, ob.round != nil && ob.round.MinRoundVoteAck != nil)

	cp := ob.kn.Provider()
	p.Send(&chain.StatusMessage{
		Version:  cp.Version(),
		Height:   cp.Height(),
		LastHash: cp.LastHash(),
	})

	var msg message.Message
	ob.Lock()
	if ob.round != nil && ob.round.MinRoundVoteAck != nil {
		if ob.round.MinRoundVoteAck.Formulator.Equal(p.address) {
			msg = &message_def.BlockReqMessage{
				RoundHash:            ob.round.RoundHash,
				PrevHash:             cp.LastHash(),
				TargetHeight:         cp.Height() + 1,
				TimeoutCount:         ob.round.MinRoundVoteAck.TimeoutCount,
				Formulator:           ob.round.MinRoundVoteAck.Formulator,
				FormulatorPublicHash: ob.round.MinRoundVoteAck.FormulatorPublicHash,
			}
		}
	}
	ob.Unlock()

	if msg != nil {
		p.Send(msg)
	}
}

// OnRecv is called when a message is received from the peer
func (ob *Observer) OnRecv(p mesh.Peer, r io.Reader, t message.Type) error {
	if err := ob.cm.OnRecv(p, r, t); err != nil {
		if err != message.ErrUnknownMessage {
			return err
		} else {
			m, err := ob.mm.ParseMessage(r, t)
			if err != nil {
				return err
			}

			if msg, is := m.(*chain.RequestMessage); is {
				is := msg.Height >= ob.kn.Provider().Height()-10
				if !is {
					if Top, _, err := ob.kn.TopRankInMap(ob.adjustFormulatorMap()); err != nil {
						if err != consensus.ErrInsufficientCandidateCount {
							return nil
						}
					} else {
						is = p.ID() == Top.String()
					}
				}
				if is {
					cd, err := ob.kn.Provider().Data(msg.Height)
					if err != nil {
						return err
					}
					sm := &chain.DataMessage{
						Data: cd,
					}
					if err := p.Send(sm); err != nil {
						return err
					}
				}
			} else {
				ob.Lock()
				err := ob.handleMessage(m)
				ob.Unlock()
				if err != nil {
					//log.Println(ob.kn.Provider().Height(), message.NameOfType(t), err)
					return nil
				}
				ob.ms.BroadcastMessage(m)
			}
		}
	}
	return nil
}

func (ob *Observer) handleMessage(m message.Message) error {
	switch msg := m.(type) {
	case *RoundVoteMessage:
		//log.Println(ob.kn.Provider().Height(), ob.roundState, ob.kn.Provider().Height(), msg.RoundVote.RoundHash, "RoundVoteMessage")
		if ob.round != nil {
			return ErrInvalidRoundState
		}
		if ob.roundState != RoundVoteState {
			return ErrInvalidRoundState
		}
		NextRoundHash := ob.nextRoundHash()
		if !msg.RoundVote.RoundHash.Equal(NextRoundHash) {
			return ErrInvalidVote
		}

		if !msg.RoundVote.ChainCoord.Equal(ob.kn.ChainCoord()) {
			return ErrInvalidVote
		}
		cp := ob.kn.Provider()
		if !msg.RoundVote.LastHash.Equal(cp.LastHash()) {
			return ErrInvalidVote
		}
		if msg.RoundVote.TargetHeight != cp.Height()+1 {
			return ErrInvalidVote
		}

		var emptyAddr common.Address
		if !msg.RoundVote.Formulator.Equal(emptyAddr) {
			Top, err := ob.kn.TopRank(int(msg.RoundVote.TimeoutCount))
			if err != nil {
				return err
			}
			if !msg.RoundVote.Formulator.Equal(Top.Address) {
				return ErrInvalidVote
			}
			if !msg.RoundVote.FormulatorPublicHash.Equal(Top.PublicHash) {
				return ErrInvalidVote
			}
		}

		if pubkey, err := common.RecoverPubkey(msg.RoundVote.Hash(), msg.Signature); err != nil {
			return err
		} else {
			pubhash := common.NewPublicHash(pubkey)
			if _, has := ob.Config.ObserverKeyMap[pubhash]; !has {
				return ErrInvalidVoteSignature
			}
			if old, has := ob.roundVoteMap[pubhash]; has {
				if !old.Formulator.Equal(emptyAddr) || msg.RoundVote.Formulator.Equal(emptyAddr) {
					return ErrAlreadyVoted
				}
			}
			ob.roundVoteMap[pubhash] = msg.RoundVote
		}

		if len(ob.roundVoteMap) >= len(ob.Config.ObserverKeyMap)/2+2 {
			var MinRoundVote *RoundVote
			for _, vt := range ob.roundVoteMap {
				if !vt.Formulator.Equal(emptyAddr) {
					if MinRoundVote == nil || MinRoundVote.TimeoutCount > vt.TimeoutCount {
						MinRoundVote = vt
					}
				}
			}
			if MinRoundVote == nil {
				ob.roundFirstTime = 0
				ob.roundFirstHeight = 0
				return ErrNoFormulatorConnected
			}
			ob.roundState = RoundVoteAckState
			ob.round = NewVoteRound(MinRoundVote.RoundHash)
			log.Println(ob.kn.Provider().Height(), "Change State", RoundVoteAckState)

			if ob.roundFirstTime == 0 {
				ob.roundFirstTime = uint64(time.Now().UnixNano())
				ob.roundFirstHeight = uint32(ob.kn.Provider().Height())
			}

			nm := &RoundVoteAckMessage{
				RoundVoteAck: &RoundVoteAck{
					RoundHash:            MinRoundVote.RoundHash,
					TimeoutCount:         MinRoundVote.TimeoutCount,
					Formulator:           MinRoundVote.Formulator,
					FormulatorPublicHash: MinRoundVote.FormulatorPublicHash,
				},
			}
			if sig, err := ob.Config.Key.Sign(nm.RoundVoteAck.Hash()); err != nil {
				return err
			} else {
				nm.Signature = sig
			}
			if err := ob.handleMessage(nm); err != nil {
				return err
			}
			ob.ms.BroadcastMessage(nm)

			for _, v := range ob.round.RoundVoteAckMessageWaitMap {
				go func(msg message.Message) {
					ob.Lock()
					err := ob.handleMessage(msg)
					ob.Unlock()
					if err != nil {
						//log.Println(ob.kn.Provider().Height(), err)
						return
					}
					ob.ms.BroadcastMessage(msg)
				}(v)
			}
		}
		return nil
	case *RoundVoteAckMessage:
		//log.Println(ob.kn.Provider().Height(), ob.roundState, ob.kn.Provider().Height(), "RoundVoteAckMessage")
		if ob.round == nil {
			return ErrInvalidRoundState
		}
		if !msg.RoundVoteAck.RoundHash.Equal(ob.round.RoundHash) {
			return ErrInvalidVote
		}
		if ob.roundState != RoundVoteAckState {
			if ob.roundState < RoundVoteAckState {
				ob.round.RoundVoteAckMessageWaitMap[msg.RoundVoteAck.Hash()] = msg
			}
			return ErrInvalidRoundState
		}

		Top, err := ob.kn.TopRank(int(msg.RoundVoteAck.TimeoutCount))
		if err != nil {
			return err
		}
		if !msg.RoundVoteAck.Formulator.Equal(Top.Address) {
			return ErrInvalidVote
		}
		if !msg.RoundVoteAck.FormulatorPublicHash.Equal(Top.PublicHash) {
			return ErrInvalidVote
		}

		if pubkey, err := common.RecoverPubkey(msg.RoundVoteAck.Hash(), msg.Signature); err != nil {
			return err
		} else {
			pubhash := common.NewPublicHash(pubkey)
			if _, has := ob.Config.ObserverKeyMap[pubhash]; !has {
				return ErrInvalidVoteSignature
			}
			if _, has := ob.round.RoundVoteAckMap[pubhash]; has {
				return ErrAlreadyVoted
			}
			ob.round.RoundVoteAckMap[pubhash] = msg.RoundVoteAck
		}

		TimeoutCountMap := map[uint32]int{}
		var MinRoundVoteAck *RoundVoteAck
		for _, vt := range ob.round.RoundVoteAckMap {
			Count := TimeoutCountMap[vt.TimeoutCount]
			Count++
			TimeoutCountMap[vt.TimeoutCount] = Count
			if Count >= len(ob.Config.ObserverKeyMap)/2+1 {
				MinRoundVoteAck = vt
				break
			}
		}

		if MinRoundVoteAck != nil {
			ob.roundState = BlockVoteState
			log.Println(ob.kn.Provider().Height(), "Change State", BlockVoteState)
			ob.round.MinRoundVoteAck = MinRoundVoteAck

			cp := ob.kn.Provider()
			nm := &message_def.BlockReqMessage{
				RoundHash:            ob.round.RoundHash,
				PrevHash:             cp.LastHash(),
				TargetHeight:         cp.Height() + 1,
				TimeoutCount:         ob.round.MinRoundVoteAck.TimeoutCount,
				Formulator:           ob.round.MinRoundVoteAck.Formulator,
				FormulatorPublicHash: ob.round.MinRoundVoteAck.FormulatorPublicHash,
			}
			ob.fs.SendTo(ob.round.MinRoundVoteAck.Formulator, nm)

			for _, v := range ob.round.BlockGenMessageWaitMap {
				go func(msg message.Message) {
					ob.Lock()
					err := ob.handleMessage(msg)
					ob.Unlock()
					if err != nil {
						//log.Println(ob.kn.Provider().Height(), err)
						return
					}
					ob.ms.BroadcastMessage(msg)
				}(v)
			}
			for _, v := range ob.round.BlockVoteMessageWaitMap {
				go func(msg message.Message) {
					ob.Lock()
					err := ob.handleMessage(msg)
					ob.Unlock()
					if err != nil {
						//log.Println(ob.kn.Provider().Height(), err)
						return
					}
					ob.ms.BroadcastMessage(msg)
				}(v)
			}
		}
		return nil
	case *BlockVoteMessage:
		//log.Println(ob.kn.Provider().Height(), ob.roundState, ob.kn.Provider().Height(), "BlockVoteMessage")
		if ob.round == nil {
			return ErrInvalidRoundState
		}
		if !msg.BlockVote.RoundHash.Equal(ob.round.RoundHash) {
			return ErrInvalidVote
		}
		if ob.roundState != BlockVoteState {
			if ob.roundState < BlockVoteState {
				ob.round.BlockVoteMessageWaitMap[msg.BlockVote.Hash()] = msg
			}
			return ErrInvalidRoundState
		}
		if ob.round.BlockGenMessage == nil {
			ob.round.BlockVoteMessageWaitMap[msg.BlockVote.Hash()] = msg
			return ErrInvalidRoundState
		}

		cp := ob.kn.Provider()
		if !msg.BlockVote.Header.PrevHash().Equal(cp.LastHash()) {
			return ErrInvalidVote
		}
		if msg.BlockVote.Header.Height() != cp.Height()+1 {
			return ErrInvalidVote
		}
		if !msg.BlockVote.Header.Hash().Equal(ob.round.BlockGenMessage.Block.Header.Hash()) {
			return ErrInvalidVote
		}

		HeaderHash := msg.BlockVote.Header.Hash()
		if pubkey, err := common.RecoverPubkey(HeaderHash, msg.BlockVote.GeneratorSignature); err != nil {
			return err
		} else {
			pubhash := common.NewPublicHash(pubkey)
			if !ob.round.MinRoundVoteAck.FormulatorPublicHash.Equal(pubhash) {
				return ErrInvalidFormulatorSignature
			}
		}
		s := &block.Signed{
			HeaderHash:         HeaderHash,
			GeneratorSignature: msg.BlockVote.GeneratorSignature,
		}
		var ObserverBlockPublicHash common.PublicHash
		if pubkey, err := common.RecoverPubkey(s.Hash(), msg.BlockVote.ObserverSignature); err != nil {
			return err
		} else {
			pubhash := common.NewPublicHash(pubkey)
			if _, has := ob.Config.ObserverKeyMap[pubhash]; !has {
				return ErrInvalidVoteSignature
			}
			ObserverBlockPublicHash = pubhash
		}

		if pubkey, err := common.RecoverPubkey(msg.BlockVote.Hash(), msg.Signature); err != nil {
			return err
		} else {
			pubhash := common.NewPublicHash(pubkey)
			if _, has := ob.Config.ObserverKeyMap[pubhash]; !has {
				return ErrInvalidVoteSignature
			}
			if !ObserverBlockPublicHash.Equal(pubhash) {
				return ErrInvalidVoteSignature
			}
			if _, has := ob.round.BlockVoteMap[pubhash]; has {
				return ErrAlreadyVoted
			}
			ob.round.BlockVoteMap[pubhash] = msg.BlockVote
		}

		if len(ob.round.BlockVoteMap) >= len(ob.Config.ObserverKeyMap)/2+1 {
			sigs := []common.Signature{}
			var AnyBlockVote *BlockVote
			for _, vt := range ob.round.BlockVoteMap {
				sigs = append(sigs, vt.ObserverSignature)
				AnyBlockVote = vt
			}

			cd := &chain.Data{
				Header:     ob.round.BlockGenMessage.Block.Header,
				Body:       ob.round.BlockGenMessage.Block.Body,
				Signatures: append([]common.Signature{ob.round.BlockGenMessage.GeneratorSignature}, sigs...),
			}

			if err := ob.cm.ProcessWithCallback(cd, ob.round.Context, func() {
				ob.isProcessing = true
			}, func() {
				ob.isProcessing = false
			}); err != nil {
				return err
			}
			ob.cm.BroadcastHeader(cd.Header)

			nm := &message_def.BlockObSignMessage{
				TargetHeight: AnyBlockVote.Header.Height(),
				ObserverSigned: &block.ObserverSigned{
					Signed: block.Signed{
						HeaderHash:         AnyBlockVote.Header.Hash(),
						GeneratorSignature: AnyBlockVote.GeneratorSignature,
					},
					ObserverSignatures: sigs,
				},
			}
			ob.fs.SendTo(ob.round.MinRoundVoteAck.Formulator, nm)

			delete(ob.ignoreMap, ob.round.MinRoundVoteAck.Formulator)

			ob.roundState = RoundVoteState
			log.Println(ob.kn.Provider().Height(), "Change State", RoundVoteState)
			ob.roundVoteMap = map[common.PublicHash]*RoundVote{}
			ob.round = nil

			if Top, _, err := ob.kn.TopRankInMap(ob.adjustFormulatorMap()); err != nil {
				if err != consensus.ErrInsufficientCandidateCount {
					return err
				}
			} else {
				ob.fs.SendTo(Top.Address, &chain.StatusMessage{
					Version:  cp.Version(),
					Height:   cp.Height(),
					LastHash: cp.LastHash(),
				})
			}

			if err := ob.sendRoundVote(); err != nil {
				return err
			}
			//log.Println(ob.kn.Provider().Height(), "BlockProgressed", ob.kn.Provider().Height())
		}
		return nil
	case *message_def.BlockGenMessage:
		//log.Println(ob.kn.Provider().Height(), ob.roundState, ob.kn.Provider().Height(), "BlockGenMessage")
		if ob.round == nil {
			return ErrInvalidRoundState
		}
		if !msg.RoundHash.Equal(ob.round.RoundHash) {
			return ErrInvalidVote
		}
		if ob.roundState != BlockVoteState {
			if ob.roundState < BlockVoteState {
				ob.round.BlockGenMessageWaitMap[msg.Block.Header.Hash()] = msg
			}
			return ErrInvalidRoundState
		}
		if ob.round.BlockGenMessage != nil {
			return ErrAlreadyVoted
		}

		if msg.Block.Header.TimeoutCount != ob.round.MinRoundVoteAck.TimeoutCount {
			return ErrInvalidVote
		}
		if !msg.Block.Header.Formulator.Equal(ob.round.MinRoundVoteAck.Formulator) {
			return ErrInvalidVote
		}

		cp := ob.kn.Provider()
		if !msg.Block.Header.PrevHash().Equal(cp.LastHash()) {
			return ErrInvalidVote
		}
		if msg.Block.Header.Height() != cp.Height()+1 {
			return ErrInvalidVote
		}

		ctx, err := ob.kn.Validate(msg.Block, msg.GeneratorSignature)
		if err != nil {
			return err
		}

		nm := &BlockVoteMessage{
			BlockVote: &BlockVote{
				RoundHash:          ob.round.RoundHash,
				Header:             msg.Block.Header,
				GeneratorSignature: msg.GeneratorSignature,
			},
		}
		s := &block.Signed{
			HeaderHash:         msg.Block.Header.Hash(),
			GeneratorSignature: msg.GeneratorSignature,
		}
		if sig, err := ob.Config.Key.Sign(s.Hash()); err != nil {
			return err
		} else {
			nm.BlockVote.ObserverSignature = sig
			ob.round.BlockGenMessage = msg
			ob.round.Context = ctx
		}
		if sig, err := ob.Config.Key.Sign(nm.BlockVote.Hash()); err != nil {
			return err
		} else {
			nm.Signature = sig
		}
		if err := ob.handleMessage(nm); err != nil {
			return err
		}

		PastTime := uint64(time.Now().UnixNano()) - ob.roundFirstTime
		ExpectedTime := uint64(msg.Block.Header.Height()-ob.roundFirstHeight) * uint64(500*time.Millisecond)

		if PastTime < ExpectedTime {
			time.Sleep(time.Duration(ExpectedTime - PastTime))
		}
		ob.ms.BroadcastMessage(nm)
		return nil
	default:
		return message.ErrUnhandledMessage
	}
}

func (ob *Observer) nextRoundHash() hash.Hash256 {
	cp := ob.kn.Provider()
	var buffer bytes.Buffer
	if _, err := ob.kn.ChainCoord().WriteTo(&buffer); err != nil {
		panic(err)
	}
	buffer.WriteString(",")
	if _, err := cp.LastHash().WriteTo(&buffer); err != nil {
		panic(err)
	}
	buffer.WriteString(",")
	if _, err := util.WriteUint32(&buffer, cp.Height()+1); err != nil {
		panic(err)
	}
	return hash.DoubleHash(buffer.Bytes())
}

// OnCreateContext called when a context creation (error prevent using context)
func (ob *Observer) OnCreateContext(kn *kernel.Kernel, ctx *data.Context) error {
	return nil
}

// OnProcessBlock called when processing block to the chain (error prevent processing block)
func (ob *Observer) OnProcessBlock(kn *kernel.Kernel, b *block.Block, s *block.ObserverSigned, ctx *data.Context) error {
	if !ob.isProcessing {
		ob.Lock()
		defer ob.Unlock()

		if ob.round != nil {
			if len(ob.roundVoteMap) > 0 {
				var AnyRoundVote *RoundVote
				for _, vt := range ob.roundVoteMap {
					AnyRoundVote = vt
					break
				}
				if b.Header.Height() >= AnyRoundVote.TargetHeight {
					ob.roundState = RoundVoteState
					log.Println(ob.kn.Provider().Height(), "Change State", RoundVoteState)
					ob.roundVoteMap = map[common.PublicHash]*RoundVote{}
					ob.round = nil
					ob.sendRoundVote()
				}
			}
		}
	}
	return nil
}

// OnPushTransaction called when pushing a transaction to the transaction pool (error prevent push transaction)
func (ob *Observer) OnPushTransaction(kn *kernel.Kernel, tx transaction.Transaction, sigs []common.Signature) error {
	return nil
}

// AfterProcessBlock called when processed block to the chain
func (ob *Observer) AfterProcessBlock(kn *kernel.Kernel, b *block.Block, s *block.ObserverSigned, ctx *data.Context) {
}

func (ob *Observer) adjustFormulatorMap() map[common.Address]bool {
	FormulatorMap := ob.fs.FormulatorMap()
	now := time.Now().UnixNano()
	for addr := range FormulatorMap {
		if now < ob.ignoreMap[addr] {
			delete(FormulatorMap, addr)
		}
	}
	return FormulatorMap
}

func (ob *Observer) sendRoundVote() error {
	cp := ob.kn.Provider()
	nm := &RoundVoteMessage{
		RoundVote: &RoundVote{
			RoundHash:    ob.nextRoundHash(),
			ChainCoord:   ob.kn.ChainCoord(),
			LastHash:     cp.LastHash(),
			TargetHeight: cp.Height() + 1,
		},
	}

	if Top, TimeoutCount, err := ob.kn.TopRankInMap(ob.adjustFormulatorMap()); err != nil {
		if err != consensus.ErrInsufficientCandidateCount {
			return err
		}
	} else {
		nm.RoundVote.TimeoutCount = uint32(TimeoutCount)
		nm.RoundVote.Formulator = Top.Address
		nm.RoundVote.FormulatorPublicHash = Top.PublicHash

		ob.fs.SendTo(Top.Address, &chain.StatusMessage{
			Version:  cp.Version(),
			Height:   cp.Height(),
			LastHash: cp.LastHash(),
		})
	}

	//log.Println(ob.kn.Provider().Height(), "SendRoundVote", nm.RoundVote.Formulator, nm.RoundVote.RoundHash)
	ob.failTimer.Reset(5 * time.Second)

	if sig, err := ob.Config.Key.Sign(nm.RoundVote.Hash()); err != nil {
		return err
	} else {
		nm.Signature = sig
	}
	ob.handleMessage(nm)
	ob.ms.BroadcastMessage(nm)
	return nil
}

func (ob *Observer) messageCreator(r io.Reader, t message.Type) (message.Message, error) {
	switch t {
	case RoundVoteMessageType:
		p := &RoundVoteMessage{
			RoundVote: &RoundVote{
				ChainCoord: &common.Coordinate{},
			},
		}
		if _, err := p.ReadFrom(r); err != nil {
			return nil, err
		}
		return p, nil
	case RoundVoteAckMessageType:
		p := &RoundVoteAckMessage{
			RoundVoteAck: &RoundVoteAck{},
		}
		if _, err := p.ReadFrom(r); err != nil {
			return nil, err
		}
		return p, nil
	case BlockVoteMessageType:
		p := &BlockVoteMessage{
			BlockVote: &BlockVote{
				Header: ob.kn.Provider().CreateHeader(),
			},
		}
		if _, err := p.ReadFrom(r); err != nil {
			return nil, err
		}
		return p, nil
	case message_def.BlockGenMessageType:
		p := &message_def.BlockGenMessage{
			Block: &block.Block{
				Header: ob.kn.Provider().CreateHeader().(*block.Header),
				Body:   ob.kn.Provider().CreateBody().(*block.Body),
			},
			Tran: ob.kn.Transactor(),
		}
		if _, err := p.ReadFrom(r); err != nil {
			return nil, err
		}
		return p, nil
	case chain.RequestMessageType:
		p := &chain.RequestMessage{}
		if _, err := p.ReadFrom(r); err != nil {
			return nil, err
		}
		return p, nil
	default:
		return nil, message.ErrUnknownMessage
	}
}
