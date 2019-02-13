package message_def

import (
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/data"

	"git.fleta.io/fleta/framework/message"
)

// BlockGenMessage TODO
type BlockGenMessage struct {
	RoundHash          hash.Hash256
	Block              *block.Block
	GeneratorSignature common.Signature
	Tran               *data.Transactor
}

// BlockGenMessageType TODO
var BlockGenMessageType = message.DefineType("fleta.BlockGen")

// Type returns the type of the transaction
func (b *BlockGenMessage) Type() message.Type {
	return BlockGenMessageType
}

// WriteTo is a serialization function
func (b *BlockGenMessage) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := b.RoundHash.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
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
	return wrote, nil
}

// ReadFrom is a deserialization function
func (b *BlockGenMessage) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := b.RoundHash.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
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
	return read, nil
}
