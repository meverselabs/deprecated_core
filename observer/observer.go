package observer

import (
	"io"
	"log"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fletaio/framework/router"

	"github.com/fletaio/common/queue"
	"github.com/fletaio/common/util"
	"github.com/fletaio/core/consensus"
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

type messageItem struct {
	PublicHash common.PublicHash
	Message    message.Message
	Raw        []byte
}

// Observer supports the chain validation
type Observer struct {
	obLock               *router.NamedLock
	Config               *Config
	observerPubHash      common.PublicHash
	round                *VoteRound
	roundFirstTime       uint64
	roundFirstHeight     uint32
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
	messageQueue         *queue.Queue

	prevRoundEndTime int64 // FOR DEBUG
}

// NewObserver returns a Observer
func NewObserver(Config *Config, kn *kernel.Kernel) (*Observer, error) {
	Height := kn.Provider().Height()
	ob := &Observer{
		Config:          Config,
		obLock:          router.NewNamedLock("ConnMap"),
		observerPubHash: common.NewPublicHash(Config.Key.PublicKey()),
		round:           NewVoteRound(Height+1, kn.Config.MaxBlocksPerFormulator),
		ignoreMap:       map[common.Address]int64{},
		cm:              chain.NewManager(kn),
		kn:              kn,
		mm:              message.NewManager(),
		runEnd:          make(chan struct{}, 1),
		messageQueue:    queue.NewQueue(),
	}
	ob.mm.SetCreator(chain.RequestMessageType, ob.messageCreator)
	ob.mm.SetCreator(message_def.BlockGenMessageType, ob.messageCreator)
	ob.mm.SetCreator(RoundVoteMessageType, ob.messageCreator)
	ob.mm.SetCreator(RoundVoteAckMessageType, ob.messageCreator)
	ob.mm.SetCreator(BlockVoteMessageType, ob.messageCreator)
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

	ob.obLock.Lock("Close")
	defer ob.obLock.Unlock()

	ob.isClose = true
	ob.kn.Close()
	ob.runEnd <- struct{}{}
}

// Run operates servers and a round voting
func (ob *Observer) Run(BindObserver string, BindFormulator string) {
	ob.obLock.Lock("Run")
	if ob.isRunning {
		ob.obLock.Unlock()
		return
	}
	ob.isRunning = true
	ob.obLock.Unlock()

	go ob.ms.Run(BindObserver)
	go ob.fs.Run(BindFormulator)
	go ob.cm.Run()

	voteTimer := time.NewTimer(time.Millisecond)
	queueTimer := time.NewTimer(time.Millisecond)
	for !ob.isClose {
		select {
		case <-queueTimer.C:
			v := ob.messageQueue.Pop()
			for v != nil {
				item := v.(*messageItem)
				if msg, is := item.Message.(*message_def.BlockGenMessage); is {
					ob.obLock.Lock("Run queueTimer msg is blockgenmessage")
					ob.handleBlockGenMessage(msg, item.Raw)
					ob.obLock.Unlock()
				} else {
					ob.obLock.Lock("Run queueTimer msg is not blockgenmessage")
					ob.handleObserverMessage(item.PublicHash, item.Message)
					ob.obLock.Unlock()
				}
				v = ob.messageQueue.Pop()
			}
			queueTimer.Reset(10 * time.Millisecond)
		case <-voteTimer.C:
			ob.obLock.Lock("Run voteTimer")
			ob.kn.DebugLog("Observer", ob.kn.Provider().Height(), "Current State", ob.round.RoundState, len(ob.adjustFormulatorMap()), ob.fs.PeerCount(), (time.Now().UnixNano()-ob.prevRoundEndTime)/int64(time.Millisecond))
			lastestProcessHeight := atomic.LoadUint32(&ob.lastestProcessHeight)
			if lastestProcessHeight >= ob.round.VoteTargetHeight {
				if ob.round.BlockRoundCount() > 0 && lastestProcessHeight < ob.round.BlockRounds[len(ob.round.BlockRounds)-1].TargetHeight {
					for _, br := range ob.round.BlockRounds {
						if br.TargetHeight <= lastestProcessHeight {
							ob.round.RemoveBlockRound(br)
						}
					}
					if ob.round.RoundState == BlockVoteState {
						if ob.round.BlockRoundCount() > 0 {
							br := ob.round.BlockRounds[0]
							if br.BlockGenMessageWait != nil && br.BlockGenMessage == nil {
								ob.messageQueue.Push(&messageItem{
									Message: br.BlockGenMessageWait,
								})
							}
						}
					}
				} else {
					ob.kn.DebugLog("Observer", ob.kn.Provider().Height(), "Change State", RoundVoteState, (time.Now().UnixNano()-ob.prevRoundEndTime)/int64(time.Millisecond))
					ob.round = NewVoteRound(ob.kn.Provider().Height()+1, ob.kn.Config.MaxBlocksPerFormulator)
					ob.prevRoundEndTime = time.Now().UnixNano()
				}
			}
			if len(ob.adjustFormulatorMap()) > 0 {
				if ob.round.RoundState == RoundVoteState && !ob.round.HasForwardVote {
					ob.sendRoundVote()
				} else {
					ob.round.VoteFailCount++
					if ob.round.VoteFailCount > 20 {
						ob.kn.DebugLog(ob.kn.Provider().Height(), "Fail State", ob.round.RoundState, len(ob.adjustFormulatorMap()), ob.fs.PeerCount())
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
						ob.kn.DebugLog("Observer", ob.kn.Provider().Height(), "Change State", RoundVoteState, (time.Now().UnixNano()-ob.prevRoundEndTime)/int64(time.Millisecond))
						ob.round = NewVoteRound(ob.kn.Provider().Height()+1, ob.kn.Config.MaxBlocksPerFormulator)
						ob.roundFirstTime = 0
						ob.roundFirstHeight = 0
						ob.prevRoundEndTime = time.Now().UnixNano()
						ob.sendRoundVote()
					}
				}
			}
			ob.obLock.Unlock()

			voteTimer.Reset(100 * time.Millisecond)
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
					ob.obLock.Lock("OnRecv msg is RequestMessage")
					defer ob.obLock.Unlock()

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
				ob.messageQueue.Push(&messageItem{
					Message: msg,
					Raw:     rd.(*Duplicator).buffer.Bytes(),
				})
			} else if op, is := p.(*Peer); is {
				ob.messageQueue.Push(&messageItem{
					PublicHash: op.pubhash,
					Message:    m,
				})
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

		//ob.kn.DebugLog("Observer", SenderPublicHash, ob.round.RoundState, ob.kn.Provider().Height(), msg.RoundVote.VoteTargetHeight, "RoundVoteMessage", ob.round.VoteTargetHeight, len(ob.round.RoundVoteMessageMap), len(ob.Config.ObserverKeyMap)/2+2)
		if !msg.RoundVote.ChainCoord.Equal(ob.kn.ChainCoord()) {
			return ErrInvalidVote
		}

		cp := ob.kn.Provider()
		if msg.RoundVote.VoteTargetHeight != ob.round.VoteTargetHeight {
			if !SenderPublicHash.Equal(ob.observerPubHash) {
				if msg.RoundVote.VoteTargetHeight < ob.round.VoteTargetHeight {
					ob.ms.SendTo(SenderPublicHash, &chain.StatusMessage{
						Version:  cp.Version(),
						Height:   cp.Height(),
						LastHash: cp.LastHash(),
					})
				} else {
					ob.round.HasForwardVote = true
				}
			}
			return ErrInvalidVote
		}

		if ob.round.RoundState != RoundVoteState {
			if !msg.RoundVote.IsReply {
				ob.sendRoundVoteTo(SenderPublicHash)
			}
			return ErrInvalidRoundState
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

		if old, has := ob.round.RoundVoteMessageMap[SenderPublicHash]; has {
			if msg.RoundVote.Timestamp <= old.RoundVote.Timestamp {
				if !msg.RoundVote.IsReply {
					ob.sendRoundVoteTo(SenderPublicHash)
				}
				return ErrAlreadyVoted
			}
		}
		ob.round.RoundVoteMessageMap[SenderPublicHash] = msg

		if !msg.RoundVote.IsReply {
			ob.sendRoundVoteTo(SenderPublicHash)
		}

		if len(ob.round.RoundVoteMessageMap) >= len(ob.Config.ObserverKeyMap)/2+2 {
			votes := []*voteSortItem{}
			for pubhash, msg := range ob.round.RoundVoteMessageMap {
				votes = append(votes, &voteSortItem{
					PublicHash: pubhash,
					Priority:   uint64(msg.RoundVote.TimeoutCount),
				})
			}
			sort.Sort(voteSorter(votes))

			ob.kn.DebugLog("Observer", ob.kn.Provider().Height(), "Change State", RoundVoteAckState, (time.Now().UnixNano()-ob.prevRoundEndTime)/int64(time.Millisecond))
			ob.round.RoundState = RoundVoteAckState
			ob.round.PublicHash = votes[0].PublicHash

			if ob.roundFirstTime == 0 {
				ob.roundFirstTime = uint64(time.Now().UnixNano())
				ob.roundFirstHeight = uint32(ob.kn.Provider().Height())
			}

			ob.sendRoundVoteAck()

			for pubhash, msg := range ob.round.RoundVoteAckMessageWaitMap {
				ob.messageQueue.Push(&messageItem{
					PublicHash: pubhash,
					Message:    msg,
				})
			}
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

		//ob.kn.DebugLog("Observer", ob.round.RoundState, ob.kn.Provider().Height(), "RoundVoteAckMessage")
		cp := ob.kn.Provider()
		if msg.RoundVoteAck.VoteTargetHeight != ob.round.VoteTargetHeight {
			if !SenderPublicHash.Equal(ob.observerPubHash) {
				if msg.RoundVoteAck.VoteTargetHeight < ob.round.VoteTargetHeight {
					ob.ms.SendTo(SenderPublicHash, &chain.StatusMessage{
						Version:  cp.Version(),
						Height:   cp.Height(),
						LastHash: cp.LastHash(),
					})
				}
			}
			return ErrInvalidVote
		}
		if ob.round.RoundState != RoundVoteAckState {
			if ob.round.RoundState < RoundVoteAckState {
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

		if old, has := ob.round.RoundVoteAckMessageMap[SenderPublicHash]; has {
			if msg.RoundVoteAck.Timestamp <= old.RoundVoteAck.Timestamp {
				if !msg.RoundVoteAck.IsReply {
					ob.sendRoundVoteAckTo(SenderPublicHash)
				}
				return ErrAlreadyVoted
			}
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
				ob.kn.DebugLog("Observer", ob.kn.Provider().Height(), "Change State", BlockVoteState, (time.Now().UnixNano()-ob.prevRoundEndTime)/int64(time.Millisecond))
				ob.round.RoundState = BlockVoteState
				ob.round.MinRoundVoteAck = MinRoundVoteAck

				if cp.Height() > 0 {
					LastHeader, err := cp.Header(cp.Height())
					if err != nil {
						return err
					}
					if LastHeader.(*block.Header).Formulator.Equal(MinRoundVoteAck.Formulator) {
						ob.round.BlockRounds = ob.round.BlockRounds[:ob.kn.Config.MaxBlocksPerFormulator-ob.kn.BlocksFromSameFormulator()]
					}
				}

				if ob.round.MinRoundVoteAck.PublicHash.Equal(ob.observerPubHash) {
					nm := &message_def.BlockReqMessage{
						PrevHash:             ob.kn.Provider().LastHash(),
						TargetHeight:         ob.round.VoteTargetHeight,
						TimeoutCount:         ob.round.MinRoundVoteAck.TimeoutCount,
						Formulator:           ob.round.MinRoundVoteAck.Formulator,
						FormulatorPublicHash: ob.round.MinRoundVoteAck.FormulatorPublicHash,
					}
					ob.fs.SendTo(ob.round.MinRoundVoteAck.Formulator, nm)
					//ob.kn.DebugLog("Observer", "Send BlockReqMessage", ob.round.VoteTargetHeight, nm.TimeoutCount, nm.Formulator, Top.Address, ob.kn.Provider().Height())
				}
				if ob.round.BlockRoundCount() > 0 {
					br := ob.round.BlockRounds[0]
					if br.BlockGenMessageWait != nil && br.BlockGenMessage == nil {
						ob.messageQueue.Push(&messageItem{
							Message: br.BlockGenMessageWait,
						})
					}
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

		//ob.kn.DebugLog("Observer", ob.round.RoundState, ob.kn.Provider().Height(), "BlockVoteMessage")
		cp := ob.kn.Provider()
		if ob.round.BlockRoundCount() == 0 {
			return ErrInvalidVote
		}
		if msg.BlockVote.VoteTargetHeight != ob.round.VoteTargetHeight {
			if !SenderPublicHash.Equal(ob.observerPubHash) {
				if msg.BlockVote.VoteTargetHeight < ob.round.VoteTargetHeight {
					ob.ms.SendTo(SenderPublicHash, &chain.StatusMessage{
						Version:  cp.Version(),
						Height:   cp.Height(),
						LastHash: cp.LastHash(),
					})
				}
			}
			return ErrInvalidVote
		}
		if ob.round.RoundState != BlockVoteState {
			if ob.round.RoundState < BlockVoteState {
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
		if msg.BlockVote.Header.Height() != br.TargetHeight {
			if msg.BlockVote.Header.Height() > br.TargetHeight {
				for _, br := range ob.round.BlockRounds {
					if br.TargetHeight == msg.BlockVote.Header.Height() {
						br.BlockVoteMessageWaitMap[SenderPublicHash] = msg
						break
					}
				}
			} else {
				if !msg.BlockVote.IsReply {
					for _, br := range ob.round.ClosedBlockRounds {
						if br.TargetHeight == msg.BlockVote.Header.Height() {
							ob.sendBlockVoteTo(br, SenderPublicHash)
							break
						}
					}
				}
			}
			return ErrInvalidRoundState
		}
		if br.BlockGenMessage == nil {
			br.BlockVoteMessageWaitMap[SenderPublicHash] = msg
			return ErrInvalidRoundState
		}

		if !msg.BlockVote.Header.PrevHash().Equal(cp.LastHash()) {
			return ErrInvalidVote
		}
		if msg.BlockVote.Header.Height() != cp.Height()+1 {
			return ErrInvalidVote
		}
		if !msg.BlockVote.Header.Hash().Equal(br.BlockGenMessage.Block.Header.Hash()) {
			return ErrInvalidVote
		}

		HeaderHash := msg.BlockVote.Header.Hash()
		if pubkey, err := common.RecoverPubkey(HeaderHash, msg.BlockVote.GeneratorSignature); err != nil {
			return err
		} else {
			if !ob.round.MinRoundVoteAck.FormulatorPublicHash.Equal(common.NewPublicHash(pubkey)) {
				return ErrInvalidFormulatorSignature
			}
		}
		s := &block.Signed{
			HeaderHash:         HeaderHash,
			GeneratorSignature: msg.BlockVote.GeneratorSignature,
		}
		if pubkey, err := common.RecoverPubkey(s.Hash(), msg.BlockVote.ObserverSignature); err != nil {
			return err
		} else {
			if !SenderPublicHash.Equal(common.NewPublicHash(pubkey)) {
				return ErrInvalidVoteSignature
			}
		}

		if _, has := br.BlockVoteMap[SenderPublicHash]; has {
			if !msg.BlockVote.IsReply {
				ob.sendBlockVoteTo(br, SenderPublicHash)
			}
			return ErrAlreadyVoted
		}
		br.BlockVoteMap[SenderPublicHash] = msg.BlockVote

		if !msg.BlockVote.IsReply {
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

			adjustMap := ob.adjustFormulatorMap()
			delete(adjustMap, ob.round.MinRoundVoteAck.Formulator)
			var NextTop *consensus.Rank
			if len(adjustMap) > 0 {
				r, _, err := ob.kn.TopRankInMap(adjustMap)
				if err != nil {
					return err
				}
				NextTop = r
			}

			if ob.round.MinRoundVoteAck.PublicHash.Equal(ob.observerPubHash) {
				nm := &message_def.BlockObSignMessage{
					TargetHeight: msg.BlockVote.Header.Height(),
					ObserverSigned: &block.ObserverSigned{
						Signed: block.Signed{
							HeaderHash:         msg.BlockVote.Header.Hash(),
							GeneratorSignature: msg.BlockVote.GeneratorSignature,
						},
						ObserverSignatures: sigs,
					},
				}
				ob.fs.SendTo(ob.round.MinRoundVoteAck.Formulator, nm)
				ob.fs.UpdateGuessHeight(ob.round.MinRoundVoteAck.Formulator, nm.TargetHeight)
				//ob.kn.DebugLog("Observer", "Send BlockObSignMessage", nm.TargetHeight)

				if NextTop != nil && !NextTop.Address.Equal(ob.round.MinRoundVoteAck.Formulator) {
					ob.fs.SendTo(NextTop.Address, &chain.StatusMessage{
						Version:  cd.Header.Version(),
						Height:   cd.Header.Height(),
						LastHash: cd.Header.Hash(),
					})
					//ob.kn.DebugLog("Observer", "Send StatusMessage to NextTop", cd.Header.Height(), NextTop.Address)
				}
			} else {
				adjustMap := ob.adjustFormulatorMap()
				if NextTop != nil {
					delete(adjustMap, NextTop.Address)
					ob.fs.UpdateGuessHeight(NextTop.Address, cd.Header.Height())
				}
				if len(adjustMap) > 0 {
					ranks, err := ob.kn.RanksInMap(adjustMap, 3)
					if err == nil {
						for _, v := range ranks {
							ob.fs.SendTo(v.Address, &chain.StatusMessage{
								Version:  cd.Header.Version(),
								Height:   cd.Header.Height(),
								LastHash: cd.Header.Hash(),
							})
							//ob.kn.DebugLog("Observer", "Send StatusMessage to 3", cd.Header.Height(), v.Address)
						}
					}
				}
			}

			ob.round.CloseBlockRound()

			PastTime := uint64(time.Now().UnixNano()) - ob.roundFirstTime
			ExpectedTime := uint64(msg.BlockVote.Header.Height()-ob.roundFirstHeight) * uint64(500*time.Millisecond)

			if PastTime < ExpectedTime {
				diff := time.Duration(ExpectedTime - PastTime)
				if diff > 500*time.Millisecond {
					diff = 500 * time.Millisecond
				}
				time.Sleep(diff)
				//ob.kn.DebugLog("Observer", ob.kn.Provider().Height(), "Sleep", int64(ExpectedTime-PastTime)/int64(time.Millisecond))
			}

			if ob.round.BlockRoundCount() > 0 {
				ob.round.VoteFailCount = 0
				if ob.round.BlockRoundCount() > 0 {
					br := ob.round.BlockRounds[0]
					if br.BlockGenMessageWait != nil && br.BlockGenMessage == nil {
						ob.messageQueue.Push(&messageItem{
							Message: br.BlockGenMessageWait,
						})
					}
				}
			} else {
				ob.kn.DebugLog("Observer", ob.kn.Provider().Height(), "Change State", RoundVoteState, (time.Now().UnixNano()-ob.prevRoundEndTime)/int64(time.Millisecond))
				ob.round = NewVoteRound(ob.kn.Provider().Height()+1, ob.kn.Config.MaxBlocksPerFormulator)
				ob.prevRoundEndTime = time.Now().UnixNano()
				ob.sendRoundVote()
			}
		}
		return nil
	default:
		return message.ErrUnhandledMessage
	}
}

func (ob *Observer) handleBlockGenMessage(msg *message_def.BlockGenMessage, raw []byte) error {
	//ob.kn.DebugLog("Observer", ob.kn.Provider().Height(), "BlockGenMessage", msg.Block.Header.Height())
	if msg.Block.Header.Height() < ob.round.VoteTargetHeight {
		return ErrInvalidVote
	}
	if msg.Block.Header.Height() >= ob.round.VoteTargetHeight+ob.kn.Config.MaxBlocksPerFormulator {
		return ErrInvalidVote
	}
	if ob.round.RoundState != BlockVoteState {
		if ob.round.RoundState < BlockVoteState {
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
		log.Println(err)
		return err
	}
	br.BlockGenMessage = msg
	br.Context = ctx

	ob.sendBlockVote(br)

	for pubhash, msg := range br.BlockVoteMessageWaitMap {
		ob.messageQueue.Push(&messageItem{
			PublicHash: pubhash,
			Message:    msg,
		})
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
			Timestamp:            uint64(time.Now().UnixNano()),
			IsReply:              false,
		},
	}
	if gh, err := ob.fs.GuessHeight(Top.Address); err != nil {
		ob.fs.SendTo(Top.Address, &chain.StatusMessage{
			Version:  cp.Version(),
			Height:   cp.Height(),
			LastHash: cp.LastHash(),
		})
	} else if gh < cp.Height() {
		ob.fs.SendTo(Top.Address, &chain.StatusMessage{
			Version:  cp.Version(),
			Height:   cp.Height(),
			LastHash: cp.LastHash(),
		})
	}
	//ob.kn.DebugLog("Observer", "Send StatusMessage in RoundVote", cp.Height(), Top.Address)

	ob.round.VoteFailCount = 0

	if sig, err := ob.Config.Key.Sign(nm.RoundVote.Hash()); err != nil {
		return err
	} else {
		nm.Signature = sig
	}

	ob.messageQueue.Push(&messageItem{
		PublicHash: ob.observerPubHash,
		Message:    nm,
	})

	//ob.kn.DebugLog("Observer", "Send RoundVote", nm.RoundVote.Formulator, nm.RoundVote.VoteTargetHeight, cp.Height())
	ob.ms.BroadcastMessage(nm)
	return nil
}

func (ob *Observer) sendRoundVoteTo(TargetPubHash common.PublicHash) error {
	if TargetPubHash.Equal(ob.observerPubHash) {
		return nil
	}

	MyMsg, has := ob.round.RoundVoteMessageMap[ob.observerPubHash]
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
			Timestamp:            MyMsg.RoundVote.Timestamp,
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
	MinRoundVote := ob.round.RoundVoteMessageMap[ob.round.PublicHash].RoundVote
	nm := &RoundVoteAckMessage{
		RoundVoteAck: &RoundVoteAck{
			VoteTargetHeight:     MinRoundVote.VoteTargetHeight,
			TimeoutCount:         MinRoundVote.TimeoutCount,
			Formulator:           MinRoundVote.Formulator,
			FormulatorPublicHash: MinRoundVote.FormulatorPublicHash,
			PublicHash:           ob.round.PublicHash,
			Timestamp:            uint64(time.Now().UnixNano()),
			IsReply:              false,
		},
	}
	if sig, err := ob.Config.Key.Sign(nm.RoundVoteAck.Hash()); err != nil {
		return err
	} else {
		nm.Signature = sig
	}

	ob.messageQueue.Push(&messageItem{
		PublicHash: ob.observerPubHash,
		Message:    nm,
	})

	//ob.kn.DebugLog("Observer", "Send RoundVoteAck", nm.RoundVoteAck.Formulator, nm.RoundVoteAck.VoteTargetHeight)
	ob.ms.BroadcastMessage(nm)
	return nil
}

func (ob *Observer) sendRoundVoteAckTo(TargetPubHash common.PublicHash) error {
	if TargetPubHash.Equal(ob.observerPubHash) {
		return nil
	}

	MyMsg, has := ob.round.RoundVoteAckMessageMap[ob.observerPubHash]
	if !has {
		return nil
	}
	nm := &RoundVoteAckMessage{
		RoundVoteAck: &RoundVoteAck{
			VoteTargetHeight:     MyMsg.RoundVoteAck.VoteTargetHeight,
			TimeoutCount:         MyMsg.RoundVoteAck.TimeoutCount,
			Formulator:           MyMsg.RoundVoteAck.Formulator,
			FormulatorPublicHash: MyMsg.RoundVoteAck.FormulatorPublicHash,
			PublicHash:           MyMsg.RoundVoteAck.PublicHash,
			Timestamp:            MyMsg.RoundVoteAck.Timestamp,
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

	ob.messageQueue.Push(&messageItem{
		PublicHash: ob.observerPubHash,
		Message:    nm,
	})

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
	default:
		return nil, message.ErrUnknownMessage
	}
}
