package consensus

import (
	"errors"
	"io"
	"sort"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
)

// ranktable rrors
var (
	ErrInvalidPhase         = errors.New("invalid phase")
	ErrExistAddress         = errors.New("exist address")
	ErrExceedCandidateCount = errors.New("exceed candidate count")
)

// RankTable maintain the rank list of formulators
type RankTable struct {
	height     uint64
	candidates []*Rank
	rankHash   map[common.Address]*Rank
}

// NewRankTable returns a RankTable
func NewRankTable() *RankTable {
	rt := &RankTable{
		candidates: []*Rank{},
		rankHash:   map[common.Address]*Rank{},
	}
	return rt
}

// WriteTo is a serialization function
func (rt *RankTable) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := util.WriteUint64(w, rt.height); err != nil {
		return wrote, err
	} else {
		wrote += n
	}

	if n, err := util.WriteUint32(w, uint32(len(rt.candidates))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, s := range rt.candidates {
			if n, err := s.WriteTo(w); err != nil {
				return wrote, err
			} else {
				wrote += n
			}
		}
	}
	return wrote, nil
}

// ReadFrom is a deserialization function
func (rt *RankTable) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if v, n, err := util.ReadUint64(r); err != nil {
		return read, err
	} else {
		read += n
		rt.height = v
	}

	if Len, n, err := util.ReadUint32(r); err != nil {
		return read, err
	} else {
		read += n
		rt.candidates = make([]*Rank, 0, Len)
		rt.rankHash = map[common.Address]*Rank{}
		for i := 0; i < int(Len); i++ {
			s := new(Rank)
			if n, err := s.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				rt.candidates = append(rt.candidates, s)
				rt.rankHash[s.Address] = s
			}
		}
	}
	return read, nil
}

// Add add the new rank
func (rt *RankTable) Add(s *Rank) error {
	if len(rt.candidates) > 0 {
		if s.Phase() < rt.candidates[0].Phase() {
			return ErrInvalidPhase
		}
	}
	if rt.Rank(s.Address) != nil {
		return ErrExistAddress
	}
	rt.candidates = InsertRankToList(rt.candidates, s)
	rt.rankHash[s.Address] = s
	return nil
}

// LargestPhase returns the largest phase of ranks
func (rt *RankTable) LargestPhase() uint32 {
	if len(rt.candidates) == 0 {
		return 0
	}
	return rt.candidates[len(rt.candidates)-1].phase
}

// Remove delete the rank by the formulation account address
func (rt *RankTable) Remove(addr common.Address) {
	if _, has := rt.rankHash[addr]; has {
		delete(rt.rankHash, addr)
		candidates := make([]*Rank, 0, len(rt.candidates))
		for _, s := range rt.candidates {
			if !s.Address.Equal(addr) {
				candidates = append(candidates, s)
			}
		}
	}
}

// Rank returns the cloned rank by the formulation account address
func (rt *RankTable) Rank(addr common.Address) *Rank {
	return rt.rankHash[addr]
}

// CandidateCount returns the number of ranks
func (rt *RankTable) CandidateCount() int {
	return len(rt.candidates)
}

// Candidates returns next miners
func (rt *RankTable) Candidates(cnt int) []*Rank {
	if cnt > len(rt.candidates) {
		return nil
	}

	list := make([]*Rank, 0, cnt)
	for _, m := range rt.candidates {
		list = append(list, m.Clone())
		if len(list) >= cnt {
			break
		}
	}
	return list
}

// ForwardCandidates rotates the rank table using the timeout count and the hash
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

// InsertRankToList inserts the rank by the score to the rank list
func InsertRankToList(ranks []*Rank, s *Rank) []*Rank {
	idx := sort.Search(len(ranks), func(i int) bool {
		return s.Less(ranks[i])
	})
	ranks = append(ranks, s)
	copy(ranks[idx+1:], ranks[idx:])
	ranks[idx] = s
	return ranks
}
