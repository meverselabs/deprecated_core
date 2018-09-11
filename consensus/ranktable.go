package consensus

import (
	"errors"
	"sort"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/core/consensus/rank"
	"git.fleta.io/fleta/core/consensus/timeout"
)

var (
	// ErrInvalidPhase TODO
	ErrInvalidPhase = errors.New("invalid phase")
	// ErrExistPublicKey TODO
	ErrExistPublicKey = errors.New("exist public key")
	// ErrInsufficientCandidateCount TODO
	ErrInsufficientCandidateCount = errors.New("insufficient candidate count")
	// ErrInvalidTimeoutPublicKey TODO
	ErrInvalidTimeoutPublicKey = errors.New("invalid timeout pubkey")
	// ErrExceedCandidateCount TODO
	ErrExceedCandidateCount = errors.New("exceed candidate count")
)

// RankTable TODO
type RankTable struct {
	recentGenerators    []*rank.Rank
	members             []*rank.Rank
	candidates          []*rank.Rank
	rankHash            map[string]*rank.Rank
	height              uint32
	lastTableAppendHash hash.Hash256
	recentSize          int
}

// NewRankTable TODO
func NewRankTable(recentSize int, GenesisTableAppendHash hash.Hash256) *RankTable {
	rm := &RankTable{
		recentGenerators:    []*rank.Rank{},
		members:             []*rank.Rank{},
		candidates:          []*rank.Rank{},
		rankHash:            map[string]*rank.Rank{},
		lastTableAppendHash: GenesisTableAppendHash,
		recentSize:          recentSize,
	}
	return rm
}

// Clone TODO
func (rt *RankTable) Clone() *RankTable {
	var rh hash.Hash256
	copy(rh[:], rt.lastTableAppendHash[:])
	nrt := &RankTable{
		recentGenerators:    make([]*rank.Rank, 0, len(rt.recentGenerators)),
		members:             make([]*rank.Rank, 0, len(rt.members)),
		candidates:          make([]*rank.Rank, 0, len(rt.candidates)),
		rankHash:            make(map[string]*rank.Rank),
		height:              rt.height,
		lastTableAppendHash: rh,
		recentSize:          rt.recentSize,
	}
	for _, m := range rt.recentGenerators {
		nrt.recentGenerators = append(nrt.recentGenerators, m.Clone())
	}
	for _, m := range rt.members {
		nrt.members = append(nrt.members, m.Clone())
	}
	for _, c := range rt.candidates {
		nrt.candidates = append(nrt.candidates, c.Clone())
	}
	for k, v := range rt.rankHash {
		nrt.rankHash[k] = v.Clone()
	}
	return nrt
}

// Add TODO
func (rt *RankTable) Add(s *rank.Rank) error {
	if len(rt.candidates) > 0 {
		if s.Phase() < rt.candidates[0].Phase() {
			return ErrInvalidPhase
		}
	}
	if rt.Rank(s.PublicKey) != nil {
		return ErrExistPublicKey
	}
	rt.candidates = InsertRankToList(rt.candidates, s)
	rt.rankHash[string(s.PublicKey[:])] = s
	return nil
}

// Rank TODO
func (rt *RankTable) Rank(pubkey common.PublicKey) *rank.Rank {
	return rt.rankHash[string(pubkey[:])]
}

// Members TODO
func (rt *RankTable) Members(cnt int) []*rank.Rank {
	list := make([]*rank.Rank, 0, cnt)
	for _, m := range rt.members {
		list = append(list, m.Clone())
	}
	return list
}

// ForwardMembers TODO
func (rt *RankTable) ForwardMembers(RemoveLen int) {
	if RemoveLen > 0 {
		consumed := rt.members[:RemoveLen]
		rt.members = rt.members[RemoveLen:]
		if len(consumed) >= rt.recentSize {
			rt.recentGenerators = consumed[len(consumed)-rt.recentSize:]
		} else {
			if len(rt.recentGenerators)+len(consumed) > rt.recentSize {
				rt.recentGenerators = rt.recentGenerators[rt.recentSize-len(consumed):]
			}
			rt.recentGenerators = append(rt.recentGenerators, consumed...)
		}
	}
}

// Candidates TODO
func (rt *RankTable) Candidates(cnt int) []*rank.Rank {
	list := make([]*rank.Rank, 0, cnt)
	for _, m := range rt.candidates {
		list = append(list, m.Clone())
	}
	return list
}

// RankList TODO
func (rt *RankTable) RankList(GroupSize int, TailTimeouts []*timeout.Timeout) ([]*rank.Rank, error) {
	if len(rt.candidates)-len(TailTimeouts) < GroupSize+1 {
		return nil, ErrInsufficientCandidateCount
	}

	RankListSize := GroupSize * 2
	newRanks := make([]*rank.Rank, 0, RankListSize)
	MergedList := rt.mergedList(GroupSize - 1)
	BeginPosition := GroupSize - len(MergedList) - 1
	for i := 0; i < BeginPosition; i++ {
		newRanks = append(newRanks, &rank.Rank{})
	}
	for _, r := range MergedList {
		newRanks = append(newRanks, r)
	}
	candidates := rt.candidates
	for i, to := range TailTimeouts {
		m := candidates[i]
		if !to.PublicKey.Equal(m.PublicKey) {
			return nil, ErrInvalidTimeoutPublicKey
		}
	}
	candidates = candidates[len(TailTimeouts):]
	for _, m := range candidates {
		newRanks = append(newRanks, m.Clone())
		if len(newRanks) >= RankListSize {
			break
		}
	}
	return newRanks, nil
}

func (rt *RankTable) mergedList(count int) []*rank.Rank {
	Mergeds := make([]*rank.Rank, 0, len(rt.members))
	if count > len(rt.members) {
		remain := count - len(rt.members)
		begin := len(rt.recentGenerators) - remain
		if begin < 0 {
			begin = 0
		}
		for _, v := range rt.recentGenerators[begin:] {
			Mergeds = append(Mergeds, v.Clone())
		}
	}
	remain := count - len(Mergeds)
	if remain > len(rt.members) {
		for _, v := range rt.members {
			Mergeds = append(Mergeds, v.Clone())
		}
	} else {
		for _, v := range rt.members[len(rt.members)-remain:] {
			Mergeds = append(Mergeds, v.Clone())
		}
	}
	return Mergeds
}

// ForwardCandidates TODO
func (rt *RankTable) ForwardCandidates(RemoveLen int, LastTableAppendHash hash.Hash256) error {
	if RemoveLen >= len(rt.candidates) {
		return ErrExceedCandidateCount
	}
	// detach from candidates
	detaches := rt.candidates[:RemoveLen]
	rt.candidates = rt.candidates[RemoveLen:]
	// clone last to members
	last := rt.candidates[0]
	rt.members = append(rt.members, last.Clone())
	// increase detaches' phase
	for _, s := range detaches {
		delete(rt.rankHash, string(s.PublicKey[:]))
		s.SetPhase(s.Phase() + 1)
	}
	// update last's hashSpace
	last.Set(last.Phase()+1, LastTableAppendHash)
	// insert last
	idx := sort.Search(len(rt.candidates)-1, func(i int) bool {
		return last.Less(rt.candidates[i+1])
	})
	copy(rt.candidates, rt.candidates[1:idx+1])
	rt.candidates[idx] = last
	// update hash
	rt.lastTableAppendHash = LastTableAppendHash
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
