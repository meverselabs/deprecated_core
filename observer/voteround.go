package observer

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/data"
	"git.fleta.io/fleta/core/message_def"
)

// consts
const (
	EmptyState        = iota
	RoundVoteState    = iota
	RoundVoteAckState = iota
	BlockVoteState    = iota
)

// VoteRound is data for the voting round
type VoteRound struct {
	TargetHeight               uint32
	PublicHash                 common.PublicHash
	RoundVoteAckMap            map[common.PublicHash]*RoundVoteAck
	MinRoundVoteAck            *RoundVoteAck
	BlockVoteMap               map[common.PublicHash]*BlockVote
	BlockGenMessage            *message_def.BlockGenMessage
	Context                    *data.Context
	RoundVoteAckMessageWaitMap map[common.PublicHash]*RoundVoteAckMessage
	BlockVoteWaitMap           map[common.PublicHash]*BlockVote
	BlockVoteMessageWaitMap    map[common.PublicHash]*BlockVoteMessage
	BlockGenMessageWait        *message_def.BlockGenMessage
}

// NewVoteRound returns a VoteRound
func NewVoteRound(TargetHeight uint32, PublicHash common.PublicHash) *VoteRound {
	vr := &VoteRound{
		TargetHeight:               TargetHeight,
		PublicHash:                 PublicHash,
		RoundVoteAckMap:            map[common.PublicHash]*RoundVoteAck{},
		BlockVoteMap:               map[common.PublicHash]*BlockVote{},
		RoundVoteAckMessageWaitMap: map[common.PublicHash]*RoundVoteAckMessage{},
		BlockVoteWaitMap:           map[common.PublicHash]*BlockVote{},
		BlockVoteMessageWaitMap:    map[common.PublicHash]*BlockVoteMessage{},
	}
	return vr
}

type voteSortItem struct {
	PublicHash common.PublicHash
	Priority   uint64
}

type voteSorter []*voteSortItem

func (s voteSorter) Len() int {
	return len(s)
}

func (s voteSorter) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s voteSorter) Less(i, j int) bool {
	a := s[i]
	b := s[j]
	if a.Priority == b.Priority {
		return a.PublicHash.Less(b.PublicHash)
	} else {
		return a.Priority < b.Priority
	}
}
