package observer

import (
	"bytes"
	"io"
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
	prevTimeoutCount uint32
	ms               *ObserverMesh
	cm               *chain.Manager
	kn               *kernel.Kernel
	fs               *FormulatorService
	manager          *message.Manager
	isRunning        bool
	isProcessing     bool
}

// NewObserver returns a Observer
func NewObserver(Config *Config, kn *kernel.Kernel) (*Observer, error) {
	ob := &Observer{
		Config:       Config,
		roundVoteMap: map[common.PublicHash]*RoundVote{},
		roundState:   RoundVoteState,
		cm:           chain.NewManager(kn),
		kn:           kn,
		manager:      message.NewManager(),
	}
	ob.manager.SetCreator(RoundVoteMessageType, ob.messageCreator)
	ob.manager.SetCreator(RoundVoteAckMessageType, ob.messageCreator)
	ob.manager.SetCreator(BlockVoteMessageType, ob.messageCreator)
	ob.manager.SetCreator(RoundFailVoteMessageType, ob.messageCreator)
	ob.manager.SetCreator(message_def.BlockGenMessageType, ob.messageCreator)
	kn.AddEventHandler(ob)

	ob.fs = NewFormulatorService(Config.Key, kn, ob)
	ob.ms = NewObserverMesh(Config.Key, Config.ObserverKeyMap, ob, ob.cm)
	ob.cm.Mesh = ob.ms

	if err := ob.cm.Init(); err != nil {
		return nil, err
	}
	return ob, nil
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

	timer := time.NewTimer(time.Millisecond)
	for {
		select {
		case <-timer.C:
			if len(ob.fs.peerHash) > 0 {
				if ob.roundState == RoundVoteState {
					ob.Lock()
					ob.sendRoundVote()
					ob.Unlock()
				}
			}
			timer.Reset(5 * time.Second)
		}
	}
}

// OnFormulatorConnected is called after a new formulator connected
func (ob *Observer) OnFormulatorConnected(p *FormulatorPeer) {
	ob.Lock()
	defer ob.Unlock()

	//log.Println("OnFormulatorConnected", ob.round != nil, ob.round != nil && ob.round.MinRoundVoteAck != nil)

	if ob.round != nil && ob.round.MinRoundVoteAck != nil {
		//log.Println("OnFormulatorConnected", ob.round.MinRoundVoteAck.Formulator, p.address)
		if ob.round.MinRoundVoteAck.Formulator.Equal(p.address) {
			cp := ob.kn.Provider()
			nm := &message_def.BlockReqMessage{
				RoundHash:            ob.round.RoundHash,
				PrevHash:             cp.PrevHash(),
				TargetHeight:         cp.Height() + 1,
				TimeoutCount:         ob.round.MinRoundVoteAck.TimeoutCount,
				Formulator:           ob.round.MinRoundVoteAck.Formulator,
				FormulatorPublicHash: ob.round.MinRoundVoteAck.FormulatorPublicHash,
			}
			p.Send(nm)
		}
	}
}

// OnRecv is called when a message is received from the peer
func (ob *Observer) OnRecv(p mesh.Peer, r io.Reader, t message.Type) error {
	ob.Lock()
	defer ob.Unlock()

	if err := ob.cm.OnRecv(p, r, t); err != nil {
		if err != message.ErrUnknownMessage {
			return err
		} else {
			m, err := ob.manager.ParseMessage(r, t)
			if err != nil {
				return err
			}
			if err := ob.handleMessage(m); err != nil {
				//log.Println(ob.Config.Key.PublicKey().String(), message.NameOfType(t), err)
				return nil
			}
			if err := ob.ms.BroadcastMessage(m); err != nil {
				//log.Println(ob.Config.Key.PublicKey().String(), message.NameOfType(t), err)
				return err
			}
		}
	}
	return nil
}

func (ob *Observer) handleMessage(m message.Message) error {
	switch msg := m.(type) {
	case *RoundVoteMessage:
		//log.Println(ob.Config.Key.PublicKey().String(), ob.roundState, ob.kn.Provider().Height(), msg.RoundVote.RoundHash, "RoundVoteMessage")
		if ob.round != nil {
			return ErrInvalidRoundState
		}
		if ob.roundState != RoundVoteState {
			return ErrInvalidRoundState
		}
		NextRoundHash := ob.NextRoundHash()
		if !msg.RoundVote.RoundHash.Equal(NextRoundHash) {
			return ErrInvalidVote
		}

		if !msg.RoundVote.ChainCoord.Equal(ob.kn.ChainCoord()) {
			return ErrInvalidVote
		}
		cp := ob.kn.Provider()
		if !msg.RoundVote.PrevHash.Equal(cp.PrevHash()) {
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
				if !old.Formulator.Equal(emptyAddr) {
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
			//log.Println(ob.Config.Key.PublicKey().String(), "Change State", RoundVoteAckState)

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
			if err := ob.ms.BroadcastMessage(nm); err != nil {
				return err
			}

			if ob.roundFirstTime == 0 {
				ob.roundFirstTime = uint64(time.Now().UnixNano())
				ob.roundFirstHeight = uint32(ob.kn.Provider().Height())
			}

			for _, v := range ob.round.RoundVoteAckMessageWaitMap {
				go func(msg message.Message) {
					ob.Lock()
					defer ob.Unlock()

					if err := ob.handleMessage(msg); err != nil {
						//log.Println(ob.Config.Key.PublicKey().String(), err)
						return
					}
					if err := ob.ms.BroadcastMessage(msg); err != nil {
						//log.Println(ob.Config.Key.PublicKey().String(), err)
						return
					}
				}(v)
			}

			RoundHash := MinRoundVote.RoundHash
			time.AfterFunc(2*time.Second, func() {
				ob.Lock()
				defer ob.Unlock()

				if RoundHash.Equal(ob.NextRoundHash()) {
					cp := ob.kn.Provider()
					NextRoundHash := ob.NextRoundHash()
					nm := &RoundFailVoteMessage{
						RoundFailVote: &RoundFailVote{
							RoundHash:    NextRoundHash,
							ChainCoord:   ob.kn.ChainCoord(),
							PrevHash:     cp.PrevHash(),
							TargetHeight: cp.Height() + 1,
						},
					}
					if sig, err := ob.Config.Key.Sign(nm.RoundFailVote.Hash()); err != nil {
						//log.Println(err)
						return
					} else {
						nm.Signature = sig
					}
					ob.handleMessage(nm)
					if err := ob.ms.BroadcastMessage(nm); err != nil {
						//log.Println(err)
						return
					}

				}
			})
		}
		return nil
	case *RoundVoteAckMessage:
		//log.Println(ob.Config.Key.PublicKey().String(), ob.roundState, ob.kn.Provider().Height(), "RoundVoteAckMessage")
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
			//log.Println(ob.Config.Key.PublicKey().String(), "Change State", BlockVoteState)
			ob.round.MinRoundVoteAck = MinRoundVoteAck

			cp := ob.kn.Provider()
			nm := &message_def.BlockReqMessage{
				RoundHash:            ob.round.RoundHash,
				PrevHash:             cp.PrevHash(),
				TargetHeight:         cp.Height() + 1,
				TimeoutCount:         ob.round.MinRoundVoteAck.TimeoutCount,
				Formulator:           ob.round.MinRoundVoteAck.Formulator,
				FormulatorPublicHash: ob.round.MinRoundVoteAck.FormulatorPublicHash,
			}
			ob.fs.SendTo(ob.round.MinRoundVoteAck.Formulator, nm)

			for _, v := range ob.round.BlockGenMessageWaitMap {
				go func(msg message.Message) {
					ob.Lock()
					defer ob.Unlock()

					if err := ob.handleMessage(msg); err != nil {
						//log.Println(ob.Config.Key.PublicKey().String(), err)
						return
					}
					if err := ob.ms.BroadcastMessage(msg); err != nil {
						//log.Println(ob.Config.Key.PublicKey().String(), err)
						return
					}
				}(v)
			}
			for _, v := range ob.round.BlockVoteMessageWaitMap {
				go func(msg message.Message) {
					ob.Lock()
					defer ob.Unlock()

					if err := ob.handleMessage(msg); err != nil {
						//log.Println(ob.Config.Key.PublicKey().String(), err)
						return
					}
					if err := ob.ms.BroadcastMessage(msg); err != nil {
						//log.Println(ob.Config.Key.PublicKey().String(), err)
						return
					}
				}(v)
			}
		}
		return nil
	case *BlockVoteMessage:
		//log.Println(ob.Config.Key.PublicKey().String(), ob.roundState, ob.kn.Provider().Height(), "BlockVoteMessage")
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
		if !msg.BlockVote.Header.PrevHash().Equal(cp.PrevHash()) {
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

			ob.isProcessing = true
			if err := ob.cm.Process(cd, ob.round.Context); err != nil {
				return err
			}
			ob.isProcessing = false
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

			ob.roundState = RoundVoteState
			//log.Println(ob.Config.Key.PublicKey().String(), "Change State", RoundVoteState)
			ob.roundVoteMap = map[common.PublicHash]*RoundVote{}
			ob.round = nil
			ob.prevTimeoutCount = 0

			if Top, _, err := ob.kn.TopRankInMap(int(ob.prevTimeoutCount), ob.fs.FormulatorMap()); err != nil {
				if err != consensus.ErrInsufficientCandidateCount {
					return err
				}
			} else {
				ob.fs.SendTo(Top.Address, &chain.DataMessage{
					Data: cd,
				})
			}

			if err := ob.sendRoundVote(); err != nil {
				return err
			}

			//log.Println(ob.Config.Key.PublicKey().String(), "BlockProgressed", ob.kn.Provider().Height())
		}
		return nil
	case *RoundFailVoteMessage:
		//log.Println(ob.Config.Key.PublicKey().String(), ob.roundState, ob.kn.Provider().Height(), "RoundFailVoteMessage")
		if ob.round == nil {
			return ErrInvalidVote
		}
		NextRoundHash := ob.NextRoundHash()
		if !msg.RoundFailVote.RoundHash.Equal(NextRoundHash) {
			return ErrInvalidVote
		}

		if !msg.RoundFailVote.ChainCoord.Equal(ob.kn.ChainCoord()) {
			return ErrInvalidVote
		}
		cp := ob.kn.Provider()
		if !msg.RoundFailVote.PrevHash.Equal(cp.PrevHash()) {
			return ErrInvalidVote
		}
		if msg.RoundFailVote.TargetHeight != cp.Height()+1 {
			return ErrInvalidVote
		}

		if pubkey, err := common.RecoverPubkey(msg.RoundFailVote.Hash(), msg.Signature); err != nil {
			return err
		} else {
			pubhash := common.NewPublicHash(pubkey)
			if _, has := ob.Config.ObserverKeyMap[pubhash]; !has {
				return ErrInvalidVoteSignature
			}
			if _, has := ob.round.RoundFailVoteMap[pubhash]; has {
				return ErrAlreadyVoted
			}
			ob.round.RoundFailVoteMap[pubhash] = msg.RoundFailVote
		}

		if len(ob.round.RoundFailVoteMap) >= len(ob.Config.ObserverKeyMap)/2+1 {
			if ob.round.MinRoundVoteAck != nil {
				ob.prevTimeoutCount = ob.round.MinRoundVoteAck.TimeoutCount + 1
			} else {
				ob.prevTimeoutCount++
			}
			ob.roundState = RoundVoteState
			//log.Println(ob.Config.Key.PublicKey().String(), "Change State", RoundVoteState)
			ob.roundVoteMap = map[common.PublicHash]*RoundVote{}
			ob.round = nil
			ob.roundFirstTime = 0
			ob.roundFirstHeight = 0

			if err := ob.sendRoundVote(); err != nil {
				return err
			}
		}
		return nil
	case *message_def.BlockGenMessage:
		//log.Println(ob.Config.Key.PublicKey().String(), ob.roundState, ob.kn.Provider().Height(), "BlockGenMessage")
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
		if !msg.Block.Header.PrevHash().Equal(cp.PrevHash()) {
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

		if err := ob.ms.BroadcastMessage(nm); err != nil {
			return err
		}

		return nil
	default:
		return message.ErrUnhandledMessage
	}
}

// NextRoundHash provides next round hash value
func (ob *Observer) NextRoundHash() hash.Hash256 {
	cp := ob.kn.Provider()
	var buffer bytes.Buffer
	if _, err := ob.kn.ChainCoord().WriteTo(&buffer); err != nil {
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

// OnCreateContext called when a context creation (error prevent using context)
func (ob *Observer) OnCreateContext(kn *kernel.Kernel, ctx *data.Context) error {
	return nil
}

// OnProcessBlock called when processing block to the chain (error prevent processing block)
func (ob *Observer) OnProcessBlock(kn *kernel.Kernel, b *block.Block, s *block.ObserverSigned, ctx *data.Context) error {
	if !ob.isProcessing {
		if ob.round != nil {
			if len(ob.roundVoteMap) > 0 {
				var AnyRoundVote *RoundVote
				for _, vt := range ob.roundVoteMap {
					AnyRoundVote = vt
					break
				}
				if b.Header.Height() >= AnyRoundVote.TargetHeight {
					ob.roundState = RoundVoteState
					//log.Println(ob.Config.Key.PublicKey().String(), "Change State", RoundVoteState)
					ob.roundVoteMap = map[common.PublicHash]*RoundVote{}
					ob.round = nil
					ob.prevTimeoutCount = 0
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

func (ob *Observer) sendRoundVote() error {
	cp := ob.kn.Provider()
	NextRoundHash := ob.NextRoundHash()
	nm := &RoundVoteMessage{
		RoundVote: &RoundVote{
			RoundHash:    NextRoundHash,
			ChainCoord:   ob.kn.ChainCoord(),
			PrevHash:     cp.PrevHash(),
			TargetHeight: cp.Height() + 1,
		},
	}

	if Top, TimeoutCount, err := ob.kn.TopRankInMap(int(ob.prevTimeoutCount), ob.fs.FormulatorMap()); err != nil {
		if err != consensus.ErrInsufficientCandidateCount {
			return err
		}
	} else {
		nm.RoundVote.TimeoutCount = uint32(TimeoutCount)
		nm.RoundVote.Formulator = Top.Address
		nm.RoundVote.FormulatorPublicHash = Top.PublicHash
	}

	if sig, err := ob.Config.Key.Sign(nm.RoundVote.Hash()); err != nil {
		return err
	} else {
		nm.Signature = sig
	}
	ob.handleMessage(nm)
	if err := ob.ms.BroadcastMessage(nm); err != nil {
		return err
	}

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
	case RoundFailVoteMessageType:
		p := &RoundFailVoteMessage{
			RoundFailVote: &RoundFailVote{
				ChainCoord: &common.Coordinate{},
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
	default:
		return nil, message.ErrUnknownMessage
	}
}
