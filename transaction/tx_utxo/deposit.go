package tx_utxo

import (
	"bytes"
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/core/amount"
	"git.fleta.io/fleta/core/transaction"
)

// Deposit TODO
type Deposit struct {
	Assign
	Amount  *amount.Amount
	Address common.Address
}

// NewDeposit TODO
func NewDeposit(coord *common.Coordinate, timestamp uint64) *Deposit {
	return &Deposit{
		Assign: Assign{
			Base: transaction.Base{
				Coordinate_: coord.Clone(),
				Timestamp_:  timestamp,
			},
			Vin:  []*transaction.TxIn{},
			Vout: []*transaction.TxOut{},
		},
		Amount: amount.NewCoinAmount(0, 0),
	}
}

// Hash TODO
func (tx *Deposit) Hash() (hash.Hash256, error) {
	var buffer bytes.Buffer
	if _, err := tx.WriteTo(&buffer); err != nil {
		return hash.Hash256{}, err
	}
	return hash.DoubleHash(buffer.Bytes()), nil
}

// WriteTo TODO
func (tx *Deposit) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := tx.Assign.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := tx.Amount.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := tx.Address.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom TODO
func (tx *Deposit) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := tx.Assign.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := tx.Amount.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := tx.Address.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	return read, nil
}
