package block

import (
	"bytes"
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/consensus"
	"git.fleta.io/fleta/core/consensus/timeout"
)

// Signed TODO
type Signed struct {
	BlockHash           hash.Hash256
	GeneratorSignature  common.Signature
	HeadTimeouts        []*timeout.Timeout
	TableAppendMessages []*consensus.SignedTableAppend
}

// Hash TODO
func (s *Signed) Hash() (hash.Hash256, error) {
	var buffer bytes.Buffer
	if _, err := s.WriteTo(&buffer); err != nil {
		return hash.Hash256{}, err
	}
	return hash.DoubleHash(buffer.Bytes()), nil
}

// WriteTo TODO
func (s *Signed) WriteTo(w io.Writer) (int64, error) {
	if len(s.HeadTimeouts) > 255 {
		return 0, ErrExceedTimeoutCount
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

	if n, err := util.WriteUint8(w, uint8(len(s.HeadTimeouts))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, to := range s.HeadTimeouts {
			if n, err := to.WriteTo(w); err != nil {
				return wrote, err
			} else {
				wrote += n
			}
		}
	}

	if n, err := util.WriteUint8(w, uint8(len(s.TableAppendMessages))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, tm := range s.TableAppendMessages {
			if n, err := tm.WriteTo(w); err != nil {
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
		s.HeadTimeouts = make([]*timeout.Timeout, 0, Len)
		for i := 0; i < int(Len); i++ {
			to := new(timeout.Timeout)
			if n, err := to.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				s.HeadTimeouts = append(s.HeadTimeouts, to)
			}
		}
	}

	if Len, n, err := util.ReadUint8(r); err != nil {
		return read, err
	} else {
		read += n
		s.TableAppendMessages = make([]*consensus.SignedTableAppend, 0, Len)
		for i := 0; i < int(Len); i++ {
			tm := new(consensus.SignedTableAppend)
			if n, err := tm.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				s.TableAppendMessages = append(s.TableAppendMessages, tm)
			}
		}
	}
	return read, nil
}

// ObserverSigned TODO
type ObserverSigned struct {
	Signed
	ObserverSignatures []common.Signature //MAXLEN : 255
}

// WriteTo TODO
func (s *ObserverSigned) WriteTo(w io.Writer) (int64, error) {
	if len(s.ObserverSignatures) > 255 {
		return 0, ErrExceedSignatureCount
	}

	var wrote int64
	if n, err := s.Signed.WriteTo(w); err != nil {
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
func (s *ObserverSigned) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := s.Signed.ReadFrom(r); err != nil {
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