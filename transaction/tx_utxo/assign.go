package tx_utxo

import (
	"bytes"
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/transaction"
)

// Assign TODO
type Assign struct {
	transaction.Base
	Vin  []*transaction.TxIn  //MAXLEN : 255
	Vout []*transaction.TxOut //MAXLEN : 255
}

// NewAssign TODO
func NewAssign(coord *common.Coordinate, timestamp uint64) *Assign {
	return &Assign{
		Base: transaction.Base{
			Coordinate_: coord.Clone(),
			Timestamp_:  timestamp,
		},
		Vin:  []*transaction.TxIn{},
		Vout: []*transaction.TxOut{},
	}
}

// Hash TODO
func (tx *Assign) Hash() (hash.Hash256, error) {
	var buffer bytes.Buffer
	if _, err := tx.WriteTo(&buffer); err != nil {
		return hash.Hash256{}, err
	}
	return hash.DoubleHash(buffer.Bytes()), nil
}

// WriteTo TODO
func (tx *Assign) WriteTo(w io.Writer) (int64, error) {
	if len(tx.Vin) > 255 {
		return 0, ErrExceedTxInCount
	}
	if len(tx.Vout) > 255 {
		return 0, transaction.ErrExceedTxOutCount
	}

	var wrote int64
	if n, err := tx.Base.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}

	if n, err := util.WriteUint8(w, uint8(len(tx.Vin))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, vin := range tx.Vin {
			if n, err := vin.WriteTo(w); err != nil {
				return wrote, err
			} else {
				wrote += n
			}
		}
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
func (tx *Assign) ReadFrom(r io.Reader) (int64, error) {
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
		tx.Vin = make([]*transaction.TxIn, 0, Len)
		for i := 0; i < int(Len); i++ {
			vin := new(transaction.TxIn)
			if n, err := vin.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				tx.Vin = append(tx.Vin, vin)
			}
		}
	}

	if Len, n, err := util.ReadUint8(r); err != nil {
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
