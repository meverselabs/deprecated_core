package message_def

import (
	"io"

	"github.com/fletaio/framework/message"
)

// PingMessage is a message for a block generation
type PingMessage struct {
}

// Type returns the type of the message
func (b *PingMessage) Type() message.Type {
	return PingMessageType
}

// WriteTo is a serialization function
func (b *PingMessage) WriteTo(w io.Writer) (int64, error) {
	return 0, nil
}

// ReadFrom is a deserialization function
func (b *PingMessage) ReadFrom(r io.Reader) (int64, error) {
	return 0, nil
}
