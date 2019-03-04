package message_def

import (
	"io"

	"github.com/fletaio/common"
	"github.com/fletaio/common/util"
	"github.com/fletaio/core/data"
	"github.com/fletaio/core/transaction"
	"github.com/fletaio/framework/message"
)

// TransactionMessage is a message for a transaction
type TransactionMessage struct {
	Tx   transaction.Transaction
	Sigs []common.Signature
	Tran *data.Transactor
}

// Type returns the type of the message
func (b *TransactionMessage) Type() message.Type {
	return TransactionMessageType
}

// WriteTo is a serialization function
func (b *TransactionMessage) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := util.WriteUint8(w, uint8(b.Tx.Type())); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := b.Tx.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint32(w, uint32(len(b.Sigs))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, s := range b.Sigs {
			if n, err = s.WriteTo(w); err != nil {
				return wrote, err
			} else {
				wrote += n
			}
		}
	}
	return wrote, nil
}

// ReadFrom is a deserialization function
func (b *TransactionMessage) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if v, n, err := util.ReadUint8(r); err != nil {
		return read, err
	} else {
		read += n
		t := transaction.Type(v)
		tx, err := b.Tran.NewByType(t)
		if err != nil {
			return read, err
		} else {
			if n, err := tx.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				b.Tx = tx
			}
		}
	}
	if Len, n, err := util.ReadUint32(r); err != nil {
		return read, err
	} else {
		read += n
		b.Sigs = []common.Signature{}
		for i := 0; i < int(Len); i++ {
			s := common.Signature{}
			if n, err := s.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				b.Sigs = append(b.Sigs, s)
			}
		}
	}
	return read, nil
}
