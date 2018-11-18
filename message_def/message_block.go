package message_def

import (
	"io"

	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/data"

	"git.fleta.io/fleta/framework/message"
)

// BlockMessage TODO
type BlockMessage struct {
	Block          *block.Block
	ObserverSigned *block.ObserverSigned
	Tran           *data.Transactor
}

// NewBlockMessage TODO
func NewBlockMessage(Tran *data.Transactor) *BlockMessage {
	return &BlockMessage{
		Block:          &block.Block{},
		ObserverSigned: &block.ObserverSigned{},
		Tran:           Tran,
	}
}

// BlockMessageType TODO
var BlockMessageType = message.DefineType("fleta.Block")

// Type TODO
func (b *BlockMessage) Type() message.Type {
	return BlockMessageType
}

// WriteTo TODO
func (b *BlockMessage) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := b.Block.WriteTo(w); err != nil {
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
func (b *BlockMessage) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := b.Block.ReadFromWith(r, b.Tran); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := b.ObserverSigned.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	return read, nil

}
