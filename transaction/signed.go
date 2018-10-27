package transaction

import (
	"bytes"
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
)

// Signed is the signature of the transaction creator
type Signed struct {
	TransactionHash hash.Hash256
	Signatures      []common.Signature
}

// Hash returns the hash value of it
func (s *Signed) Hash() (hash.Hash256, error) {
	var buffer bytes.Buffer
	if _, err := s.WriteTo(&buffer); err != nil {
		return hash.Hash256{}, err
	}
	return hash.DoubleHash(buffer.Bytes()), nil
}

// WriteTo is a serialization function
func (s *Signed) WriteTo(w io.Writer) (int64, error) {
	if len(s.Signatures) > 255 {
		return 0, ErrExceedSignatureCount
	}

	var wrote int64
	if n, err := s.TransactionHash.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}

	if n, err := util.WriteUint8(w, uint8(len(s.Signatures))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, sig := range s.Signatures {
			wrote += n
			if n, err := sig.WriteTo(w); err != nil {
				return wrote, err
			} else {
				wrote += n
			}
		}
	}
	return wrote, nil
}

// ReadFrom is a deserialization function
func (s *Signed) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := s.TransactionHash.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}

	if Len, n, err := util.ReadUint8(r); err != nil {
		return read, err
	} else {
		read += n
		s.Signatures = make([]common.Signature, 0, Len)
		for i := 0; i < int(Len); i++ {
			var sig common.Signature
			if n, err := sig.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				s.Signatures = append(s.Signatures, sig)
			}
		}
	}
	return read, nil
}
