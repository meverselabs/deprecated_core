package block

import (
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
)

// Signed TODO
type Signed struct {
	BlockHash          hash.Hash256
	GeneratorSignature common.Signature
	ObserverSignatures []common.Signature //MAXLEN : 255
}

// WriteTo TODO
func (s *Signed) WriteTo(w io.Writer) (int64, error) {
	if len(s.ObserverSignatures) > 255 {
		return 0, ErrExceedSignatureCount
	}

	var wrote int64
	if n, err := s.BlockHash.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}

	if n, err := s.GeneratorSignature.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}

	if n, err := util.WriteUint8(w, uint8(len(s.ObserverSignatures))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, sig := range s.ObserverSignatures {
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

// ReadFrom TODO
func (s *Signed) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := s.BlockHash.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}

	if n, err := s.GeneratorSignature.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}

	if Len, n, err := util.ReadUint8(r); err != nil {
		return read, err
	} else {
		read += n
		s.ObserverSignatures = make([]common.Signature, 0, Len)
		for i := 0; i < int(Len); i++ {
			var sig common.Signature
			if n, err := sig.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				s.ObserverSignatures = append(s.ObserverSignatures, sig)
			}
		}
	}
	return read, nil
}
