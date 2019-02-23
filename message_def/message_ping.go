package message_def

import (
	"io"

	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/framework/message"
)

// PingMessage is a message for a block generation
type PingMessage struct {
	Timestamp uint64
}

// Type returns the type of the message
func (b *PingMessage) Type() message.Type {
	return PingMessageType
}

// WriteTo is a serialization function
func (b *PingMessage) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := util.WriteUint64(w, b.Timestamp); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom is a deserialization function
func (b *PingMessage) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if v, n, err := util.ReadUint64(r); err != nil {
		return read, err
	} else {
		read += n
		b.Timestamp = v
	}
	return read, nil
}
