package consensus

import (
	"bytes"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/account"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/data"
)

// Consensus supports the proof of formulation algorithm
type Consensus struct {
	RankTable              *RankTable
	ObserverKeyMap         map[common.PublicHash]bool
	FormulationAccountType account.Type
}

// NewConsensus returns a Consensus
func NewConsensus(ObserverKeyMap map[common.PublicHash]bool, FormulationAccountType account.Type) *Consensus {
	cs := &Consensus{
		RankTable:              NewRankTable(),
		ObserverKeyMap:         ObserverKeyMap,
		FormulationAccountType: FormulationAccountType,
	}
	return cs
}

// ValidateBlockHeader validate the block header with signatures and previus block's information
func (cs *Consensus) ValidateBlockHeader(bh *block.Header, s *block.Signed) error {
	members := cs.RankTable.Candidates(int(bh.TimeoutCount) + 1)
	if members == nil {
		return ErrInsufficientCandidateCount
	}
	Top := members[bh.TimeoutCount]
	if !Top.Address.Equal(bh.FormulationAddress) {
		return ErrInvalidTopMember
	}
	pubkey, err := common.RecoverPubkey(s.HeaderHash, s.GeneratorSignature)
	if err != nil {
		return err
	}
	pubhash := common.NewPublicHash(pubkey)
	if !Top.PublicHash.Equal(pubhash) {
		return ErrInvalidTopSignature
	}
	return nil
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
	phase := cs.RankTable.LargestPhase() + 1
	for _, a := range ctd.CreatedAccountHash {
		if a.Type() == cs.FormulationAccountType {
			acc := a.(*FormulationAccount)
			addr := acc.Address()
			if err := cs.RankTable.Add(NewRank(addr, acc.KeyHash, phase, hash.DoubleHash(addr[:]))); err != nil {
				return nil, err
			}
		}
	}
	for _, acc := range ctd.DeletedAccountHash {
		if acc.Type() == cs.FormulationAccountType {
			cs.RankTable.Remove(acc.Address())
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
	if err := cs.RankTable.ForwardCandidates(int(bh.TimeoutCount), HeaderHash); err != nil {
		return nil, err
	}
	phase := cs.RankTable.LargestPhase() + 1
	for _, a := range ctd.CreatedAccountHash {
		if a.Type() == cs.FormulationAccountType {
			acc := a.(*FormulationAccount)
			addr := acc.Address()
			if err := cs.RankTable.Add(NewRank(addr, acc.KeyHash, phase, hash.DoubleHash(addr[:]))); err != nil {
				return nil, err
			}
		}
	}
	for _, acc := range ctd.DeletedAccountHash {
		if acc.Type() == cs.FormulationAccountType {
			cs.RankTable.Remove(acc.Address())
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
	members := cs.RankTable.Candidates(int(TimeoutCount) + 1)
	if len(members) == 0 {
		return false, ErrInsufficientCandidateCount
	}
	return members[TimeoutCount].Address.Equal(addr), nil
}

func (cs *Consensus) buildSaveData() ([]byte, error) {
	SaveData := []byte{}
	{
		var buffer bytes.Buffer
		if _, err := cs.RankTable.WriteTo(&buffer); err != nil {
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
	r := bytes.NewReader(SaveData)
	if _, err := cs.RankTable.ReadFrom(r); err != nil {
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
