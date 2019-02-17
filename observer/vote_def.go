package observer

import (
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/framework/chain"
)

type RoundVote struct {
	RoundHash            hash.Hash256
	PrevRoundHash        hash.Hash256
	ChainCoord           *common.Coordinate
	PrevHash             hash.Hash256
	TargetHeight         uint32
	TimeoutCount         uint32
	Formulator           common.Address
	FormulatorPublicHash common.PublicHash
}

// Hash returns the hash value of it
func (vt *RoundVote) Hash() hash.Hash256 {
	return hash.DoubleHashByWriterTo(vt)
}

// WriteTo is a serialization function
func (vt *RoundVote) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := vt.RoundHash.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := vt.PrevRoundHash.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := vt.ChainCoord.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := vt.PrevHash.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint32(w, vt.TargetHeight); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint32(w, vt.TimeoutCount); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := vt.Formulator.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := vt.FormulatorPublicHash.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom is a deserialization function
func (vt *RoundVote) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := vt.RoundHash.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := vt.PrevRoundHash.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := vt.ChainCoord.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := vt.PrevHash.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if v, n, err := util.ReadUint32(r); err != nil {
		return read, err
	} else {
		read += n
		vt.TargetHeight = v
	}
	if v, n, err := util.ReadUint32(r); err != nil {
		return read, err
	} else {
		read += n
		vt.TimeoutCount = v
	}
	if n, err := vt.Formulator.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := vt.FormulatorPublicHash.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	return read, nil
}

// RoundVoteAck is a message for a round vote ack
type RoundVoteAck struct {
	RoundHash            hash.Hash256
	TimeoutCount         uint32
	Formulator           common.Address
	FormulatorPublicHash common.PublicHash
}

// Hash returns the hash value of it
func (vt *RoundVoteAck) Hash() hash.Hash256 {
	return hash.DoubleHashByWriterTo(vt)
}

// WriteTo is a serialization function
func (vt *RoundVoteAck) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := vt.RoundHash.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint32(w, vt.TimeoutCount); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := vt.Formulator.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := vt.FormulatorPublicHash.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom is a deserialization function
func (vt *RoundVoteAck) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := vt.RoundHash.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if v, n, err := util.ReadUint32(r); err != nil {
		return read, err
	} else {
		read += n
		vt.TimeoutCount = v
	}
	if n, err := vt.Formulator.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := vt.FormulatorPublicHash.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	return read, nil
}

// BlockVote is message for a block vote
type BlockVote struct {
	RoundHash          hash.Hash256
	Header             chain.Header
	GeneratorSignature common.Signature
	ObserverSignature  common.Signature
}

// Hash returns the hash value of it
func (vt *BlockVote) Hash() hash.Hash256 {
	return hash.DoubleHashByWriterTo(vt)
}

// WriteTo is a serialization function
func (vt *BlockVote) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := vt.RoundHash.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := vt.Header.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := vt.GeneratorSignature.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := vt.ObserverSignature.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom is a deserialization function
func (vt *BlockVote) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := vt.RoundHash.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := vt.Header.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := vt.GeneratorSignature.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := vt.ObserverSignature.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	return read, nil
}

type BlockVoteEnd struct {
	RoundHash  hash.Hash256
	BlockVotes []*BlockVote
	Signatures []common.Signature
}

// Hash returns the hash value of it
func (vt *BlockVoteEnd) Hash() hash.Hash256 {
	return hash.DoubleHashByWriterTo(vt)
}

// WriteTo is a serialization function
func (vt *BlockVoteEnd) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := vt.RoundHash.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint32(w, uint32(len(vt.BlockVotes))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, vt := range vt.BlockVotes {
			if n, err = vt.WriteTo(w); err != nil {
				return wrote, err
			} else {
				wrote += n
			}
		}
	}
	if n, err := util.WriteUint32(w, uint32(len(vt.Signatures))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, s := range vt.Signatures {
			if n, err = s.WriteTo(w); err != nil {
				return wrote, err
			} else {
				wrote += n
			}
		}
	}
	return wrote, nil
}

// ReadFrom is a deserialization function
func (vt *BlockVoteEnd) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := vt.RoundHash.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if Len, n, err := util.ReadUint32(r); err != nil {
		return read, err
	} else {
		read += n
		vt.BlockVotes = []*BlockVote{}
		for i := 0; i < int(Len); i++ {
			v := &BlockVote{}
			if n, err := v.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				vt.BlockVotes = append(vt.BlockVotes, v)
			}
		}
	}
	if Len, n, err := util.ReadUint32(r); err != nil {
		return read, err
	} else {
		read += n
		vt.Signatures = []common.Signature{}
		for i := 0; i < int(Len); i++ {
			s := common.Signature{}
			if n, err := s.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				vt.Signatures = append(vt.Signatures, s)
			}
		}
	}
	return read, nil
}

type BlockVoteAck struct {
	RoundHash          hash.Hash256
	ObserverSignatures []common.Signature
}

// Hash returns the hash value of it
func (vt *BlockVoteAck) Hash() hash.Hash256 {
	return hash.DoubleHashByWriterTo(vt)
}

// WriteTo is a serialization function
func (vt *BlockVoteAck) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := vt.RoundHash.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}

	if n, err := util.WriteUint8(w, uint8(len(vt.ObserverSignatures))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, sig := range vt.ObserverSignatures {
			wrote += n
			if n, err := sig.WriteTo(w); err != nil {
				return wrote, err
			} else {
				wrote += n
			}
		}
	}
	return wrote, nil
}

// ReadFrom is a deserialization function
func (vt *BlockVoteAck) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := vt.RoundHash.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}

	if Len, n, err := util.ReadUint8(r); err != nil {
		return read, err
	} else {
		read += n
		vt.ObserverSignatures = make([]common.Signature, 0, Len)
		for i := 0; i < int(Len); i++ {
			var sig common.Signature
			if n, err := sig.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				vt.ObserverSignatures = append(vt.ObserverSignatures, sig)
			}
		}
	}
	return read, nil
}

type RoundFailVote struct {
	RoundHash     hash.Hash256
	PrevRoundHash hash.Hash256
	ChainCoord    *common.Coordinate
	PrevHash      hash.Hash256
	TargetHeight  uint32
}

// Hash returns the hash value of it
func (vt *RoundFailVote) Hash() hash.Hash256 {
	return hash.DoubleHashByWriterTo(vt)
}

// WriteTo is a serialization function
func (vt *RoundFailVote) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := vt.RoundHash.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := vt.PrevRoundHash.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := vt.ChainCoord.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := vt.PrevHash.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint32(w, vt.TargetHeight); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom is a deserialization function
func (vt *RoundFailVote) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := vt.RoundHash.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := vt.PrevRoundHash.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := vt.ChainCoord.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := vt.PrevHash.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if v, n, err := util.ReadUint32(r); err != nil {
		return read, err
	} else {
		read += n
		vt.TargetHeight = v
	}
	return read, nil
}
