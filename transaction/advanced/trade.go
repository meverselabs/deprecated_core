package advanced

import (
	"bytes"
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/transaction"
)

// Trade TODO
type Trade struct {
	transaction.Base
	From common.Address
	Vout []*transaction.TxOut //MAXLEN : 65535
}

// NewTrade TODO
func NewTrade(version uint16, timestamp uint64) *Trade {
	return &Trade{
		Base: transaction.Base{
			Version_:   version,
			Timestamp_: timestamp,
		},
		Vout: []*transaction.TxOut{},
	}
}

// Hash TODO
func (tx *Trade) Hash() (hash.Hash256, error) {
	var buffer bytes.Buffer
	if _, err := tx.WriteTo(&buffer); err != nil {
		return hash.Hash256{}, err
	}
	return hash.DoubleHash(buffer.Bytes()), nil
}

// WriteTo TODO
func (tx *Trade) WriteTo(w io.Writer) (int64, error) {
	if len(tx.Vout) > 65535 {
		return 0, ErrExceedTxOutCount
	}

	var wrote int64
	if n, err := tx.Base.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := tx.From.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}

	if n, err := util.WriteUint16(w, uint16(len(tx.Vout))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, vout := range tx.Vout {
			if n, err := vout.WriteTo(w); err != nil {
				return wrote, err
			} else {
				wrote += n
			}
		}
	}
	return wrote, nil
}

// ReadFrom TODO
func (tx *Trade) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := tx.Base.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := tx.From.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}

	if Len, n, err := util.ReadUint16(r); err != nil {
		return read, err
	} else {
		read += n
		tx.Vout = make([]*transaction.TxOut, 0, Len)
		for i := 0; i < int(Len); i++ {
			vout := transaction.NewTxOut()
			if n, err := vout.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				tx.Vout = append(tx.Vout, vout)
			}
		}
	}
	return read, nil
}
