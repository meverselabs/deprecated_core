package observer

import (
	"io"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fletaio/common/util"
	"github.com/fletaio/core/data"
	"github.com/fletaio/core/transaction"

	"github.com/fletaio/core/kernel"

	"github.com/fletaio/common"
	"github.com/fletaio/core/block"
	"github.com/fletaio/core/message_def"
	"github.com/fletaio/framework/chain"
	"github.com/fletaio/framework/chain/mesh"
	"github.com/fletaio/framework/message"
)

// Observer supports the chain validation
type Observer struct {
	sync.Mutex
	Config               *Config
	observerPubHash      common.PublicHash
	round                *VoteRound
	roundState           int
	roundVoteMessageMap  map[common.PublicHash]*RoundVoteMessage
	roundFirstTime       uint64
	roundFirstHeight     uint32
	roundFailCount       int
	prevRoundEndTime     int64
	ignoreMap            map[common.Address]int64
	ms                   *ObserverMesh
	cm                   *chain.Manager
	kn                   *kernel.Kernel
	fs                   *FormulatorService
	mm                   *message.Manager
	isRunning            bool
	lastestProcessHeight uint32
	closeLock            sync.RWMutex
	runEnd               chan struct{}
	isClose              bool
}

// NewObserver returns a Observer
func NewObserver(Config *Config, kn *kernel.Kernel) (*Observer, error) {
	ob := &Observer{
		Config:              Config,
		observerPubHash:     common.NewPublicHash(Config.Key.PublicKey()),
		roundVoteMessageMap: map[common.PublicHash]*RoundVoteMessage{},
		roundState:          RoundVoteState,
		ignoreMap:           map[common.Address]int64{},
		cm:                  chain.NewManager(kn),
		kn:                  kn,
		mm:                  message.NewManager(),
		runEnd:              make(chan struct{}, 1),
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
				if len(ob.roundVoteMessageMap) > 0 {
					var AnyRoundVote *RoundVote
					for _, msg := range ob.roundVoteMessageMap {
						AnyRoundVote = msg.RoundVote
						break
					}
					lastestProcessHeight := atomic.LoadUint32(&ob.lastestProcessHeight)
					if lastestProcessHeight >= AnyRoundVote.VoteTargetHeight {
						if ob.round != nil && lastestProcessHeight < ob.round.BlockRounds[len(ob.round.BlockRounds)-1].TargetHeight {
							for _, br := range ob.round.BlockRounds {
								if br.TargetHeight <= lastestProcessHeight {
									ob.round.RemoveBlockRound(br)
								}
							}
							if ob.round.BlockRoundCount() > 0 {
								br := ob.round.BlockRounds[0]
								if br.BlockGenMessageWait != nil && br.BlockGenMessage != nil {
									ob.handleBlockGenMessage(br.BlockGenMessageWait, nil)
								}
							}
						} else {
							ob.roundState = RoundVoteState
							ob.kn.DebugLog("Observer", ob.kn.Provider().Height(), "Change State", RoundVoteState, (time.Now().UnixNano()-ob.prevRoundEndTime)/int64(time.Millisecond))
							ob.roundVoteMessageMap = map[common.PublicHash]*RoundVoteMessage{}
							ob.round = nil
							ob.prevRoundEndTime = time.Now().UnixNano()
						}
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
					if ob.roundFailCount > 20 {
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
						ob.roundVoteMessageMap = map[common.PublicHash]*RoundVoteMessage{}
						ob.round = nil
						ob.roundFirstTime = 0
						ob.roundFirstHeight = 0
						ob.prevRoundEndTime = time.Now().UnixNano()
						ob.sendRoundVote()
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
	//ob.kn.DebugLog("OnFormulatorConnected", ob.round != nil, ob.round != nil && ob.round.MinRoundVoteAck != nil)

	cp := ob.kn.Provider()
	p.Send(&chain.StatusMessage{
		Version:  cp.Version(),
		Height:   cp.Height(),
		LastHash: cp.LastHash(),
	})
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
						if CountMap[ob.kn.Provider().Height()] < 3 {
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
	//ob.kn.DebugLog("Observer", ob.kn.Provider().Height(), "BlockGenMessage", msg.Block.Header.Height())
	if ob.round == nil {
		return ErrInvalidRoundState
	}
	if msg.Block.Header.Height() < ob.round.VoteTargetHeight {
		return ErrInvalidVote
	}
	if msg.Block.Header.Height() >= ob.round.VoteTargetHeight+ob.kn.Config.MaxBlocksPerFormulator {
		return ErrInvalidVote
	}
	if ob.roundState != BlockVoteState {
		if ob.roundState < BlockVoteState {
			for _, br := range ob.round.BlockRounds {
				if br.TargetHeight == msg.Block.Header.Height() {
					br.BlockGenMessageWait = msg
					break
				}
			}
		}
		return ErrInvalidRoundState
	}
	if !msg.Block.Header.Formulator.Equal(ob.round.MinRoundVoteAck.Formulator) {
		return ErrInvalidVote
	}

	br := ob.round.BlockRounds[0]
	if msg.Block.Header.Height() < br.TargetHeight {
		ob.sendBlockVoteEnd(msg.Block.Header.Height())
		return ErrInvalidRoundState
	}
	if ob.round.MinRoundVoteAck.PublicHash.Equal(ob.observerPubHash) && len(raw) > 0 {
		ob.ms.BroadcastRaw(raw)
	}
	if msg.Block.Header.Height() > br.TargetHeight {
		for _, br := range ob.round.BlockRounds {
			if br.TargetHeight == msg.Block.Header.Height() {
				br.BlockGenMessageWait = msg
				break
			}
		}
		return ErrInvalidVote
	}
	if br.BlockGenMessage != nil {
		return ErrAlreadyVoted
	}

	if msg.Block.Header.Height() == ob.round.VoteTargetHeight {
		if msg.Block.Header.TimeoutCount != ob.round.MinRoundVoteAck.TimeoutCount {
			return ErrInvalidVote
		}
	} else {
		if msg.Block.Header.TimeoutCount != 0 {
			return ErrInvalidVote
		}
	}

	Now := uint64(time.Now().UnixNano())
	if msg.Block.Header.Timestamp() > Now+uint64(1000*time.Millisecond) {
		return ErrInvalidVote
	}

	cp := ob.kn.Provider()
	if !msg.Block.Header.PrevHash().Equal(cp.LastHash()) {
		return ErrInvalidVote
	}
	Height := cp.Height()
	if msg.Block.Header.Height() != Height+1 {
		return ErrInvalidVote
	}
	if Height > 0 {
		LastHeader, err := cp.Header(Height)
		if err != nil {
			return err
		}
		if msg.Block.Header.Timestamp() < LastHeader.Timestamp() {
			return ErrInvalidVote
		}
	}

	ctx, err := ob.kn.Validate(msg.Block, msg.GeneratorSignature)
	if err != nil {
		return err
	}
	br.BlockGenMessage = msg
	br.Context = ctx

	ob.sendBlockVote(br)

	if ob.round != nil {
		for pubhash, msg := range br.BlockVoteMessageWaitMap {
			ob.handleObserverMessage(pubhash, msg)
			if ob.round == nil {
				return nil
			}
		}
		for pubhash, vt := range br.BlockVoteWaitMap {
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

		//ob.kn.DebugLog("Observer", ob.roundState, ob.kn.Provider().Height(), msg.RoundVote.VoteTargetHeight, "RoundVoteMessage")
		cp := ob.kn.Provider()
		if ob.roundState != RoundVoteState {
			if !msg.RoundVote.IsReply {
				switch ob.roundState {
				case RoundVoteAckState:
					ob.sendRoundVoteTo(SenderPublicHash)
				case BlockVoteState:
					for _, msg := range ob.roundVoteMessageMap {
						ob.ms.SendTo(SenderPublicHash, msg)
					}
				}
			}
			return ErrInvalidRoundState
		}
		if msg.RoundVote.VoteTargetHeight != cp.Height()+1 {
			if msg.RoundVote.VoteTargetHeight < cp.Height()+1 {
				if !SenderPublicHash.Equal(ob.observerPubHash) {
					ob.ms.SendTo(SenderPublicHash, &chain.StatusMessage{
						Version:  cp.Version(),
						Height:   cp.Height(),
						LastHash: cp.LastHash(),
					})
				}
			}
			return ErrInvalidVote
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

		if old, has := ob.roundVoteMessageMap[SenderPublicHash]; has {
			if msg.RoundVote.Formulator.Equal(old.RoundVote.Formulator) {
				return ErrAlreadyVoted
			}
		}
		ob.roundVoteMessageMap[SenderPublicHash] = msg

		if !msg.RoundVote.IsReply {
			ob.sendRoundVoteTo(SenderPublicHash)
		}

		if len(ob.roundVoteMessageMap) >= len(ob.Config.ObserverKeyMap)/2+2 {
			votes := []*voteSortItem{}
			for pubhash, msg := range ob.roundVoteMessageMap {
				votes = append(votes, &voteSortItem{
					PublicHash: pubhash,
					Priority:   uint64(msg.RoundVote.TimeoutCount),
				})
			}
			sort.Sort(voteSorter(votes))
			MinRoundVote := ob.roundVoteMessageMap[votes[0].PublicHash].RoundVote
			ob.roundState = RoundVoteAckState
			ob.round = NewVoteRound(MinRoundVote.VoteTargetHeight, votes[0].PublicHash, ob.kn.Config.MaxBlocksPerFormulator)
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

		//ob.kn.DebugLog("Observer", ob.roundState, ob.kn.Provider().Height(), "RoundVoteAckMessage")
		if ob.round == nil {
			return ErrInvalidRoundState
		}
		if msg.RoundVoteAck.VoteTargetHeight != ob.round.VoteTargetHeight {
			return ErrInvalidVote
		}
		if ob.roundState != RoundVoteAckState {
			if ob.roundState < RoundVoteAckState {
				ob.round.RoundVoteAckMessageWaitMap[SenderPublicHash] = msg
			} else {
				if !msg.RoundVoteAck.IsReply {
					for _, msg := range ob.round.RoundVoteAckMessageMap {
						ob.ms.SendTo(SenderPublicHash, msg)
					}
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

		if _, has := ob.round.RoundVoteAckMessageMap[SenderPublicHash]; has {
			return ErrAlreadyVoted
		}
		ob.round.RoundVoteAckMessageMap[SenderPublicHash] = msg

		if !msg.RoundVoteAck.IsReply {
			ob.sendRoundVoteAckTo(SenderPublicHash)
		}

		if len(ob.round.RoundVoteAckMessageMap) >= len(ob.Config.ObserverKeyMap)/2+1 {
			var MinRoundVoteAck *RoundVoteAck
			PublicHashCountMap := map[common.PublicHash]int{}
			TimeoutCountMap := map[uint32]int{}
			for _, msg := range ob.round.RoundVoteAckMessageMap {
				vt := msg.RoundVoteAck
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
						TargetHeight:         ob.round.VoteTargetHeight,
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
					//ob.kn.DebugLog("Observer", "Send BlockReqMessage", ob.round.VoteTargetHeight)
				}

				br := ob.round.BlockRounds[0]
				if br.BlockGenMessageWait != nil {
					ob.handleBlockGenMessage(br.BlockGenMessageWait, nil)
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

		//ob.kn.DebugLog("Observer", ob.roundState, ob.kn.Provider().Height(), "BlockVoteMessage")
		if ob.round == nil {
			return ErrInvalidRoundState
		}
		if msg.BlockVote.VoteTargetHeight != ob.round.VoteTargetHeight {
			return ErrInvalidVote
		}
		if ob.roundState != BlockVoteState {
			if ob.roundState < BlockVoteState {
				for _, br := range ob.round.BlockRounds {
					if br.TargetHeight == msg.BlockVote.Header.Height() {
						br.BlockVoteMessageWaitMap[SenderPublicHash] = msg
						break
					}
				}
			}
			return ErrInvalidRoundState
		}
		br := ob.round.BlockRounds[0]
		if msg.BlockVote.Header.Height() < br.TargetHeight {
			if !msg.BlockVote.IsReply {
				ob.sendBlockVoteEnd(msg.BlockVote.Header.Height())
			}
			return ErrInvalidVote
		} else if msg.BlockVote.Header.Height() > br.TargetHeight {
			for _, br := range ob.round.BlockRounds {
				if br.TargetHeight == msg.BlockVote.Header.Height() {
					br.BlockVoteMessageWaitMap[SenderPublicHash] = msg
					break
				}
			}
			return ErrInvalidRoundState
		}
		if br.BlockGenMessage == nil {
			br.BlockVoteMessageWaitMap[SenderPublicHash] = msg
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

		//ob.kn.DebugLog(ob.kn.Provider().Height(), ob.roundState, ob.kn.Provider().Height(), "BlockVoteEndMessage")
		if ob.round == nil {
			return ErrInvalidRoundState
		}
		if msg.BlockVoteEnd.VoteTargetHeight != ob.round.VoteTargetHeight {
			return ErrInvalidVote
		}
		if msg.BlockVoteEnd.TargetHeight < ob.round.BlockRounds[0].TargetHeight {
			return ErrInvalidVote
		}
		for i, br := range ob.round.BlockRounds {
			if br.TargetHeight == msg.BlockVoteEnd.TargetHeight {
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
							if i == 0 && ob.roundState == BlockVoteState && br.BlockGenMessage != nil {
								ob.handleBlockVote(pubhash, vt, true)
								if ob.round == nil {
									return nil
								}
							} else {
								br.BlockVoteWaitMap[pubhash] = vt
							}
						}
					}
				}
				break
			}
		}
		return nil
	default:
		return message.ErrUnhandledMessage
	}
}

func (ob *Observer) handleBlockVote(SenderPublicHash common.PublicHash, vt *BlockVote, IsVoteEnd bool) error {
	if ob.round == nil {
		return ErrInvalidRoundState
	}
	br := ob.round.BlockRounds[0]
	if br.BlockGenMessage == nil {
		return ErrInvalidVote
	}
	if vt.Header.Height() != br.TargetHeight {
		return ErrInvalidVote
	}

	cp := ob.kn.Provider()
	if !vt.Header.PrevHash().Equal(cp.LastHash()) {
		return ErrInvalidVote
	}
	if vt.Header.Height() != cp.Height()+1 {
		return ErrInvalidVote
	}
	if !vt.Header.Hash().Equal(br.BlockGenMessage.Block.Header.Hash()) {
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

	if _, has := br.BlockVoteMap[SenderPublicHash]; has {
		return ErrAlreadyVoted
	}
	br.BlockVoteMap[SenderPublicHash] = vt

	if !vt.IsReply && !IsVoteEnd {
		ob.sendBlockVoteTo(br, SenderPublicHash)
	}

	if len(br.BlockVoteMap) >= len(ob.Config.ObserverKeyMap)/2+1 {
		sigs := []common.Signature{}
		for _, vt := range br.BlockVoteMap {
			sigs = append(sigs, vt.ObserverSignature)
		}

		cd := &chain.Data{
			Header:     br.BlockGenMessage.Block.Header,
			Body:       br.BlockGenMessage.Block.Body,
			Signatures: append([]common.Signature{br.BlockGenMessage.GeneratorSignature}, sigs...),
		}

		if err := ob.cm.Process(cd, br.Context); err != nil {
			if err != chain.ErrInvalidHeight {
				return err
			}
		} else {
			ob.cm.BroadcastHeader(cd.Header)
		}
		delete(ob.ignoreMap, ob.round.MinRoundVoteAck.Formulator)

		if ob.round.MinRoundVoteAck.PublicHash.Equal(ob.observerPubHash) {
			nm := &message_def.BlockObSignMessage{
				TargetHeight: vt.Header.Height(),
				ObserverSigned: &block.ObserverSigned{
					Signed: block.Signed{
						HeaderHash:         vt.Header.Hash(),
						GeneratorSignature: vt.GeneratorSignature,
					},
					ObserverSignatures: sigs,
				},
			}
			ob.fs.SendTo(ob.round.MinRoundVoteAck.Formulator, nm)
			ob.fs.UpdateGuessHeight(ob.round.MinRoundVoteAck.Formulator, nm.TargetHeight)
			//ob.kn.DebugLog("Observer", "Send BlockObSignMessage", nm.TargetHeight)
		} else {
			ranks, err := ob.kn.RanksInMap(ob.adjustFormulatorMap(), 5)
			if err == nil {
				for _, v := range ranks {
					if !v.Address.Equal(ob.round.MinRoundVoteAck.Formulator) {
						ob.fs.SendTo(v.Address, &chain.StatusMessage{
							Version:  cp.Version(),
							Height:   cp.Height(),
							LastHash: cp.LastHash(),
						})
					}
				}
			}
		}

		ob.round.CloseBlockRound()
		ob.sendBlockVoteEnd(vt.Header.Height())

		PastTime := uint64(time.Now().UnixNano()) - ob.roundFirstTime
		ExpectedTime := uint64(vt.Header.Height()-ob.roundFirstHeight) * uint64(500*time.Millisecond)

		if PastTime < ExpectedTime {
			diff := time.Duration(ExpectedTime - PastTime)
			if diff > 500*time.Millisecond {
				diff = 500 * time.Millisecond
			}
			time.Sleep(diff)
			//ob.kn.DebugLog("Observer", ob.kn.Provider().Height(), "Sleep", int64(ExpectedTime-PastTime)/int64(time.Millisecond))
		}

		if ob.round.BlockRoundCount() > 0 {
			ob.roundFailCount = 0
			br := ob.round.BlockRounds[0]
			if br.BlockGenMessageWait != nil {
				ob.handleBlockGenMessage(br.BlockGenMessageWait, nil)
			}
		} else {
			ob.roundState = RoundVoteState
			ob.kn.DebugLog("Observer", ob.kn.Provider().Height(), "Change State", RoundVoteState, (time.Now().UnixNano()-ob.prevRoundEndTime)/int64(time.Millisecond))
			ob.roundVoteMessageMap = map[common.PublicHash]*RoundVoteMessage{}
			ob.round = nil
			ob.prevRoundEndTime = time.Now().UnixNano()
			ob.sendRoundVote()
		}
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
			VoteTargetHeight:     cp.Height() + 1,
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

	//ob.kn.DebugLog("Observer", "Send RoundVote", nm.RoundVote.Formulator, nm.RoundVote.VoteTargetHeight)
	ob.ms.BroadcastMessage(nm)
	return nil
}

func (ob *Observer) sendRoundVoteTo(TargetPubHash common.PublicHash) error {
	if TargetPubHash.Equal(ob.observerPubHash) {
		return nil
	}

	MyMsg, has := ob.roundVoteMessageMap[ob.observerPubHash]
	if !has {
		return nil
	}
	nm := &RoundVoteMessage{
		RoundVote: &RoundVote{
			ChainCoord:           MyMsg.RoundVote.ChainCoord,
			LastHash:             MyMsg.RoundVote.LastHash,
			VoteTargetHeight:     MyMsg.RoundVote.VoteTargetHeight,
			TimeoutCount:         MyMsg.RoundVote.TimeoutCount,
			Formulator:           MyMsg.RoundVote.Formulator,
			FormulatorPublicHash: MyMsg.RoundVote.FormulatorPublicHash,
			IsReply:              true,
		},
	}

	if sig, err := ob.Config.Key.Sign(nm.RoundVote.Hash()); err != nil {
		return err
	} else {
		nm.Signature = sig
	}

	//ob.kn.DebugLog("Observer", "Send RoundVote Reply", nm.RoundVote.Formulator, nm.RoundVote.VoteTargetHeight)
	ob.ms.SendTo(TargetPubHash, nm)
	return nil
}

func (ob *Observer) sendRoundVoteAck() error {
	MinRoundVote := ob.roundVoteMessageMap[ob.round.PublicHash].RoundVote
	nm := &RoundVoteAckMessage{
		RoundVoteAck: &RoundVoteAck{
			VoteTargetHeight:     MinRoundVote.VoteTargetHeight,
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
	//ob.kn.DebugLog("Observer", "Send RoundVoteAck", nm.RoundVoteAck.Formulator, nm.RoundVoteAck.VoteTargetHeight)
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

	MinRoundVote := ob.roundVoteMessageMap[ob.round.PublicHash].RoundVote
	nm := &RoundVoteAckMessage{
		RoundVoteAck: &RoundVoteAck{
			VoteTargetHeight:     MinRoundVote.VoteTargetHeight,
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

	//ob.kn.DebugLog("Observer", "Send RoundVoteAck Reply", nm.RoundVoteAck.Formulator, nm.RoundVoteAck.VoteTargetHeight)
	ob.ms.SendTo(TargetPubHash, nm)
	return nil
}

func (ob *Observer) sendBlockVote(br *BlockRound) error {
	nm := &BlockVoteMessage{
		BlockVote: &BlockVote{
			VoteTargetHeight:   ob.round.VoteTargetHeight,
			Header:             br.BlockGenMessage.Block.Header,
			GeneratorSignature: br.BlockGenMessage.GeneratorSignature,
			IsReply:            false,
		},
	}

	s := &block.Signed{
		HeaderHash:         br.BlockGenMessage.Block.Header.Hash(),
		GeneratorSignature: br.BlockGenMessage.GeneratorSignature,
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

	//ob.kn.DebugLog("Observer", "Send BlockVote", br.BlockGenMessage.Block.Header.Formulator, br.BlockGenMessage.Block.Header.Height)
	ob.ms.BroadcastMessage(nm)
	return nil
}

func (ob *Observer) sendBlockVoteTo(br *BlockRound, TargetPubHash common.PublicHash) error {
	if TargetPubHash.Equal(ob.observerPubHash) {
		return nil
	}

	nm := &BlockVoteMessage{
		BlockVote: &BlockVote{
			Header:             br.BlockGenMessage.Block.Header,
			GeneratorSignature: br.BlockGenMessage.GeneratorSignature,
			IsReply:            true,
		},
	}

	s := &block.Signed{
		HeaderHash:         br.BlockGenMessage.Block.Header.Hash(),
		GeneratorSignature: br.BlockGenMessage.GeneratorSignature,
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

	//ob.kn.DebugLog("Observer", "Send BlockVote Reply", br.BlockGenMessage.Block.Header.Formulator, br.BlockGenMessage.Block.Header.Height)
	ob.ms.SendTo(TargetPubHash, nm)
	return nil
}

func (ob *Observer) sendBlockVoteEnd(Height uint32) error {
	for _, br := range ob.round.ClosedBlockRounds {
		if br.TargetHeight == Height {
			votes := []*BlockVote{}
			for _, vt := range br.BlockVoteMap {
				votes = append(votes, vt)
			}
			nm := &BlockVoteEndMessage{
				BlockVoteEnd: &BlockVoteEnd{
					TargetHeight: br.TargetHeight,
					BlockVotes:   votes,
				},
			}
			if sig, err := ob.Config.Key.Sign(nm.BlockVoteEnd.Hash()); err != nil {
				return err
			} else {
				nm.Signature = sig
			}
			//ob.kn.DebugLog("Observer", "Send BlockVoteEnd", br.BlockGenMessage.Block.Header.Formulator, br.BlockGenMessage.Block.Header.Height)
			ob.ms.BroadcastMessage(nm)
			break
		}
	}
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
