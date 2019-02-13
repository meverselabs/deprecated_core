package message_def

import (
	"io"

	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/block"

	"git.fleta.io/fleta/framework/message"
)

// BlockObSignMessage TODO
type BlockObSignMessage struct {
	TargetHeight   uint32
	ObserverSigned *block.ObserverSigned
}

// BlockObSignMessageType TODO
var BlockObSignMessageType = message.DefineType("fleta.BlockObSign")

// Type TODO
func (b *BlockObSignMessage) Type() message.Type {
	return BlockObSignMessageType
}

// WriteTo TODO
func (b *BlockObSignMessage) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := util.WriteUint32(w, b.TargetHeight); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := b.ObserverSigned.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom TODO
func (b *BlockObSignMessage) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if v, n, err := util.ReadUint32(r); err != nil {
		return read, err
	} else {
		read += n
		b.TargetHeight = v
	}
	if n, err := b.ObserverSigned.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	return read, nil
}
