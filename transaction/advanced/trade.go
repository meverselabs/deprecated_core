package advanced

import (
	"bytes"
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/amount"
	"git.fleta.io/fleta/core/transaction"
)

// Trade TODO
type Trade struct {
	transaction.Base
	Vout []*TradeOut //MAXLEN : 255
}

// NewTrade TODO
func NewTrade(coord *common.Coordinate, timestamp uint64, seq uint64) *Trade {
	return &Trade{
		Base: transaction.Base{
			Coordinate_: coord.Clone(),
			Timestamp_:  timestamp,
			Seq_:        seq,
		},
		Vout: []*TradeOut{},
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
	if len(tx.Vout) > 255 {
		return 0, ErrExceedTradeOutCount
	}

	var wrote int64
	if n, err := tx.Base.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}

	if n, err := util.WriteUint8(w, uint8(len(tx.Vout))); err != nil {
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

	if Len, n, err := util.ReadUint8(r); err != nil {
		return read, err
	} else {
		read += n
		tx.Vout = make([]*TradeOut, 0, Len)
		for i := 0; i < int(Len); i++ {
			vout := NewTradeOut()
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

// TradeOut TODO
type TradeOut struct {
	Amount  *amount.Amount
	Address common.Address
}

// NewTradeOut TODO
func NewTradeOut() *TradeOut {
	out := &TradeOut{
		Amount: amount.NewCoinAmount(0, 0),
	}
	return out
}

// WriteTo TODO
func (out *TradeOut) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := out.Amount.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := out.Address.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom TODO
func (out *TradeOut) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := out.Amount.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := out.Address.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	return read, nil
}
