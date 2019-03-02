package observer

import (
	"bytes"
	"io"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/data"
	"git.fleta.io/fleta/core/transaction"

	"git.fleta.io/fleta/core/kernel"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/message_def"
	"git.fleta.io/fleta/framework/chain"
	"git.fleta.io/fleta/framework/chain/mesh"
	"git.fleta.io/fleta/framework/message"
)

// Observer supports the chain validation
type Observer struct {
	sync.Mutex
	Config                  *Config
	observerPubHash         common.PublicHash
	round                   *VoteRound
	roundState              int
	roundVoteMap            map[common.PublicHash]*RoundVote
	nextRoundVoteMessageMap map[common.PublicHash]*RoundVoteMessage
	roundFirstTime          uint64
	roundFirstHeight        uint32
	roundFailCount          int
	prevRoundEndTime        int64
	ignoreMap               map[common.Address]int64
	ms                      *ObserverMesh
	cm                      *chain.Manager
	kn                      *kernel.Kernel
	fs                      *FormulatorService
	mm                      *message.Manager
	isRunning               bool
	lastestProcessHeight    uint32
	closeLock               sync.RWMutex
	runEnd                  chan struct{}
	isClose                 bool
}

// NewObserver returns a Observer
func NewObserver(Config *Config, kn *kernel.Kernel) (*Observer, error) {
	ob := &Observer{
		Config:                  Config,
		observerPubHash:         common.NewPublicHash(Config.Key.PublicKey()),
		roundVoteMap:            map[common.PublicHash]*RoundVote{},
		nextRoundVoteMessageMap: map[common.PublicHash]*RoundVoteMessage{},
		roundState:              RoundVoteState,
		ignoreMap:               map[common.Address]int64{},
		cm:                      chain.NewManager(kn),
		kn:                      kn,
		mm:                      message.NewManager(),
		runEnd:                  make(chan struct{}, 1),
	}
	ob.mm.SetCreator(chain.RequestMessageType, ob.messageCreator)
	ob.mm.SetCreator(message_def.BlockGenMessageType, ob.messageCreator)
	ob.mm.SetCreator(RoundVoteMessageType, ob.messageCreator)
	ob.mm.SetCreator(RoundVoteAckMessageType, ob.messageCreator)
	ob.mm.SetCreator(BlockVoteMessageType, ob.messageCreator)
	ob.mm.SetCreator(BlockVoteEndMessageType, ob.messageCreator)
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

	ob.Lock()
	defer ob.Unlock()

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

	timer := time.NewTimer(time.Millisecond)
	for !ob.isClose {
		select {
		case <-timer.C:
			ob.kn.DebugLog("Observer", ob.kn.Provider().Height(), "Current State", ob.roundState, len(ob.adjustFormulatorMap()), ob.fs.PeerCount(), (time.Now().UnixNano()-ob.prevRoundEndTime)/int64(time.Millisecond))

			/*
				ob.fs.Lock()
				frMap := map[string]interface{}{}
				for addr, p := range ob.fs.peerMap {
					frMap[addr.String()] = map[string]interface{}{
						"start_time":  p.startTime,
						"write_total": atomic.LoadUint64(&p.writeTotal),
						"read_total":  atomic.LoadUint64(&p.readTotal),
					}
				}
				ob.fs.Unlock()
				ob.ms.Lock()
				pubhashMap := map[common.PublicHash]uint64{}
				dataMap := map[string]uint64{}
				for pubhash, p := range ob.ms.clientPeerMap {
					pubhashMap[pubhash] = p.startTime
					dataMap[pubhash.String()+":w"] += atomic.LoadUint64(&p.writeTotal)
					dataMap[pubhash.String()+":r"] += atomic.LoadUint64(&p.readTotal)
				}
				for pubhash, p := range ob.ms.serverPeerMap {
					pubhashMap[pubhash] = p.startTime
					dataMap[pubhash.String()+":w"] += atomic.LoadUint64(&p.writeTotal)
					dataMap[pubhash.String()+":r"] += atomic.LoadUint64(&p.readTotal)
				}
				obMap := map[string]interface{}{}
				for pubhash, v := range pubhashMap {
					obMap[pubhash.String()] = map[string]interface{}{
						"start_time":  v,
						"write_total": dataMap[pubhash.String()+":w"],
						"read_total":  dataMap[pubhash.String()+":r"],
					}
				}
				ob.ms.Unlock()
				body, _ := json.Marshal(map[string]interface{}{
					"formulator": frMap,
					"observer":   obMap,
				})
				ob.kn.DebugLog("Observer", string(body))
			*/

			ob.Lock()
			if ob.round != nil {
				if len(ob.roundVoteMap) > 0 {
					var AnyRoundVote *RoundVote
					for _, vt := range ob.roundVoteMap {
						AnyRoundVote = vt
						break
					}
					if atomic.LoadUint32(&ob.lastestProcessHeight) >= AnyRoundVote.TargetHeight {
						ob.roundState = RoundVoteState
						ob.kn.DebugLog("Observer", ob.kn.Provider().Height(), "Change State", RoundVoteState, (time.Now().UnixNano()-ob.prevRoundEndTime)/int64(time.Millisecond))
						ob.roundVoteMap = map[common.PublicHash]*RoundVote{}
						ob.round = nil
						ob.prevRoundEndTime = time.Now().UnixNano()
						for pubhash, msg := range ob.nextRoundVoteMessageMap {
							ob.handleObserverMessage(pubhash, msg)
						}
						ob.nextRoundVoteMessageMap = map[common.PublicHash]*RoundVoteMessage{}
					}
				}
			}
			ob.Unlock()

			if len(ob.adjustFormulatorMap()) > 0 {
				ob.Lock()
				if ob.roundState == RoundVoteState {
					ob.sendRoundVote()
				} else {
					ob.roundFailCount++
					if ob.roundFailCount > 30 {
						ob.kn.DebugLog(ob.kn.Provider().Height(), "Fail State", ob.roundState, len(ob.adjustFormulatorMap()), ob.fs.PeerCount())
						if ob.round != nil && ob.round.MinRoundVoteAck != nil {
							addr := ob.round.MinRoundVoteAck.Formulator
							_, has := ob.ignoreMap[addr]
							if has {
								ob.fs.RemovePeer(addr)
								ob.ignoreMap[addr] = time.Now().UnixNano() + int64(120*time.Second)
							} else {
								ob.ignoreMap[addr] = time.Now().UnixNano() + int64(30*time.Second)
							}
						}
						ob.roundState = RoundVoteState
						ob.kn.DebugLog("Observer", ob.kn.Provider().Height(), "Change State", RoundVoteState, (time.Now().UnixNano()-ob.prevRoundEndTime)/int64(time.Millisecond))
						ob.roundVoteMap = map[common.PublicHash]*RoundVote{}
						ob.round = nil
						ob.roundFirstTime = 0
						ob.roundFirstHeight = 0
						ob.prevRoundEndTime = time.Now().UnixNano()
						ob.sendRoundVote()
						for pubhash, msg := range ob.nextRoundVoteMessageMap {
							ob.handleObserverMessage(pubhash, msg)
						}
						ob.nextRoundVoteMessageMap = map[common.PublicHash]*RoundVoteMessage{}
					}
				}
				ob.Unlock()
			}

			timer.Reset(100 * time.Millisecond)
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
	ob.kn.DebugLog("OnFormulatorConnected", ob.round != nil, ob.round != nil && ob.round.MinRoundVoteAck != nil)

	cp := ob.kn.Provider()
	p.Send(&chain.StatusMessage{
		Version:  cp.Version(),
		Height:   cp.Height(),
		LastHash: cp.LastHash(),
	})

	var msg message.Message
	ob.Lock()
	if ob.round != nil && ob.round.MinRoundVoteAck != nil {
		if ob.round.MinRoundVoteAck.PublicHash.Equal(ob.observerPubHash) {
			if ob.round.MinRoundVoteAck.Formulator.Equal(p.Address()) {
				m := &message_def.BlockReqMessage{
					PrevHash:             cp.LastHash(),
					TargetHeight:         ob.round.TargetHeight,
					TimeoutCount:         ob.round.MinRoundVoteAck.TimeoutCount,
					Formulator:           ob.round.MinRoundVoteAck.Formulator,
					FormulatorPublicHash: ob.round.MinRoundVoteAck.FormulatorPublicHash,
				}
				if ob.kn.Provider().Height() > 0 {
					cd, err := ob.kn.Provider().Data(ob.kn.Provider().Height())
					if err == nil {
						m.PrevData = cd
					}
				}
				msg = m
			}
		}
	}
	ob.Unlock()

	if msg != nil {
		p.Send(msg)
	}
}

type Duplicator struct {
	reader io.Reader
	buffer bytes.Buffer
}

func NewDuplicator(reader io.Reader) *Duplicator {
	return &Duplicator{
		reader: reader,
	}
}

func (r *Duplicator) Read(bs []byte) (int, error) {
	n, err := r.reader.Read(bs)
	if err != nil {
		return n, err
	}
	r.buffer.Write(bs[:n])
	return n, err
}

// OnRecv is called when a message is received from the peer
func (ob *Observer) OnRecv(p mesh.Peer, r io.Reader, t message.Type) error {
	if err := ob.cm.OnRecv(p, r, t); err != nil {
		if err != message.ErrUnknownMessage {
			return err
		} else {
			rd := r
			if t == message_def.BlockGenMessageType {
				dp := NewDuplicator(r)
				util.WriteUint64(&dp.buffer, uint64(t))
				rd = dp
			}
			m, err := ob.mm.ParseMessage(rd, t)
			if err != nil {
				return err
			}

			if msg, is := m.(*chain.RequestMessage); is {
				if fp, is := p.(*FormulatorPeer); is {
					ob.Lock()
					defer ob.Unlock()

					enable := false

					if fp.GuessHeight() < msg.Height {
						CountMap := ob.fs.GuessHeightCountMap()
						if CountMap[ob.kn.Provider().Height()] < 5 {
							enable = true
						} else {
							ranks, err := ob.kn.RanksInMap(ob.adjustFormulatorMap(), 5)
							if err != nil {
								return err
							}
							rankMap := map[string]bool{}
							for _, r := range ranks {
								rankMap[r.Address.String()] = true
							}
							enable = rankMap[p.ID()]
						}
						if enable {
							fp.UpdateGuessHeight(msg.Height)

							//ob.kn.DebugLog("Observer", ob.kn.Provider().Height(), "Send DataMessage", msg.Height, (time.Now().UnixNano()-ob.prevRoundEndTime)/int64(time.Millisecond))

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
					}
				}
			} else if msg, is := m.(*message_def.BlockGenMessage); is {
				ob.Lock()
				ob.handleBlockGenMessage(msg, rd.(*Duplicator).buffer.Bytes())
				ob.Unlock()
			} else if op, is := p.(*Peer); is {
				ob.Lock()
				ob.handleObserverMessage(op.pubhash, m)
				ob.Unlock()
			}
		}
	}
	return nil
}

func (ob *Observer) handleBlockGenMessage(msg *message_def.BlockGenMessage, raw []byte) error {
	ob.kn.DebugLog(ob.kn.Provider().Height(), ob.roundState, ob.kn.Provider().Height(), "BlockGenMessage")
	if ob.round == nil {
		return ErrInvalidRoundState
	}
	if msg.Block.Header.Height() != ob.round.TargetHeight {
		return ErrInvalidVote
	}
	if ob.roundState != BlockVoteState {
		if ob.roundState < BlockVoteState {
			ob.round.BlockGenMessageWait = msg
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

	if ob.round.MinRoundVoteAck.PublicHash.Equal(ob.observerPubHash) && len(raw) > 0 {
		ob.ms.BroadcastRaw(raw)
	}

	ctx, err := ob.kn.Validate(msg.Block, msg.GeneratorSignature)
	if err != nil {
		return err
	}
	ob.round.BlockGenMessage = msg
	ob.round.Context = ctx

	ob.sendBlockVote()

	if ob.round != nil {
		for pubhash, msg := range ob.round.BlockVoteMessageWaitMap {
			ob.handleObserverMessage(pubhash, msg)
			if ob.round == nil {
				return nil
			}
		}
		for pubhash, vt := range ob.round.BlockVoteWaitMap {
			ob.handleBlockVote(pubhash, vt, true)
			if ob.round == nil {
				return nil
			}
		}
	}
	return nil
}

func (ob *Observer) handleObserverMessage(SenderPublicHash common.PublicHash, m message.Message) error {
	if _, has := ob.Config.ObserverKeyMap[SenderPublicHash]; !has {
		return ErrInvalidVoteSignature
	}

	switch msg := m.(type) {
	case *RoundVoteMessage:
		if pubkey, err := common.RecoverPubkey(msg.RoundVote.Hash(), msg.Signature); err != nil {
			return err
		} else {
			if !SenderPublicHash.Equal(common.NewPublicHash(pubkey)) {
				return common.ErrInvalidPublicHash
			}
		}

		//ob.kn.DebugLog(ob.kn.Provider().Height(), ob.roundState, ob.kn.Provider().Height(), msg.RoundVote.TargetHeight, "RoundVoteMessage")
		cp := ob.kn.Provider()
		if msg.RoundVote.TargetHeight != cp.Height()+1 {
			if msg.RoundVote.TargetHeight < cp.Height()+1 {
				if !SenderPublicHash.Equal(ob.observerPubHash) {
					ob.ms.SendTo(SenderPublicHash, &chain.StatusMessage{
						Version:  cp.Version(),
						Height:   cp.Height(),
						LastHash: cp.LastHash(),
					})
				}
			} else if msg.RoundVote.TargetHeight == cp.Height()+2 {
				ob.nextRoundVoteMessageMap[SenderPublicHash] = msg
				return nil
			}
			return ErrInvalidVote
		}
		if ob.roundState != RoundVoteState {
			if ob.roundState > RoundVoteState {
				if !msg.RoundVote.IsReply {
					ob.sendRoundVoteTo(SenderPublicHash)
				}
			}
			return ErrInvalidRoundState
		}
		if ob.round != nil {
			return ErrInvalidRoundState
		}

		if !msg.RoundVote.ChainCoord.Equal(ob.kn.ChainCoord()) {
			return ErrInvalidVote
		}
		if !msg.RoundVote.LastHash.Equal(cp.LastHash()) {
			return ErrInvalidVote
		}

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

		if old, has := ob.roundVoteMap[SenderPublicHash]; has {
			if old.Formulator.Equal(msg.RoundVote.Formulator) {
				return ErrAlreadyVoted
			}
		}
		ob.roundVoteMap[SenderPublicHash] = msg.RoundVote

		if !msg.RoundVote.IsReply {
			ob.sendRoundVoteTo(SenderPublicHash)
		}

		if len(ob.roundVoteMap) >= len(ob.Config.ObserverKeyMap)/2+2 {
			votes := []*voteSortItem{}
			for pubhash, vt := range ob.roundVoteMap {
				votes = append(votes, &voteSortItem{
					PublicHash: pubhash,
					Priority:   uint64(vt.TimeoutCount),
				})
			}
			sort.Sort(voteSorter(votes))
			MinRoundVote := ob.roundVoteMap[votes[0].PublicHash]
			ob.roundState = RoundVoteAckState
			ob.round = NewVoteRound(MinRoundVote.TargetHeight, votes[0].PublicHash)
			ob.kn.DebugLog("Observer", ob.kn.Provider().Height(), "Change State", RoundVoteAckState, (time.Now().UnixNano()-ob.prevRoundEndTime)/int64(time.Millisecond))

			if ob.roundFirstTime == 0 {
				ob.roundFirstTime = uint64(time.Now().UnixNano())
				ob.roundFirstHeight = uint32(ob.kn.Provider().Height())
			}

			ob.sendRoundVoteAck()
		}
		return nil
	case *RoundVoteAckMessage:
		if pubkey, err := common.RecoverPubkey(msg.RoundVoteAck.Hash(), msg.Signature); err != nil {
			return err
		} else {
			if !SenderPublicHash.Equal(common.NewPublicHash(pubkey)) {
				return common.ErrInvalidPublicHash
			}
		}

		//ob.kn.DebugLog(ob.kn.Provider().Height(), ob.roundState, ob.kn.Provider().Height(), "RoundVoteAckMessage")
		if ob.round == nil {
			return ErrInvalidRoundState
		}
		if msg.RoundVoteAck.TargetHeight != ob.round.TargetHeight {
			return ErrInvalidVote
		}
		if ob.roundState != RoundVoteAckState {
			if ob.roundState < RoundVoteAckState {
				ob.round.RoundVoteAckMessageWaitMap[SenderPublicHash] = msg
			} else {
				if !msg.RoundVoteAck.IsReply {
					ob.sendRoundVoteAckTo(SenderPublicHash)
				}
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

		if _, has := ob.round.RoundVoteAckMap[SenderPublicHash]; has {
			return ErrAlreadyVoted
		}
		ob.round.RoundVoteAckMap[SenderPublicHash] = msg.RoundVoteAck

		if !msg.RoundVoteAck.IsReply {
			ob.sendRoundVoteAckTo(SenderPublicHash)
		}

		if len(ob.round.RoundVoteAckMap) >= len(ob.Config.ObserverKeyMap)/2+1 {
			var MinRoundVoteAck *RoundVoteAck
			PublicHashCountMap := map[common.PublicHash]int{}
			TimeoutCountMap := map[uint32]int{}
			for _, vt := range ob.round.RoundVoteAckMap {
				TimeoutCount := TimeoutCountMap[vt.TimeoutCount]
				TimeoutCount++
				TimeoutCountMap[vt.TimeoutCount] = TimeoutCount
				PublicHashCount := PublicHashCountMap[vt.PublicHash]
				PublicHashCount++
				PublicHashCountMap[vt.PublicHash] = PublicHashCount
				if TimeoutCount >= len(ob.Config.ObserverKeyMap)/2+1 && PublicHashCount >= len(ob.Config.ObserverKeyMap)/2+1 {
					MinRoundVoteAck = vt
					break
				}
			}

			if MinRoundVoteAck != nil {
				ob.roundState = BlockVoteState
				ob.kn.DebugLog("Observer", ob.kn.Provider().Height(), "Change State", BlockVoteState, (time.Now().UnixNano()-ob.prevRoundEndTime)/int64(time.Millisecond))
				ob.round.MinRoundVoteAck = MinRoundVoteAck

				if ob.round.MinRoundVoteAck.PublicHash.Equal(ob.observerPubHash) {
					nm := &message_def.BlockReqMessage{
						PrevHash:             ob.kn.Provider().LastHash(),
						TargetHeight:         ob.round.TargetHeight,
						TimeoutCount:         ob.round.MinRoundVoteAck.TimeoutCount,
						Formulator:           ob.round.MinRoundVoteAck.Formulator,
						FormulatorPublicHash: ob.round.MinRoundVoteAck.FormulatorPublicHash,
					}
					if ob.kn.Provider().Height() > 0 {
						cd, err := ob.kn.Provider().Data(ob.kn.Provider().Height())
						if err != nil {
							return err
						}
						nm.PrevData = cd
					}
					ob.fs.SendTo(ob.round.MinRoundVoteAck.Formulator, nm)
					ob.kn.DebugLog("Observer", "Send BlockReqMessage", ob.round.TargetHeight)
				}

				if ob.round.BlockGenMessageWait != nil {
					ob.handleBlockGenMessage(ob.round.BlockGenMessageWait, nil)
				}
			}
		}
		return nil
	case *BlockVoteMessage:
		if pubkey, err := common.RecoverPubkey(msg.BlockVote.Hash(), msg.Signature); err != nil {
			return err
		} else {
			if !SenderPublicHash.Equal(common.NewPublicHash(pubkey)) {
				return common.ErrInvalidPublicHash
			}
		}

		ob.kn.DebugLog(ob.kn.Provider().Height(), ob.roundState, ob.kn.Provider().Height(), "BlockVoteMessage")
		if ob.round == nil {
			return ErrInvalidRoundState
		}
		if msg.BlockVote.Header.Height() != ob.round.TargetHeight {
			return ErrInvalidVote
		}
		if ob.roundState != BlockVoteState {
			if ob.roundState < BlockVoteState {
				ob.round.BlockVoteMessageWaitMap[SenderPublicHash] = msg
			}
			return ErrInvalidRoundState
		}
		if ob.round.BlockGenMessage == nil {
			ob.round.BlockVoteMessageWaitMap[SenderPublicHash] = msg
			return ErrInvalidRoundState
		}

		if err := ob.handleBlockVote(SenderPublicHash, msg.BlockVote, false); err != nil {
			return err
		}
		return nil
	case *BlockVoteEndMessage:
		if pubkey, err := common.RecoverPubkey(msg.BlockVoteEnd.Hash(), msg.Signature); err != nil {
			return err
		} else {
			if !SenderPublicHash.Equal(common.NewPublicHash(pubkey)) {
				return common.ErrInvalidPublicHash
			}
		}

		ob.kn.DebugLog(ob.kn.Provider().Height(), ob.roundState, ob.kn.Provider().Height(), "BlockVoteEndMessage")
		if ob.round == nil {
			return ErrInvalidRoundState
		}
		if msg.BlockVoteEnd.TargetHeight != ob.round.TargetHeight {
			return ErrInvalidVote
		}
		for _, vt := range msg.BlockVoteEnd.BlockVotes {
			s := &block.Signed{
				HeaderHash:         vt.Header.Hash(),
				GeneratorSignature: vt.GeneratorSignature,
			}
			if pubkey, err := common.RecoverPubkey(s.Hash(), vt.ObserverSignature); err != nil {
				return err
			} else {
				pubhash := common.NewPublicHash(pubkey)
				if !pubhash.Equal(ob.observerPubHash) {
					if ob.roundState == BlockVoteState {
						ob.handleBlockVote(pubhash, vt, true)
						if ob.roundState == RoundVoteState {
							return nil
						}
					} else {
						ob.round.BlockVoteWaitMap[pubhash] = vt
					}
				}
			}
		}
		return nil
	default:
		return message.ErrUnhandledMessage
	}
}

func (ob *Observer) handleBlockVote(SenderPublicHash common.PublicHash, vt *BlockVote, IsVoteEnd bool) error {
	//ob.kn.DebugLog(ob.kn.Provider().Height(), ob.roundState, ob.kn.Provider().Height(), "BlockVoteMessage")
	if ob.round == nil {
		return ErrInvalidRoundState
	}
	if vt.Header.Height() != ob.round.TargetHeight {
		return ErrInvalidVote
	}
	if ob.round.BlockGenMessage == nil {
		return ErrInvalidVote
	}

	cp := ob.kn.Provider()
	if !vt.Header.PrevHash().Equal(cp.LastHash()) {
		return ErrInvalidVote
	}
	if vt.Header.Height() != cp.Height()+1 {
		return ErrInvalidVote
	}
	if !vt.Header.Hash().Equal(ob.round.BlockGenMessage.Block.Header.Hash()) {
		return ErrInvalidVote
	}

	HeaderHash := vt.Header.Hash()
	if pubkey, err := common.RecoverPubkey(HeaderHash, vt.GeneratorSignature); err != nil {
		return err
	} else {
		if !ob.round.MinRoundVoteAck.FormulatorPublicHash.Equal(common.NewPublicHash(pubkey)) {
			return ErrInvalidFormulatorSignature
		}
	}
	s := &block.Signed{
		HeaderHash:         HeaderHash,
		GeneratorSignature: vt.GeneratorSignature,
	}
	if pubkey, err := common.RecoverPubkey(s.Hash(), vt.ObserverSignature); err != nil {
		return err
	} else {
		if !SenderPublicHash.Equal(common.NewPublicHash(pubkey)) {
			return ErrInvalidVoteSignature
		}
	}

	if _, has := ob.round.BlockVoteMap[SenderPublicHash]; has {
		return ErrAlreadyVoted
	}
	ob.round.BlockVoteMap[SenderPublicHash] = vt

	if !vt.IsReply && !IsVoteEnd {
		ob.sendBlockVoteTo(SenderPublicHash)
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

		if err := ob.cm.Process(cd, ob.round.Context); err != nil {
			if err != chain.ErrInvalidHeight {
				return err
			}
		} else {
			ob.cm.BroadcastHeader(cd.Header)
		}
		delete(ob.ignoreMap, ob.round.MinRoundVoteAck.Formulator)

		if ob.round.MinRoundVoteAck.PublicHash.Equal(ob.observerPubHash) {
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
			ob.fs.UpdateGuessHeight(ob.round.MinRoundVoteAck.Formulator, nm.TargetHeight)
			ob.kn.DebugLog("Observer", "Send BlockObSignMessage", nm.TargetHeight)
		} else {
			ranks, err := ob.kn.RanksInMap(ob.adjustFormulatorMap(), 5)
			if err == nil {
				for _, v := range ranks {
					ob.fs.SendTo(v.Address, &chain.StatusMessage{
						Version:  cp.Version(),
						Height:   cp.Height(),
						LastHash: cp.LastHash(),
					})
				}
			}
		}

		ob.sendBlockVoteEnd()

		PastTime := uint64(time.Now().UnixNano()) - ob.roundFirstTime
		ExpectedTime := uint64(vt.Header.Height()-ob.roundFirstHeight) * uint64(500*time.Millisecond)

		if PastTime < ExpectedTime {
			time.Sleep(time.Duration(ExpectedTime - PastTime))
			ob.kn.DebugLog("Observer", ob.kn.Provider().Height(), "Sleep", int64(ExpectedTime-PastTime)/int64(time.Millisecond))
		}

		ob.roundState = RoundVoteState
		ob.kn.DebugLog("Observer", ob.kn.Provider().Height(), "Change State", RoundVoteState, (time.Now().UnixNano()-ob.prevRoundEndTime)/int64(time.Millisecond))
		ob.roundVoteMap = map[common.PublicHash]*RoundVote{}
		ob.round = nil
		ob.prevRoundEndTime = time.Now().UnixNano()
		ob.sendRoundVote()
		for pubhash, msg := range ob.nextRoundVoteMessageMap {
			ob.handleObserverMessage(pubhash, msg)
		}
		ob.nextRoundVoteMessageMap = map[common.PublicHash]*RoundVoteMessage{}
	}
	return nil
}

// OnProcessBlock called when processing block to the chain (error prevent processing block)
func (ob *Observer) OnProcessBlock(kn *kernel.Kernel, b *block.Block, s *block.ObserverSigned, ctx *data.Context) error {
	atomic.StoreUint32(&ob.lastestProcessHeight, b.Header.Height())
	return nil
}

// AfterProcessBlock called when processed block to the chain
func (ob *Observer) AfterProcessBlock(kn *kernel.Kernel, b *block.Block, s *block.ObserverSigned, ctx *data.Context) {
}

// OnPushTransaction called when pushing a transaction to the transaction pool (error prevent push transaction)
func (ob *Observer) OnPushTransaction(kn *kernel.Kernel, tx transaction.Transaction, sigs []common.Signature) error {
	return nil
}

// AfterPushTransaction called when pushed a transaction to the transaction pool
func (ob *Observer) AfterPushTransaction(kn *kernel.Kernel, tx transaction.Transaction, sigs []common.Signature) {
}

// DoTransactionBroadcast called when a transaction need to be broadcast
func (ob *Observer) DoTransactionBroadcast(kn *kernel.Kernel, msg *message_def.TransactionMessage) {
}

// DebugLog TEMP
func (ob *Observer) DebugLog(kn *kernel.Kernel, args ...interface{}) {
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
	Top, TimeoutCount, err := ob.kn.TopRankInMap(ob.adjustFormulatorMap())
	if err != nil {
		return err
	}

	cp := ob.kn.Provider()
	nm := &RoundVoteMessage{
		RoundVote: &RoundVote{
			ChainCoord:           ob.kn.ChainCoord(),
			LastHash:             cp.LastHash(),
			TargetHeight:         cp.Height() + 1,
			TimeoutCount:         uint32(TimeoutCount),
			Formulator:           Top.Address,
			FormulatorPublicHash: Top.PublicHash,
			IsReply:              false,
		},
	}

	ob.fs.SendTo(Top.Address, &chain.StatusMessage{
		Version:  cp.Version(),
		Height:   cp.Height(),
		LastHash: cp.LastHash(),
	})

	ob.roundFailCount = 0

	if sig, err := ob.Config.Key.Sign(nm.RoundVote.Hash()); err != nil {
		return err
	} else {
		nm.Signature = sig
	}

	ob.handleObserverMessage(ob.observerPubHash, nm)

	//ob.kn.DebugLog("Observer", "Send RoundVote", nm.RoundVote.Formulator, nm.RoundVote.TargetHeight)
	ob.ms.BroadcastMessage(nm)
	return nil
}

func (ob *Observer) sendRoundVoteTo(TargetPubHash common.PublicHash) error {
	if TargetPubHash.Equal(ob.observerPubHash) {
		return nil
	}

	MyVote, has := ob.roundVoteMap[ob.observerPubHash]
	if !has {
		Top, TimeoutCount, err := ob.kn.TopRankInMap(ob.adjustFormulatorMap())
		if err != nil {
			return err
		}
		cp := ob.kn.Provider()
		MyVote = &RoundVote{
			ChainCoord:           ob.kn.ChainCoord(),
			LastHash:             cp.LastHash(),
			TargetHeight:         cp.Height() + 1,
			TimeoutCount:         uint32(TimeoutCount),
			Formulator:           Top.Address,
			FormulatorPublicHash: Top.PublicHash,
			IsReply:              false,
		}
	}
	nm := &RoundVoteMessage{
		RoundVote: &RoundVote{
			ChainCoord:           MyVote.ChainCoord,
			LastHash:             MyVote.LastHash,
			TargetHeight:         MyVote.TargetHeight,
			TimeoutCount:         MyVote.TimeoutCount,
			Formulator:           MyVote.Formulator,
			FormulatorPublicHash: MyVote.FormulatorPublicHash,
			IsReply:              true,
		},
	}

	if sig, err := ob.Config.Key.Sign(nm.RoundVote.Hash()); err != nil {
		return err
	} else {
		nm.Signature = sig
	}

	//ob.kn.DebugLog("Observer", "Send RoundVote Reply", nm.RoundVote.Formulator, nm.RoundVote.TargetHeight)
	ob.ms.SendTo(TargetPubHash, nm)
	return nil
}

func (ob *Observer) sendRoundVoteAck() error {
	MinRoundVote := ob.roundVoteMap[ob.round.PublicHash]
	nm := &RoundVoteAckMessage{
		RoundVoteAck: &RoundVoteAck{
			TargetHeight:         MinRoundVote.TargetHeight,
			TimeoutCount:         MinRoundVote.TimeoutCount,
			Formulator:           MinRoundVote.Formulator,
			FormulatorPublicHash: MinRoundVote.FormulatorPublicHash,
			PublicHash:           ob.round.PublicHash,
			IsReply:              false,
		},
	}
	if sig, err := ob.Config.Key.Sign(nm.RoundVoteAck.Hash()); err != nil {
		return err
	} else {
		nm.Signature = sig
	}

	if err := ob.handleObserverMessage(ob.observerPubHash, nm); err != nil {
		return err
	}
	//ob.kn.DebugLog("Observer", "Send RoundVoteAck", nm.RoundVoteAck.Formulator, nm.RoundVoteAck.TargetHeight)
	ob.ms.BroadcastMessage(nm)

	if ob.round != nil {
		for pubhash, msg := range ob.round.RoundVoteAckMessageWaitMap {
			ob.handleObserverMessage(pubhash, msg)
		}
	}
	return nil
}

func (ob *Observer) sendRoundVoteAckTo(TargetPubHash common.PublicHash) error {
	if TargetPubHash.Equal(ob.observerPubHash) {
		return nil
	}

	MinRoundVote := ob.roundVoteMap[ob.round.PublicHash]
	nm := &RoundVoteAckMessage{
		RoundVoteAck: &RoundVoteAck{
			TargetHeight:         MinRoundVote.TargetHeight,
			TimeoutCount:         MinRoundVote.TimeoutCount,
			Formulator:           MinRoundVote.Formulator,
			FormulatorPublicHash: MinRoundVote.FormulatorPublicHash,
			PublicHash:           ob.round.PublicHash,
			IsReply:              true,
		},
	}

	if sig, err := ob.Config.Key.Sign(nm.RoundVoteAck.Hash()); err != nil {
		return err
	} else {
		nm.Signature = sig
	}

	//ob.kn.DebugLog("Observer", "Send RoundVoteAck Reply", nm.RoundVoteAck.Formulator, nm.RoundVoteAck.TargetHeight)
	ob.ms.SendTo(TargetPubHash, nm)
	return nil
}

func (ob *Observer) sendBlockVote() error {
	nm := &BlockVoteMessage{
		BlockVote: &BlockVote{
			Header:             ob.round.BlockGenMessage.Block.Header,
			GeneratorSignature: ob.round.BlockGenMessage.GeneratorSignature,
			IsReply:            false,
		},
	}

	s := &block.Signed{
		HeaderHash:         ob.round.BlockGenMessage.Block.Header.Hash(),
		GeneratorSignature: ob.round.BlockGenMessage.GeneratorSignature,
	}
	if sig, err := ob.Config.Key.Sign(s.Hash()); err != nil {
		return err
	} else {
		nm.BlockVote.ObserverSignature = sig
	}
	if sig, err := ob.Config.Key.Sign(nm.BlockVote.Hash()); err != nil {
		return err
	} else {
		nm.Signature = sig
	}

	if err := ob.handleObserverMessage(ob.observerPubHash, nm); err != nil {
		return err
	}

	//ob.kn.DebugLog("Observer", "Send BlockVote", ob.round.BlockGenMessage.Block.Header.Formulator, ob.round.BlockGenMessage.Block.Header.Height)
	ob.ms.BroadcastMessage(nm)
	return nil
}

func (ob *Observer) sendBlockVoteTo(TargetPubHash common.PublicHash) error {
	if TargetPubHash.Equal(ob.observerPubHash) {
		return nil
	}

	nm := &BlockVoteMessage{
		BlockVote: &BlockVote{
			Header:             ob.round.BlockGenMessage.Block.Header,
			GeneratorSignature: ob.round.BlockGenMessage.GeneratorSignature,
			IsReply:            true,
		},
	}

	s := &block.Signed{
		HeaderHash:         ob.round.BlockGenMessage.Block.Header.Hash(),
		GeneratorSignature: ob.round.BlockGenMessage.GeneratorSignature,
	}
	if sig, err := ob.Config.Key.Sign(s.Hash()); err != nil {
		return err
	} else {
		nm.BlockVote.ObserverSignature = sig
	}
	if sig, err := ob.Config.Key.Sign(nm.BlockVote.Hash()); err != nil {
		return err
	} else {
		nm.Signature = sig
	}

	//ob.kn.DebugLog("Observer", "Send BlockVote Reply", ob.round.BlockGenMessage.Block.Header.Formulator, ob.round.BlockGenMessage.Block.Header.Height)
	ob.ms.SendTo(TargetPubHash, nm)
	return nil
}

func (ob *Observer) sendBlockVoteEnd() error {
	votes := []*BlockVote{}
	for _, vt := range ob.round.BlockVoteMap {
		votes = append(votes, vt)
	}
	nm := &BlockVoteEndMessage{
		BlockVoteEnd: &BlockVoteEnd{
			TargetHeight: ob.round.TargetHeight,
			BlockVotes:   votes,
		},
	}
	if sig, err := ob.Config.Key.Sign(nm.BlockVoteEnd.Hash()); err != nil {
		return err
	} else {
		nm.Signature = sig
	}
	//ob.kn.DebugLog("Observer", "Send BlockVoteEnd", ob.round.BlockGenMessage.Block.Header.Formulator, ob.round.BlockGenMessage.Block.Header.Height)
	ob.ms.BroadcastMessage(nm)
	return nil
}

func (ob *Observer) messageCreator(r io.Reader, t message.Type) (message.Message, error) {
	switch t {
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
	case BlockVoteEndMessageType:
		p := &BlockVoteEndMessage{
			BlockVoteEnd: &BlockVoteEnd{
				Provider: ob.kn.Provider(),
			},
		}
		if _, err := p.ReadFrom(r); err != nil {
			return nil, err
		}
		return p, nil
	default:
		return nil, message.ErrUnknownMessage
	}
}
