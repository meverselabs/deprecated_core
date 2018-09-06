package block

import (
	"bytes"
	"io"

	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
)

// Header TODO
type Header struct {
	Version       uint16
	HashPrevBlock hash.Hash256
	HashLevelRoot hash.Hash256
	Timestamp     uint64
}

// Hash TODO
func (bh *Header) Hash() (hash.Hash256, error) {
	var buffer bytes.Buffer
	if _, err := bh.WriteTo(&buffer); err != nil {
		return hash.Hash256{}, err
	}
	return hash.DoubleHash(buffer.Bytes()), nil
}

// WriteTo TODO
func (bh *Header) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
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
	return wrote, nil
}

// ReadFrom TODO
func (bh *Header) ReadFrom(r io.Reader) (int64, error) {
	var read int64
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
	return read, nil
}