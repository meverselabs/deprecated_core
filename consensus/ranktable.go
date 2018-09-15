package consensus

import (
	"errors"
	"sort"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/core/consensus/rank"
)

var (
	// ErrInvalidPhase TODO
	ErrInvalidPhase = errors.New("invalid phase")
	// ErrExistAddress TODO
	ErrExistAddress = errors.New("exist address")
	// ErrExceedCandidateCount TODO
	ErrExceedCandidateCount = errors.New("exceed candidate count")
)

// RankTable TODO
type RankTable struct {
	candidates []*rank.Rank
	rankHash   map[string]*rank.Rank
	height     uint64
}

// NewRankTable TODO
func NewRankTable() *RankTable {
	rt := &RankTable{
		candidates: []*rank.Rank{},
		rankHash:   map[string]*rank.Rank{},
	}
	return rt
}

// Height TODO
func (rt *RankTable) Height() uint64 {
	return rt.height
}

// Add TODO
func (rt *RankTable) Add(s *rank.Rank) error {
	if len(rt.candidates) > 0 {
		if s.Phase() < rt.candidates[0].Phase() {
			return ErrInvalidPhase
		}
	}
	if rt.Rank(s.Address) != nil {
		return ErrExistAddress
	}
	rt.candidates = InsertRankToList(rt.candidates, s)
	rt.rankHash[string(s.Address[:])] = s
	return nil
}

// Remove TODO
func (rt *RankTable) Remove(addr common.Address) {
	delete(rt.rankHash, string(addr[:]))
	candidates := make([]*rank.Rank, 0, len(rt.candidates))
	for _, s := range rt.candidates {
		if !s.Address.Equal(addr) {
			candidates = append(candidates, s)
		}
	}
}

// Rank TODO
func (rt *RankTable) Rank(addr common.Address) *rank.Rank {
	return rt.rankHash[string(addr[:])]
}

// Candidates TODO
func (rt *RankTable) Candidates(cnt int) []*rank.Rank {
	if cnt > len(rt.candidates) {
		return nil
	}

	list := make([]*rank.Rank, 0, cnt)
	for _, m := range rt.candidates {
		list = append(list, m.Clone())
		if len(list) >= cnt {
			break
		}
	}
	return list
}

// ForwardCandidates TODO
func (rt *RankTable) ForwardCandidates(TimeoutCount int, LastTableAppendHash hash.Hash256) error {
	if TimeoutCount >= len(rt.candidates) {
		return ErrExceedCandidateCount
	}

	// increase phase
	for i := 0; i < TimeoutCount; i++ {
		m := rt.candidates[0]
		m.SetPhase(m.Phase() + 1)
		idx := sort.Search(len(rt.candidates)-1, func(i int) bool {
			return m.Less(rt.candidates[i+1])
		})
		copy(rt.candidates, rt.candidates[1:idx+1])
		rt.candidates[idx] = m
	}

	// update top phase and hashSpace
	top := rt.candidates[0]
	top.Set(top.Phase()+1, LastTableAppendHash)
	idx := sort.Search(len(rt.candidates)-1, func(i int) bool {
		return top.Less(rt.candidates[i+1])
	})
	copy(rt.candidates, rt.candidates[1:idx+1])
	rt.candidates[idx] = top

	rt.height++
	return nil
}

// InsertRankToList TODO
func InsertRankToList(ranks []*rank.Rank, s *rank.Rank) []*rank.Rank {
	idx := sort.Search(len(ranks), func(i int) bool {
		return s.Less(ranks[i])
	})
	ranks = append(ranks, s)
	copy(ranks[idx+1:], ranks[idx:])
	ranks[idx] = s
	return ranks
}
