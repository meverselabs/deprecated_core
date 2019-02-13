package consensus

import (
	"bytes"
	"sync"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/account"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/data"
)

// Consensus supports the proof of formulation algorithm
type Consensus struct {
	sync.Mutex
	rankTable              *RankTable
	ObserverKeyMap         map[common.PublicHash]bool
	FormulationAccountType account.Type
}

// NewConsensus returns a Consensus
func NewConsensus(ObserverKeyMap map[common.PublicHash]bool, FormulationAccountType account.Type) *Consensus {
	cs := &Consensus{
		rankTable:              NewRankTable(),
		ObserverKeyMap:         ObserverKeyMap,
		FormulationAccountType: FormulationAccountType,
	}
	return cs
}

// TopRank returns the top rank by Timeoutcount
func (cs *Consensus) TopRank(TimeoutCount int) (*Rank, error) {
	cs.Lock()
	defer cs.Unlock()

	members := cs.rankTable.Candidates(TimeoutCount + 1)
	if members == nil {
		return nil, ErrInsufficientCandidateCount
	}
	Top := members[TimeoutCount]
	return Top, nil
}

// TopRankInMap returns the top rank by Timeoutcount
func (cs *Consensus) TopRankInMap(TimeoutCount int, FormulatorMap map[common.Address]bool) (*Rank, int, error) {
	cs.Lock()
	defer cs.Unlock()

	if len(FormulatorMap) == 0 {
		return nil, 0, ErrInsufficientCandidateCount
	}
	members := cs.rankTable.Candidates(cs.rankTable.CandidateCount())
	if members == nil {
		return nil, 0, ErrInsufficientCandidateCount
	}
	for i := TimeoutCount; i < len(members); i++ {
		r := members[i]
		if FormulatorMap[r.Address] {
			return r, i, nil
		}
	}
	return nil, 0, ErrInsufficientCandidateCount
}

// IsFormulator returns the given information is correct or not
func (cs *Consensus) IsFormulator(Formulator common.Address, Publichash common.PublicHash) bool {
	cs.Lock()
	defer cs.Unlock()

	rank := cs.rankTable.Rank(Formulator)
	if rank == nil {
		return false
	}
	if !rank.PublicHash.Equal(Publichash) {
		return false
	}
	return true
}

// ValidateObserverSignatures validates observer signatures with the signed hash
func (cs *Consensus) ValidateObserverSignatures(signedHash hash.Hash256, sigs []common.Signature) error {
	if len(sigs) != len(cs.ObserverKeyMap)/2+1 {
		return ErrInsufficientObserverSignature
	}
	sigMap := map[common.PublicHash]bool{}
	for _, sig := range sigs {
		pubkey, err := common.RecoverPubkey(signedHash, sig)
		if err != nil {
			return err
		}
		pubhash := common.NewPublicHash(pubkey)
		if !cs.ObserverKeyMap[pubhash] {
			return ErrInvalidObserverSignature
		}
		sigMap[pubhash] = true
	}
	if len(sigMap) != len(sigs) {
		return ErrDuplicatedObserverSignature
	}
	return nil
}

// ApplyGenesis initialize the consensus using the genesis context data
func (cs *Consensus) ApplyGenesis(ctd *data.ContextData) ([]byte, error) {
	cs.Lock()
	defer cs.Unlock()

	phase := cs.rankTable.LargestPhase() + 1
	for _, a := range ctd.CreatedAccountMap {
		if a.Type() == cs.FormulationAccountType {
			acc := a.(*FormulationAccount)
			addr := acc.Address()
			if err := cs.rankTable.Add(NewRank(addr, acc.KeyHash, phase, hash.DoubleHash(addr[:]))); err != nil {
				return nil, err
			}
		}
	}
	for _, acc := range ctd.DeletedAccountMap {
		if acc.Type() == cs.FormulationAccountType {
			cs.rankTable.Remove(acc.Address())
		}
	}
	SaveData, err := cs.buildSaveData()
	if err != nil {
		return nil, err
	}
	return SaveData, nil
}

// ProcessContext processes the consensus using the block and its context data
func (cs *Consensus) ProcessContext(ctd *data.ContextData, HeaderHash hash.Hash256, bh *block.Header) ([]byte, error) {
	cs.Lock()
	defer cs.Unlock()

	if err := cs.rankTable.ForwardCandidates(int(bh.TimeoutCount), HeaderHash); err != nil {
		return nil, err
	}
	phase := cs.rankTable.LargestPhase() + 1
	for _, a := range ctd.CreatedAccountMap {
		if a.Type() == cs.FormulationAccountType {
			acc := a.(*FormulationAccount)
			addr := acc.Address()
			if err := cs.rankTable.Add(NewRank(addr, acc.KeyHash, phase, hash.DoubleHash(addr[:]))); err != nil {
				return nil, err
			}
		}
	}
	for _, acc := range ctd.DeletedAccountMap {
		if acc.Type() == cs.FormulationAccountType {
			cs.rankTable.Remove(acc.Address())
		}
	}

	SaveData, err := cs.buildSaveData()
	if err != nil {
		return nil, err
	}
	return SaveData, nil
}

// IsMinable checks a mining chance of the address with the timeout count
func (cs *Consensus) IsMinable(addr common.Address, TimeoutCount uint32) (bool, error) {
	cs.Lock()
	defer cs.Unlock()

	members := cs.rankTable.Candidates(int(TimeoutCount) + 1)
	if len(members) == 0 {
		return false, ErrInsufficientCandidateCount
	}
	return members[TimeoutCount].Address.Equal(addr), nil
}

func (cs *Consensus) buildSaveData() ([]byte, error) {
	SaveData := []byte{}
	{
		var buffer bytes.Buffer
		if _, err := cs.rankTable.WriteTo(&buffer); err != nil {
			return nil, err
		}
		SaveData = append(SaveData, buffer.Bytes()...)
	}
	{
		var buffer bytes.Buffer
		if _, err := util.WriteUint8(&buffer, uint8(len(cs.ObserverKeyMap))); err != nil {
			return nil, err
		}
		for k := range cs.ObserverKeyMap {
			if _, err := k.WriteTo(&buffer); err != nil {
				return nil, err
			}
		}
		SaveData = append(SaveData, buffer.Bytes()...)
	}
	return SaveData, nil
}

// LoadFromSaveData recover the status using the save data
func (cs *Consensus) LoadFromSaveData(SaveData []byte) error {
	cs.Lock()
	defer cs.Unlock()

	r := bytes.NewReader(SaveData)
	if _, err := cs.rankTable.ReadFrom(r); err != nil {
		return err
	}
	ObserverKeyMap := map[common.PublicHash]bool{}
	if Len, _, err := util.ReadUint8(r); err != nil {
		return err
	} else {
		for i := 0; i < int(Len); i++ {
			var pubhash common.PublicHash
			if _, err := pubhash.ReadFrom(r); err != nil {
				return err
			}
			ObserverKeyMap[pubhash] = true
		}
	}
	cs.ObserverKeyMap = ObserverKeyMap
	return nil
}
