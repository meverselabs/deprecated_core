package message_def

import (
	"io"

	"github.com/fletaio/common"
	"github.com/fletaio/common/util"
	"github.com/fletaio/core/block"
	"github.com/fletaio/core/data"

	"github.com/fletaio/framework/message"
)

// BlockGenMessage is a message for a block generation
type BlockGenMessage struct {
	Block              *block.Block
	GeneratorSignature common.Signature
	IsReply            bool
	Tran               *data.Transactor
}

// Type returns the type of the message
func (b *BlockGenMessage) Type() message.Type {
	return BlockGenMessageType
}

// WriteTo is a serialization function
func (b *BlockGenMessage) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := b.Block.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := b.GeneratorSignature.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteBool(w, b.IsReply); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom is a deserialization function
func (b *BlockGenMessage) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := b.Block.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := b.GeneratorSignature.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if v, n, err := util.ReadBool(r); err != nil {
		return read, err
	} else {
		read += n
		b.IsReply = v
	}
	return read, nil
}
