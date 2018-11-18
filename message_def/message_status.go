package message_def

import (
	"io"

	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/framework/message"
)

// StatusMessage TODO
type StatusMessage struct {
	Version       uint16
	Height        uint32
	LastBlockHash hash.Hash256
}

// StatusMessageType TODO
var StatusMessageType = message.DefineType("fleta.Status")

// Type TODO
func (b *StatusMessage) Type() message.Type {
	return StatusMessageType
}

// WriteTo TODO
func (b *StatusMessage) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := util.WriteUint16(w, b.Version); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint32(w, b.Height); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := b.LastBlockHash.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom TODO
func (b *StatusMessage) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if v, n, err := util.ReadUint16(r); err != nil {
		return read, err
	} else {
		read += n
		b.Version = v
	}
	if v, n, err := util.ReadUint32(r); err != nil {
		return read, err
	} else {
		read += n
		b.Height = v
	}
	if n, err := b.LastBlockHash.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	return read, nil

}
