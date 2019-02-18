package observer

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
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
	RoundHash                  hash.Hash256
	RoundVoteAckMap            map[common.PublicHash]*RoundVoteAck
	MinRoundVoteAck            *RoundVoteAck
	BlockVoteMap               map[common.PublicHash]*BlockVote
	BlockGenMessage            *message_def.BlockGenMessage
	Context                    *data.Context
	RoundVoteAckMessageWaitMap map[hash.Hash256]*RoundVoteAckMessage
	BlockVoteMessageWaitMap    map[hash.Hash256]*BlockVoteMessage
	BlockGenMessageWaitMap     map[hash.Hash256]*message_def.BlockGenMessage
}

// NewVoteRound returns a VoteRound
func NewVoteRound(RoundHash hash.Hash256) *VoteRound {
	vr := &VoteRound{
		RoundHash:                  RoundHash,
		RoundVoteAckMap:            map[common.PublicHash]*RoundVoteAck{},
		BlockVoteMap:               map[common.PublicHash]*BlockVote{},
		RoundVoteAckMessageWaitMap: map[hash.Hash256]*RoundVoteAckMessage{},
		BlockVoteMessageWaitMap:    map[hash.Hash256]*BlockVoteMessage{},
		BlockGenMessageWaitMap:     map[hash.Hash256]*message_def.BlockGenMessage{},
	}
	return vr
}
