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
	Required  uint8
	KeyHashes []common.PublicHash //MAXLEN : 15
}

// NewMultiSigAccount TODO
func NewMultiSigAccount(coord *common.Coordinate, timestamp uint64, seq uint64) *MultiSigAccount {
	return &MultiSigAccount{
		Base: transaction.Base{
			Coordinate_: coord.Clone(),
			Timestamp_:  timestamp,
			Seq_:        seq,
		},
		KeyHashes: []common.PublicHash{},
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
	if len(tx.KeyHashes) > 15 {
		return 0, ErrExceedPublicHashCount
	}

	var wrote int64
	if n, err := tx.Base.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint8(w, tx.Required); err != nil {
		return wrote, err
	} else {
		wrote += n
	}

	if n, err := util.WriteUint8(w, uint8(len(tx.KeyHashes))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, addr := range tx.KeyHashes {
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
		tx.KeyHashes = make([]common.PublicHash, 0, Len)
		for i := 0; i < int(Len); i++ {
			var addr common.PublicHash
			if n, err := addr.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				tx.KeyHashes = append(tx.KeyHashes, addr)
			}
		}
	}
	return read, nil
}
