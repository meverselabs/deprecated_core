package transaction

import (
	"bytes"
	"encoding/json"
	"io"

	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
)

// Transaction TODO
type Transaction interface {
	io.WriterTo
	io.ReaderFrom
	Version() uint16
	Timestamp() uint64
	Vin() []*TxIn
	Vout() []*TxOut
	Hash() (hash.Hash256, error)
	Debug() (string, error)
}

// Base TODO
type Base struct {
	Version_   uint16
	Timestamp_ uint64
	Vin_       []*TxIn  //MAXLEN : 65535
	Vout_      []*TxOut //MAXLEN : 65535
}

// NewBase TODO
func NewBase(version uint16, timestamp uint64) *Base {
	return &Base{
		Version_:   version,
		Timestamp_: timestamp,
		Vin_:       []*TxIn{},
		Vout_:      []*TxOut{},
	}
}

// Version TODO
func (tx *Base) Version() uint16 {
	return tx.Version_
}

// Timestamp TODO
func (tx *Base) Timestamp() uint64 {
	return tx.Timestamp_
}

// SetTimestamp TODO
func (tx *Base) SetTimestamp(t uint64) {
	tx.Timestamp_ = t
}

// Vin TODO
func (tx *Base) Vin() []*TxIn {
	return tx.Vin_
}

// AppendVin TODO
func (tx *Base) AppendVin(op *TxIn) {
	tx.Vin_ = append(tx.Vin_, op)
}

// Vout TODO
func (tx *Base) Vout() []*TxOut {
	return tx.Vout_
}

// AppendVout TODO
func (tx *Base) AppendVout(out *TxOut) {
	tx.Vout_ = append(tx.Vout_, out)
}

// Hash TODO
func (tx *Base) Hash() (hash.Hash256, error) {
	var buffer bytes.Buffer
	if _, err := tx.WriteTo(&buffer); err != nil {
		return hash.Hash256{}, err
	}
	return hash.DoubleHash(buffer.Bytes()), nil
}

// WriteTo TODO
func (tx *Base) WriteTo(w io.Writer) (int64, error) {
	if len(tx.Vin_) > 65535 {
		return 0, ErrExceedTransactionCount
	}
	if len(tx.Vin_) > 65535 {
		return 0, ErrExceedTransactionCount
	}

	var wrote int64
	if n, err := util.WriteUint64(w, tx.Timestamp_); err != nil {
		return wrote, err
	} else {
		wrote += n
	}

	if n, err := util.WriteUint16(w, uint16(len(tx.Vin_))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, vin := range tx.Vin() {
			if n, err := vin.WriteTo(w); err != nil {
				return wrote, err
			} else {
				wrote += n
			}
		}
	}

	if n, err := util.WriteUint16(w, uint16(len(tx.Vout_))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, vout := range tx.Vout() {
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
func (tx *Base) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if v, n, err := util.ReadUint64(r); err != nil {
		return read, err
	} else {
		read += n
		tx.Timestamp_ = v
	}

	if Len, n, err := util.ReadUint16(r); err != nil {
		return read, err
	} else {
		read += n
		tx.Vin_ = make([]*TxIn, 0, Len)
		for i := 0; i < int(Len); i++ {
			vin := new(TxIn)
			if n, err := vin.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				tx.Vin_ = append(tx.Vin_, vin)
			}
		}
	}

	if Len, n, err := util.ReadUint16(r); err != nil {
		return read, err
	} else {
		read += n
		tx.Vout_ = make([]*TxOut, 0, Len)
		for i := 0; i < int(Len); i++ {
			vout := new(TxOut)
			if n, err := vout.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				tx.Vout_ = append(tx.Vout_, vout)
			}
		}
	}
	return read, nil
}

// MarshalJSON TODO
func (tx *Base) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	enc := json.NewEncoder(&buffer)
	if err := enc.Encode(map[string]interface{}{
		"Timestamp": tx.Timestamp_,
		"Vin":       tx.Vin_,
		"Vout":      tx.Vout_,
	}); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// Debug TODO
func (tx *Base) Debug() (string, error) {
	if bs, err := tx.MarshalJSON(); err != nil {
		return "", err
	} else {
		return string(bs), err
	}
}
