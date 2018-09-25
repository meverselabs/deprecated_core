package tx_utxo

import (
	"bytes"
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/core/transaction"
)

// OpenAccount TODO
type OpenAccount struct {
	Assign
	KeyHash common.PublicHash
}

// NewOpenAccount TODO
func NewOpenAccount(coord *common.Coordinate, timestamp uint64) *OpenAccount {
	return &OpenAccount{
		Assign: Assign{
			Base: transaction.Base{
				Coordinate_: coord.Clone(),
				Timestamp_:  timestamp,
			},
			Vin:  []*transaction.TxIn{},
			Vout: []*transaction.TxOut{},
		},
	}
}

// Hash TODO
func (tx *OpenAccount) Hash() (hash.Hash256, error) {
	var buffer bytes.Buffer
	if _, err := tx.WriteTo(&buffer); err != nil {
		return hash.Hash256{}, err
	}
	return hash.DoubleHash(buffer.Bytes()), nil
}

// WriteTo TODO
func (tx *OpenAccount) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := tx.Assign.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := tx.KeyHash.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom TODO
func (tx *OpenAccount) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := tx.Assign.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := tx.KeyHash.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	return read, nil
}
