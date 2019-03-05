package message_def

import (
	"io"

	"github.com/fletaio/common"
	"github.com/fletaio/common/hash"
	"github.com/fletaio/common/util"

	"github.com/fletaio/framework/message"
)

// BlockReqMessage is a message for a block request
type BlockReqMessage struct {
	PrevHash             hash.Hash256
	TargetHeight         uint32
	TimeoutCount         uint32
	Formulator           common.Address
	FormulatorPublicHash common.PublicHash
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
	return read, nil
}
