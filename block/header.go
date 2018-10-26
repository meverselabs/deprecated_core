package block

import (
	"bytes"
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
)

// Header is validation informations
type Header struct {
	ChainCoord         common.Coordinate
	Height             uint32
	Version            uint16
	HashPrevBlock      hash.Hash256
	HashLevelRoot      hash.Hash256
	Timestamp          uint64
	FormulationAddress common.Address
	TimeoutCount       uint32
}

// Hash retuns the hash value of it
func (bh *Header) Hash() hash.Hash256 {
	var buffer bytes.Buffer
	if _, err := bh.WriteTo(&buffer); err != nil {
		panic(err)
	}
	return hash.DoubleHash(buffer.Bytes())
}

// WriteTo is a serialization function
func (bh *Header) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := bh.ChainCoord.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint32(w, bh.Height); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint16(w, bh.Version); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := bh.HashPrevBlock.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := bh.HashLevelRoot.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint64(w, bh.Timestamp); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := bh.FormulationAddress.WriteTo(w); err != nil {
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
	if n, err := bh.ChainCoord.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if v, n, err := util.ReadUint32(r); err != nil {
		return read, err
	} else {
		bh.Height = v
		read += n
	}
	if v, n, err := util.ReadUint16(r); err != nil {
		return read, err
	} else {
		bh.Version = v
		read += n
	}
	if n, err := bh.HashPrevBlock.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := bh.HashLevelRoot.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if v, n, err := util.ReadUint64(r); err != nil {
		return read, err
	} else {
		bh.Timestamp = v
		read += n
	}
	if n, err := bh.FormulationAddress.ReadFrom(r); err != nil {
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
