package chain

import (
	"io"

	"git.fleta.io/fleta/core/transaction"
)

// UTXO TODO
type UTXO struct {
	transaction.TxIn
	transaction.TxOut
}

// WriteTo TODO
func (utxo *UTXO) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := utxo.TxIn.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := utxo.TxOut.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom TODO
func (utxo *UTXO) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := utxo.TxIn.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := utxo.TxOut.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	return read, nil
}
