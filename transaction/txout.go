package transaction

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/fletaio/common"
	"github.com/fletaio/core/amount"
)

// TxOut represents recipient of the UTXO
type TxOut struct {
	Amount     *amount.Amount
	PublicHash common.PublicHash
}

// NewTxOut returns a TxOut
func NewTxOut() *TxOut {
	out := &TxOut{
		Amount: amount.NewCoinAmount(0, 0),
	}
	return out
}

// Clone returns the clonend value of it
func (out *TxOut) Clone() *TxOut {
	return &TxOut{
		Amount:     out.Amount.Clone(),
		PublicHash: out.PublicHash.Clone(),
	}
}

// WriteTo is a serialization function
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

// ReadFrom is a deserialization function
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

// MarshalJSON is a marshaler function
func (tx *TxOut) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString(`{`)
	buffer.WriteString(`"amount":`)
	if bs, err := tx.Amount.MarshalJSON(); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`,`)
	buffer.WriteString(`"public_hash":`)
	if bs, err := json.Marshal(tx.PublicHash); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`}`)
	return buffer.Bytes(), nil
}
