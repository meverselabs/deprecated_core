package block

import (
	"bytes"
	"io"

	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/consensus"
	"git.fleta.io/fleta/core/consensus/timeout"
)

// Header TODO
type Header struct {
	Height              uint32
	Version             uint16
	HashPrevBlock       hash.Hash256
	HashLevelRoot       hash.Hash256
	Timestamp           uint64
	HeadTimeouts        []*timeout.Timeout
	TableAppendMessages []*consensus.SignedTableAppend
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
	if len(bh.HeadTimeouts) > 255 {
		return 0, ErrExceedTimeoutCount
	}

	var wrote int64
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

	if n, err := util.WriteUint8(w, uint8(len(bh.HeadTimeouts))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, to := range bh.HeadTimeouts {
			if n, err := to.WriteTo(w); err != nil {
				return wrote, err
			} else {
				wrote += n
			}
		}
	}

	if n, err := util.WriteUint8(w, uint8(len(bh.TableAppendMessages))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, tm := range bh.TableAppendMessages {
			if n, err := tm.WriteTo(w); err != nil {
				return wrote, err
			} else {
				wrote += n
			}
		}
	}
	return wrote, nil
}

// ReadFrom TODO
func (bh *Header) ReadFrom(r io.Reader) (int64, error) {
	var read int64
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

	if Len, n, err := util.ReadUint8(r); err != nil {
		return read, err
	} else {
		read += n
		bh.HeadTimeouts = make([]*timeout.Timeout, 0, Len)
		for i := 0; i < int(Len); i++ {
			to := new(timeout.Timeout)
			if n, err := to.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				bh.HeadTimeouts = append(bh.HeadTimeouts, to)
			}
		}
	}

	if Len, n, err := util.ReadUint8(r); err != nil {
		return read, err
	} else {
		read += n
		bh.TableAppendMessages = make([]*consensus.SignedTableAppend, 0, Len)
		for i := 0; i < int(Len); i++ {
			tm := new(consensus.SignedTableAppend)
			if n, err := tm.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				bh.TableAppendMessages = append(bh.TableAppendMessages, tm)
			}
		}
	}
	return read, nil
}
