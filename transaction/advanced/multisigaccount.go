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
	From       common.Address     //MAXLEN : 65535
	PublicKeys []common.PublicKey //MAXLEN : 256
}

// NewMultiSigAccount TODO
func NewMultiSigAccount(version uint16, timestamp uint64) *MultiSigAccount {
	return &MultiSigAccount{
		Base: transaction.Base{
			Version_:   version,
			Timestamp_: timestamp,
		},
		PublicKeys: []common.PublicKey{},
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
	if len(tx.PublicKeys) > 256 {
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

	if n, err := util.WriteUint8(w, uint8(len(tx.PublicKeys))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, pubkey := range tx.PublicKeys {
			if n, err := pubkey.WriteTo(w); err != nil {
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

	if Len, n, err := util.ReadUint16(r); err != nil {
		return read, err
	} else {
		read += n
		tx.PublicKeys = make([]common.PublicKey, 0, Len)
		for i := 0; i < int(Len); i++ {
			var pubkey common.PublicKey
			if n, err := pubkey.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				tx.PublicKeys = append(tx.PublicKeys, pubkey)
			}
		}
	}
	return read, nil
}
