package timeout

import (
	"bytes"
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
)

// Timeout TODO
type Timeout struct {
	PublicKey common.PublicKey
}

// Hash TODO
func (to *Timeout) Hash() (hash.Hash256, error) {
	var buffer bytes.Buffer
	if _, err := to.WriteTo(&buffer); err != nil {
		return hash.Hash256{}, err
	}
	return hash.Hash(buffer.Bytes()), nil
}

// WriteTo TODO
func (to *Timeout) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := to.PublicKey.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom TODO
func (to *Timeout) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := to.PublicKey.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	return read, nil
}

// ReadFrom TODO
func (to *Timeout) String() string {
	pubkey := to.PublicKey.String()
	return pubkey[:2] + pubkey[len(pubkey)-2:]
}

// ObserverSignedTimeout TODO
type ObserverSignedTimeout struct {
	*Timeout
	ObserverSignatures []common.Signature
}

// WriteTo TODO
func (to *ObserverSignedTimeout) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := to.Timeout.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint8(w, uint8(len(to.ObserverSignatures))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, sig := range to.ObserverSignatures {
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
func (to *ObserverSignedTimeout) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := to.Timeout.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if Len, n, err := util.ReadUint8(r); err != nil {
		return read, err
	} else {
		read += n
		to.ObserverSignatures = make([]common.Signature, 0, Len)
		for i := 0; i < int(Len); i++ {
			var sig common.Signature
			if n, err := sig.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				to.ObserverSignatures = append(to.ObserverSignatures, sig)
			}
		}
	}
	return read, nil
}
