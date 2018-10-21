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

// Consensus TODO
type Consensus struct {
	RankTable              *RankTable
	ObserverSignatureHash  map[common.PublicHash]bool
	FormulationAccountType account.Type
}

// NewConsensus TODO
func NewConsensus(ObserverSignatureHash map[common.PublicHash]bool, FormulationAccountType account.Type) *Consensus {
	cs := &Consensus{
		RankTable:              NewRankTable(),
		ObserverSignatureHash:  ObserverSignatureHash,
		FormulationAccountType: FormulationAccountType,
	}
	return cs
}

// ValidateBlock TODO
func (cs *Consensus) ValidateBlock(b *block.Block, s *block.ObserverSigned, PrevHeight uint32, LastBlockHash hash.Hash256) error {
	if b.Header.Height != PrevHeight+1 {
		return ErrInvalidPrevBlockHeight
	}
	if !b.Header.HashPrevBlock.Equal(LastBlockHash) {
		return ErrInvalidPrevBlockHash
	}
	members := cs.RankTable.Candidates(int(b.Header.TimeoutCount) + 1)
	if members == nil {
		return ErrInsufficientCandidateCount
	}
	Top := members[b.Header.TimeoutCount]
	if !Top.Address.Equal(b.Header.FormulationAddress) {
		return ErrInvalidTopMember
	}
	blockHash := b.Header.Hash()
	pubkey, err := common.RecoverPubkey(blockHash, s.GeneratorSignature)
	if err != nil {
		return err
	}
	pubhash := common.NewPublicHash(pubkey)
	if !Top.PublicHash.Equal(pubhash) {
		return ErrInvalidTopSignature
	}
	if err := cs.ValidateObserverSignatures(blockHash, s.ObserverSignatures); err != nil {
		return err
	}
	return nil
}

// ValidateObserverSignatures TODO
func (cs *Consensus) ValidateObserverSignatures(blockHash hash.Hash256, sigs []common.Signature) error {
	if len(sigs) != len(cs.ObserverSignatureHash)/2+1 {
		return ErrInsufficientObserverSignature
	}
	sigHash := map[common.PublicHash]bool{}
	for _, sig := range sigs {
		pubkey, err := common.RecoverPubkey(blockHash, sig)
		if err != nil {
			return err
		}
		pubhash := common.NewPublicHash(pubkey)
		if !cs.ObserverSignatureHash[pubhash] {
			return ErrInvalidObserverSignature
		}
		sigHash[pubhash] = true
	}
	if len(sigHash) != len(sigs) {
		return ErrDuplicatedObserverSignature
	}
	return nil
}

// ApplyGenesis TODO
func (cs *Consensus) ApplyGenesis(ctd *data.ContextData) ([]byte, error) {
	phase := cs.RankTable.LargestPhase() + 1
	for _, a := range ctd.CreatedAccountHash {
		if a.Type() == cs.FormulationAccountType {
			acc := a.(*Account)
			addr := acc.Address()
			if err := cs.RankTable.Add(NewRank(addr, acc.KeyHash, phase, hash.Hash(addr[:]))); err != nil {
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

// ApplyBlock TODO
func (cs *Consensus) ApplyBlock(ctd *data.ContextData, b *block.Block) ([]byte, error) {
	if err := cs.RankTable.ForwardCandidates(int(b.Header.TimeoutCount), b.Header.Hash()); err != nil {
		return nil, err
	}
	phase := cs.RankTable.LargestPhase() + 1
	for _, a := range ctd.CreatedAccountHash {
		if a.Type() == cs.FormulationAccountType {
			acc := a.(*Account)
			addr := acc.Address()
			if err := cs.RankTable.Add(NewRank(addr, acc.KeyHash, phase, hash.Hash(addr[:]))); err != nil {
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

// IsMinable TODO
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
		if _, err := util.WriteUint8(&buffer, uint8(len(cs.ObserverSignatureHash))); err != nil {
			return nil, err
		}
		for k := range cs.ObserverSignatureHash {
			if _, err := k.WriteTo(&buffer); err != nil {
				return nil, err
			}
		}
		SaveData = append(SaveData, buffer.Bytes()...)
	}
	return SaveData, nil
}

// LoadFromSaveData TODO
func (cs *Consensus) LoadFromSaveData(SaveData []byte) error {
	r := bytes.NewReader(SaveData)
	if _, err := cs.RankTable.ReadFrom(r); err != nil {
		return err
	}
	ObserverSignatureHash := map[common.PublicHash]bool{}
	if Len, _, err := util.ReadUint8(r); err != nil {
		return err
	} else {
		for i := 0; i < int(Len); i++ {
			var pubhash common.PublicHash
			if _, err := pubhash.ReadFrom(r); err != nil {
				return err
			}
			ObserverSignatureHash[pubhash] = true
		}
	}
	cs.ObserverSignatureHash = ObserverSignatureHash
	return nil
}
