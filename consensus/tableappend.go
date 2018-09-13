package consensus

import (
	"bytes"
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/consensus/rank"
	"git.fleta.io/fleta/core/consensus/timeout"
)

// TableAppend TODO
type TableAppend struct {
	Height              uint64
	PrevTableAppendHash hash.Hash256
	Ranks               []*rank.Rank
	TailTimeouts        []*timeout.Timeout
}

// Hash TODO
func (msg *TableAppend) Hash() (hash.Hash256, error) {
	var buffer bytes.Buffer
	if _, err := msg.WriteTo(&buffer); err != nil {
		return hash.Hash256{}, err
	}
	return hash.Hash(buffer.Bytes()), nil
}

// WriteTo TODO
func (msg *TableAppend) WriteTo(w io.Writer) (int64, error) {
	if len(msg.TailTimeouts) > 255 {
		return 0, ErrExceedTimeoutCount
	}

	var wrote int64
	if n, err := util.WriteUint64(w, msg.Height); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := msg.PrevTableAppendHash.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint8(w, uint8(len(msg.Ranks))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, pubkey := range msg.Ranks {
			if n, err := pubkey.WriteTo(w); err != nil {
				return wrote, err
			} else {
				wrote += n
			}
		}
	}
	if n, err := util.WriteUint8(w, uint8(len(msg.TailTimeouts))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, to := range msg.TailTimeouts {
			if n, err := to.WriteTo(w); err != nil {
				return wrote, err
			} else {
				wrote += n
			}
		}
	}
	return wrote, nil
}

// ReadFrom TODO
func (msg *TableAppend) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if v, n, err := util.ReadUint64(r); err != nil {
		return read, err
	} else {
		read += n
		msg.Height = v
	}
	if n, err := msg.PrevTableAppendHash.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if Len, n, err := util.ReadUint8(r); err != nil {
		return read, err
	} else {
		read += n
		msg.Ranks = make([]*rank.Rank, 0, Len)
		for i := 0; i < int(Len); i++ {
			rank := new(rank.Rank)
			if n, err := rank.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				msg.Ranks = append(msg.Ranks, rank)
			}
		}
	}
	if Len, n, err := util.ReadUint8(r); err != nil {
		return read, err
	} else {
		read += n
		msg.TailTimeouts = make([]*timeout.Timeout, 0, Len)
		for i := 0; i < int(Len); i++ {
			to := new(timeout.Timeout)
			if n, err := to.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				msg.TailTimeouts = append(msg.TailTimeouts, to)
			}
		}
	}
	return read, nil
}

// SignedTableAppend TODO
type SignedTableAppend struct {
	TableAppend
	Signature common.Signature
}

// Hash TODO
func (msg *SignedTableAppend) Hash() (hash.Hash256, error) {
	var buffer bytes.Buffer
	if _, err := msg.WriteTo(&buffer); err != nil {
		return hash.Hash256{}, err
	}
	return hash.Hash(buffer.Bytes()), nil
}

// WriteTo TODO
func (msg *SignedTableAppend) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := msg.TableAppend.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := msg.Signature.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom TODO
func (msg *SignedTableAppend) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := msg.TableAppend.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := msg.Signature.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	return read, nil
}

// ObserverSignedTableAppend TODO
type ObserverSignedTableAppend struct {
	SignedTableAppend
	ObserverSignatures []common.Signature
}

// Hash TODO
func (msg *ObserverSignedTableAppend) Hash() (hash.Hash256, error) {
	var buffer bytes.Buffer
	if _, err := msg.WriteTo(&buffer); err != nil {
		return hash.Hash256{}, err
	}
	return hash.Hash(buffer.Bytes()), nil
}

// WriteTo TODO
func (msg *ObserverSignedTableAppend) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := msg.SignedTableAppend.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint8(w, uint8(len(msg.ObserverSignatures))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, sig := range msg.ObserverSignatures {
			if n, err := sig.WriteTo(w); err != nil {
				return wrote, err
			} else {
				wrote += n
			}
		}
	}
	return wrote, nil
}

// ReadFrom TODO
func (msg *ObserverSignedTableAppend) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := msg.SignedTableAppend.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if Len, n, err := util.ReadUint8(r); err != nil {
		return read, err
	} else {
		read += n
		msg.ObserverSignatures = make([]common.Signature, 0, Len)
		for i := 0; i < int(Len); i++ {
			var sig common.Signature
			if n, err := sig.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				msg.ObserverSignatures = append(msg.ObserverSignatures, sig)
			}
		}
	}
	return read, nil
}
