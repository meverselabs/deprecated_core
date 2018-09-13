package consensus

import (
	"errors"
	"log"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/consensus/rank"
)

var (
	// ErrInvalidRanksLength TODO
	ErrInvalidRanksLength = errors.New("invalid ranks length")
	// ErrAlreadyAppend TODO
	ErrAlreadyAppend = errors.New("already appended")
	// ErrInvalidTableAppendHeight TODO
	ErrInvalidTableAppendHeight = errors.New("invalid table append height")
	// ErrInvalidPrevTableAppendHash TODO
	ErrInvalidPrevTableAppendHash = errors.New("invalid prev table append hash")
	// ErrUnmatchedConfirmedRank TODO
	ErrUnmatchedConfirmedRank = errors.New("unmatched confirmed rank")
	// ErrInvalidRankListOrder TODO
	ErrInvalidRankListOrder = errors.New("invalid rank list order")
	// ErrInvalidTopPubkey TODO
	ErrInvalidTopPubkey = errors.New("invalid top pubkey")
	// ErrWorseRankListOrder TODO
	ErrWorseRankListOrder = errors.New("worse rank list order")
	// ErrInvalidRankPubkey TODO
	ErrInvalidRankPubkey = errors.New("invalid rank pubkey")
)

// ValidateTableAppend TODO
func ValidateTableAppend(rt *RankTable, GroupSize int, msg *SignedTableAppend) error {
	if msg.Height <= rt.height {
		return ErrAlreadyAppend
	}
	if msg.Height != rt.height+1 {
		return ErrInvalidTableAppendHeight
	}
	if len(msg.Ranks) != GroupSize*2 {
		return ErrInvalidRanksLength
	}
	if !rt.lastTableAppendHash.Equal(msg.PrevTableAppendHash) {
		return ErrInvalidPrevTableAppendHash
	}

	Ranks, err := rt.RankList(GroupSize, msg.TailTimeouts)
	if err != nil {
		return err
	}
	var prevRank *rank.Rank
	for i, rank := range msg.Ranks {
		if i < GroupSize {
			tableRank := Ranks[i]
			if !rank.Equal(tableRank) {
				log.Println(msg.Ranks)
				log.Println(Ranks)
				return ErrUnmatchedConfirmedRank
			}
		}
		if prevRank != nil && !prevRank.IsZero() {
			if !prevRank.Less(rank) {
				log.Println(msg.Ranks)
				return ErrInvalidRankListOrder
			}
		}
		prevRank = rank
	}

	joiner := msg.Ranks[GroupSize-1]
	Candidates := Ranks[GroupSize-1:]
	Top := Candidates[0]
	if !Top.Equal(joiner) {
		log.Println("msg.Ranks", msg.Ranks)
		log.Println("msg.TailTimeouts", msg.TailTimeouts)
		log.Println("Members", rt.Members)
		log.Println("Candidates", Candidates)
		log.Println("Top/Joiner", Top, joiner)
		return ErrInvalidTopPubkey
	}

	// Check suggestion is same or better
	Candidates = Candidates[1:]
	rankHash := map[string]*rank.Rank{}
	for _, r := range msg.Ranks[GroupSize:] {
		rankHash[r.Key()] = r
	}
	for _, r := range Candidates {
		rankHash[r.Key()] = r
	}
	mranks := make([]*rank.Rank, 0, len(rankHash))
	for _, v := range rankHash {
		mranks = InsertRankToList(mranks, v)
	}

	for i, r := range msg.Ranks[GroupSize:] {
		if !r.Equal(mranks[i]) {
			log.Println("msg.Rk", msg.Ranks[GroupSize:])
			log.Println("mranks", mranks)
			log.Println("Candid", Candidates)
			return ErrWorseRankListOrder
		}
	}

	// Process slow signature verification after another fast verifications.
	if h, err := msg.TableAppend.Hash(); err != nil {
		return err
	} else if pubkey, err := common.RecoverPubkey(h, msg.Signature); err != nil {
		return err
	} else if !pubkey.Equal(joiner.PublicKey) {
		return ErrInvalidRankPubkey
	}
	return nil
}
