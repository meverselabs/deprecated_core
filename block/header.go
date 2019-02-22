package block

import (
	"bytes"
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

// MarshalJSON is a marshaler function
func (bh *Header) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString(`{`)
	buffer.WriteString(`"version":`)
	if bs, err := json.Marshal(bh.Version_); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`,`)
	buffer.WriteString(`"height":`)
	if bs, err := json.Marshal(bh.Height_); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`,`)
	buffer.WriteString(`"prev_hash":`)
	if bs, err := bh.PrevHash_.MarshalJSON(); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`,`)
	buffer.WriteString(`"timestamp":`)
	if bs, err := json.Marshal(bh.Timestamp_); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`,`)
	buffer.WriteString(`"chain_coord":`)
	if bs, err := bh.ChainCoord.MarshalJSON(); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`,`)
	buffer.WriteString(`"level_root_hash":`)
	if bs, err := bh.LevelRootHash.MarshalJSON(); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`,`)
	buffer.WriteString(`"context_hash":`)
	if bs, err := bh.ContextHash.MarshalJSON(); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`,`)
	buffer.WriteString(`"formulator":`)
	if bs, err := bh.Formulator.MarshalJSON(); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`,`)
	buffer.WriteString(`"timeout_count":`)
	if bs, err := json.Marshal(bh.TimeoutCount); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`}`)
	return buffer.Bytes(), nil
}
