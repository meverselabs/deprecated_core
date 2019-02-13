package block

import (
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/framework/chain"
)

// Header is validation informations
type Header struct {
	chain.Base
	ChainCoord    common.Coordinate
	LevelRootHash hash.Hash256
	ContextHash   hash.Hash256
	Formulator    common.Address
	TimeoutCount  uint32
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
