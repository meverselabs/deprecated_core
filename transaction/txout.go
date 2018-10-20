package transaction

import (
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/amount"
)

// TxOut TODO
type TxOut struct {
	Amount     *amount.Amount
	PublicHash common.PublicHash
}

// NewTxOut TODO
func NewTxOut() *TxOut {
	out := &TxOut{
		Amount: amount.NewCoinAmount(0, 0),
	}
	return out
}

// Clone TODO
func (out *TxOut) Clone() *TxOut {
	return &TxOut{
		Amount:     out.Amount.Clone(),
		PublicHash: out.PublicHash.Clone(),
	}
}

// WriteTo TODO
func (out *TxOut) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := out.Amount.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := out.PublicHash.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom TODO
func (out *TxOut) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := out.Amount.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := out.PublicHash.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	return read, nil
}