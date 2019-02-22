package block

import (
	"encoding/json"
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/framework/chain"
)

// Header is validation informations
type Header struct {
	chain.Base
	ChainCoord    common.Coordinate `json:"chain_coord"`
	LevelRootHash hash.Hash256      `json:"level_root_hash"`
	ContextHash   hash.Hash256      `json:"context_hash"`
	Formulator    common.Address    `json:"formulator"`
	TimeoutCount  uint32            `json:"timeout_count"`
}

// Hash returns the hash value of it
func (bh *Header) Hash() hash.Hash256 {
	return hash.DoubleHashByWriterTo(bh)
}

// WriteTo is a serialization function
func (bh *Header) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := bh.Base.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := bh.ChainCoord.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := bh.LevelRootHash.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := bh.ContextHash.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := bh.Formulator.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint32(w, bh.TimeoutCount); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom is a deserialization function
func (bh *Header) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := bh.Base.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := bh.ChainCoord.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := bh.LevelRootHash.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := bh.ContextHash.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := bh.Formulator.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if v, n, err := util.ReadUint32(r); err != nil {
		return read, err
	} else {
		bh.TimeoutCount = v
		read += n
	}
	return read, nil
}

// UnmarshalJSON is a unmarshaler function
func (bh *Header) UnmarshalJSON(bs []byte) error {
	return json.Unmarshal(bs, &bh)
}

// MarshalJSON is a marshaler function
func (bh *Header) MarshalJSON() ([]byte, error) {
	return json.Marshal(bh)
}
