package tx_account

import (
	"bytes"
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/amount"
	"git.fleta.io/fleta/core/transaction"
)

// Transfer TODO
type Transfer struct {
	transaction.Base
	Seq  uint64
	From common.Address //MAXLEN : 255
	Vout []*TransferOut //MAXLEN : 255
}

// NewTransfer TODO
func NewTransfer(coord *common.Coordinate, timestamp uint64, seq uint64) *Transfer {
	return &Transfer{
		Base: transaction.Base{
			Coordinate_: coord.Clone(),
			Timestamp_:  timestamp,
		},
		Seq:  seq,
		Vout: []*TransferOut{},
	}
}

// Hash TODO
func (tx *Transfer) Hash() (hash.Hash256, error) {
	var buffer bytes.Buffer
	if _, err := tx.WriteTo(&buffer); err != nil {
		return hash.Hash256{}, err
	}
	return hash.DoubleHash(buffer.Bytes()), nil
}

// WriteTo TODO
func (tx *Transfer) WriteTo(w io.Writer) (int64, error) {
	if len(tx.Vout) > 255 {
		return 0, ErrExceedTransferOutCount
	}

	var wrote int64
	if n, err := tx.Base.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint64(w, tx.Seq); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := tx.From.WriteTo(w); err != nil {
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
func (tx *Transfer) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := tx.Base.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if v, n, err := util.ReadUint64(r); err != nil {
		return read, err
	} else {
		read += n
		tx.Seq = v
	}
	if n, err := tx.From.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}

	if Len, n, err := util.ReadUint8(r); err != nil {
		return read, err
	} else {
		read += n
		tx.Vout = make([]*TransferOut, 0, Len)
		for i := 0; i < int(Len); i++ {
			vout := NewTransferOut()
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

// TransferOut TODO
type TransferOut struct {
	Amount  *amount.Amount
	Address common.Address
}

// NewTransferOut TODO
func NewTransferOut() *TransferOut {
	out := &TransferOut{
		Amount: amount.NewCoinAmount(0, 0),
	}
	return out
}

// WriteTo TODO
func (out *TransferOut) WriteTo(w io.Writer) (int64, error) {
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
func (out *TransferOut) ReadFrom(r io.Reader) (int64, error) {
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
