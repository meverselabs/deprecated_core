package observer

import (
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/framework/message"
)

// message types
var (
	RoundVoteMessageType    = message.DefineType("observer.RoundVote")
	RoundVoteAckMessageType = message.DefineType("observer.RoundVoteAck")
	BlockVoteMessageType    = message.DefineType("observer.BlockVote")
	BatchRequestMessageType = message.DefineType("observer.BatchRequest")
)

// RoundVoteMessage is a message for a round vote
type RoundVoteMessage struct {
	RoundVote *RoundVote
	Signature common.Signature
}

// Type is a type of the message
func (msg *RoundVoteMessage) Type() message.Type {
	return RoundVoteMessageType
}

// WriteTo is a serialization function
func (msg *RoundVoteMessage) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := msg.RoundVote.WriteTo(w); err != nil {
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

// ReadFrom is a deserialization function
func (msg *RoundVoteMessage) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := msg.RoundVote.ReadFrom(r); err != nil {
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

// RoundVoteAckMessage is a message for a round vote ack
type RoundVoteAckMessage struct {
	RoundVoteAck *RoundVoteAck
	Signature    common.Signature
}

// Type returns a type of the message
func (msg *RoundVoteAckMessage) Type() message.Type {
	return RoundVoteAckMessageType
}

// WriteTo is a serialization function
func (msg *RoundVoteAckMessage) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := msg.RoundVoteAck.WriteTo(w); err != nil {
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

// ReadFrom is a deserialization function
func (msg *RoundVoteAckMessage) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := msg.RoundVoteAck.ReadFrom(r); err != nil {
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

// BlockVoteMessage is a message for a block vote
type BlockVoteMessage struct {
	BlockVote *BlockVote
	Signature common.Signature
}

// Type returns a type of the message
func (msg *BlockVoteMessage) Type() message.Type {
	return BlockVoteMessageType
}

// WriteTo is a serialization function
func (msg *BlockVoteMessage) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := msg.BlockVote.WriteTo(w); err != nil {
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

// ReadFrom is a deserialization function
func (msg *BlockVoteMessage) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := msg.BlockVote.ReadFrom(r); err != nil {
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
