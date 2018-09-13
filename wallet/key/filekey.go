package key

import (
	"crypto/ecdsa"
	"io"
	"math/big"

	ecrypto "github.com/ethereum/go-ethereum/crypto"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
)

// FileKey TODO
type FileKey struct {
	privkey *ecdsa.PrivateKey
	pubkey  common.PublicKey
}

// NewFileKey TODO
func NewFileKey() (*FileKey, error) {
	privKey, err := ecrypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	ac := &FileKey{
		privkey: privKey,
	}
	if err := ac.calcPubkey(); err != nil {
		return nil, err
	}
	return ac, nil
}

// NewFileKeyFromBytes TODO
func NewFileKeyFromBytes(pk []byte) (*FileKey, error) {
	ac := &FileKey{
		privkey: &ecdsa.PrivateKey{
			PublicKey: ecdsa.PublicKey{
				Curve: ecrypto.S256(),
			},
			D: new(big.Int),
		},
	}
	ac.privkey.D.SetBytes(pk)
	ac.privkey.PublicKey.X, ac.privkey.PublicKey.Y = ac.privkey.Curve.ScalarBaseMult(ac.privkey.D.Bytes())
	if err := ac.calcPubkey(); err != nil {
		return nil, err
	}
	return ac, nil
}

// PublicKey TODO
func (ac *FileKey) calcPubkey() error {
	pk := ecrypto.CompressPubkey(&ac.privkey.PublicKey)
	copy(ac.pubkey[:], pk[:])
	return nil
}

// IsLock TODO TEMP
func (ac *FileKey) IsLock() bool {
	return false
}

// PublicKey TODO
func (ac *FileKey) PublicKey() common.PublicKey {
	return ac.pubkey
}

// Sign TODO
func (ac *FileKey) Sign(h hash.Hash256) (common.Signature, error) {
	bs, err := ecrypto.Sign(h[:], ac.privkey)
	if err != nil {
		return common.Signature{}, err
	}
	var sig common.Signature
	copy(sig[:], bs)
	return sig, nil
}

// SignWithPassphrase TODO TEMP
func (ac *FileKey) SignWithPassphrase(h hash.Hash256, passphrase []byte) (common.Signature, error) {
	return common.Signature{}, nil
}

// Verify TODO
func (ac *FileKey) Verify(h hash.Hash256, sig common.Signature) bool {
	return ecrypto.VerifySignature(ac.pubkey[:], h[:], sig[:])
}

// WriteTo TODO
func (ac *FileKey) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	bs := ac.privkey.D.Bytes()
	if n, err := util.WriteUint16(w, uint16(len(bs))); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := w.Write(bs); err != nil {
		return wrote, err
	} else {
		wrote += int64(n)
	}
	return wrote, nil
}

// ReadFrom TODO
func (ac *FileKey) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if Len, n, err := util.ReadUint16(r); err != nil {
		return read, err
	} else {
		read += n
		bs := make([]byte, Len)
		if n, err := r.Read(bs); err != nil {
			return read, err
		} else {
			read += int64(n)
			ac.privkey = &ecdsa.PrivateKey{
				PublicKey: ecdsa.PublicKey{
					Curve: ecrypto.S256(),
				},
				D: new(big.Int),
			}
			ac.privkey.D.SetBytes(bs)
			ac.privkey.PublicKey.X, ac.privkey.PublicKey.Y = ac.privkey.Curve.ScalarBaseMult(ac.privkey.D.Bytes())
			if err := ac.calcPubkey(); err != nil {
				return read, err
			}
		}
	}
	return read, nil
}
