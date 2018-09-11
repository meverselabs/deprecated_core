package advanced

import (
	"bytes"
	"encoding/json"
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/transaction"
)

// Formulation TODO
type Formulation struct {
	Version_   uint16
	Timestamp_ uint64
	PublicKey  common.PublicKey
	Vin        []*transaction.TxIn //MAXLEN : 65535
}

// FormulationTransaction TODO
func FormulationTransaction(version uint16, timestamp uint64, PublicKey common.PublicKey) *Formulation {
	return &Formulation{
		Version_:   version,
		Timestamp_: timestamp,
		PublicKey:  PublicKey,
		Vin:        []*transaction.TxIn{},
	}
}

// Version TODO
func (tx *Formulation) Version() uint16 {
	return tx.Version_
}

// Timestamp TODO
func (tx *Formulation) Timestamp() uint64 {
	return tx.Timestamp_
}

// AppendVin TODO
func (tx *Formulation) AppendVin(op *transaction.TxIn) {
	tx.Vin = append(tx.Vin, op)
}

// Hash TODO
func (tx *Formulation) Hash() (hash.Hash256, error) {
	var buffer bytes.Buffer
	if _, err := tx.WriteTo(&buffer); err != nil {
		return hash.Hash256{}, err
	}
	return hash.DoubleHash(buffer.Bytes()), nil
}

// WriteTo TODO
func (tx *Formulation) WriteTo(w io.Writer) (int64, error) {
	if len(tx.Vin) > 65535 {
		return 0, transaction.ErrExceedTransactionCount
	}

	var wrote int64
	if n, err := util.WriteUint16(w, tx.Version_); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint64(w, tx.Timestamp_); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := tx.PublicKey.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}

	if n, err := util.WriteUint16(w, uint16(len(tx.Vin))); err != nil {
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
	return wrote, nil
}

// ReadFrom TODO
func (tx *Formulation) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if v, n, err := util.ReadUint16(r); err != nil {
		return read, err
	} else {
		read += n
		tx.Version_ = v
	}
	if v, n, err := util.ReadUint64(r); err != nil {
		return read, err
	} else {
		read += n
		tx.Timestamp_ = v
	}
	if n, err := tx.PublicKey.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}

	if Len, n, err := util.ReadUint16(r); err != nil {
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
	return read, nil
}

// MarshalJSON TODO
func (tx *Formulation) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	enc := json.NewEncoder(&buffer)
	if err := enc.Encode(map[string]interface{}{
		"Version":   tx.Version_,
		"Timestamp": tx.Timestamp_,
		"PublicKey": tx.PublicKey,
		"Vin":       tx.Vin,
	}); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// Debug TODO
func (tx *Formulation) Debug() (string, error) {
	if bs, err := tx.MarshalJSON(); err != nil {
		return "", err
	} else {
		return string(bs), err
	}
}
