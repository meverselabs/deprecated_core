package message_def

import (
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"

	"git.fleta.io/fleta/framework/chain"
	"git.fleta.io/fleta/framework/message"
)

// BlockReqMessage is a message for a block request
type BlockReqMessage struct {
	PrevHash             hash.Hash256
	TargetHeight         uint32
	TimeoutCount         uint32
	Formulator           common.Address
	FormulatorPublicHash common.PublicHash
	PrevData             *chain.Data
}

// Type returns the type of the message
func (b *BlockReqMessage) Type() message.Type {
	return BlockReqMessageType
}

// WriteTo is a serialization function
func (b *BlockReqMessage) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := b.PrevHash.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint32(w, b.TargetHeight); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint32(w, b.TimeoutCount); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := b.Formulator.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := b.FormulatorPublicHash.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	HasPrevData := (b.PrevData != nil)
	if n, err := util.WriteBool(w, HasPrevData); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if HasPrevData {
		if n, err := b.PrevData.WriteTo(w); err != nil {
			return wrote, err
		} else {
			wrote += n
		}
	}
	return wrote, nil
}

// ReadFrom is a deserialization function
func (b *BlockReqMessage) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := b.PrevHash.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if v, n, err := util.ReadUint32(r); err != nil {
		return read, err
	} else {
		read += n
		b.TargetHeight = v
	}
	if v, n, err := util.ReadUint32(r); err != nil {
		return read, err
	} else {
		read += n
		b.TimeoutCount = v
	}
	if n, err := b.Formulator.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := b.FormulatorPublicHash.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if v, n, err := util.ReadBool(r); err != nil {
		return read, err
	} else {
		read += n
		if v {
			if n, err := b.PrevData.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
			}
		}
	}
	return read, nil
}
