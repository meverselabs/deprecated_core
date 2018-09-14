package advanced

import (
	"bytes"
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/transaction"
)

// MultiSigAccount TODO
type MultiSigAccount struct {
	transaction.Base
	From         common.Address
	Required     uint8
	KeyAddresses []common.Address //MAXLEN : 256
}

// NewMultiSigAccount TODO
func NewMultiSigAccount(version uint16, timestamp uint64) *MultiSigAccount {
	return &MultiSigAccount{
		Base: transaction.Base{
			Version_:   version,
			Timestamp_: timestamp,
		},
		KeyAddresses: []common.Address{},
	}
}

// Hash TODO
func (tx *MultiSigAccount) Hash() (hash.Hash256, error) {
	var buffer bytes.Buffer
	if _, err := tx.WriteTo(&buffer); err != nil {
		return hash.Hash256{}, err
	}
	return hash.DoubleHash(buffer.Bytes()), nil
}

// WriteTo TODO
func (tx *MultiSigAccount) WriteTo(w io.Writer) (int64, error) {
	if len(tx.KeyAddresses) > 256 {
		return 0, ErrExceedTxOutCount
	}

	var wrote int64
	if n, err := tx.Base.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := tx.From.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint8(w, tx.Required); err != nil {
		return wrote, err
	} else {
		wrote += n
	}

	if n, err := util.WriteUint8(w, uint8(len(tx.KeyAddresses))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, addr := range tx.KeyAddresses {
			if n, err := addr.WriteTo(w); err != nil {
				return wrote, err
			} else {
				wrote += n
			}
		}
	}
	return wrote, nil
}

// ReadFrom TODO
func (tx *MultiSigAccount) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := tx.Base.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := tx.From.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if v, n, err := util.ReadUint8(r); err != nil {
		return read, err
	} else {
		read += n
		tx.Required = v
	}

	if Len, n, err := util.ReadUint8(r); err != nil {
		return read, err
	} else {
		read += n
		tx.KeyAddresses = make([]common.Address, 0, Len)
		for i := 0; i < int(Len); i++ {
			var addr common.Address
			if n, err := addr.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				tx.KeyAddresses = append(tx.KeyAddresses, addr)
			}
		}
	}
	return read, nil
}
