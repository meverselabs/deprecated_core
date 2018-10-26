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

// MemoryKey TODO
type MemoryKey struct {
	privkey *ecdsa.PrivateKey
	pubkey  common.PublicKey
}

// NewMemoryKey TODO
func NewMemoryKey() (*MemoryKey, error) {
	privKey, err := ecrypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	ac := &MemoryKey{
		privkey: privKey,
	}
	if err := ac.calcPubkey(); err != nil {
		return nil, err
	}
	return ac, nil
}

// NewMemoryKeyFromBytes TODO
func NewMemoryKeyFromBytes(pk []byte) (*MemoryKey, error) {
	ac := &MemoryKey{
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
func (ac *MemoryKey) calcPubkey() error {
	pk := ecrypto.CompressPubkey(&ac.privkey.PublicKey)
	copy(ac.pubkey[:], pk[:])
	return nil
}

// PublicKey TODO
func (ac *MemoryKey) PublicKey() common.PublicKey {
	return ac.pubkey
}

// Sign TODO
func (ac *MemoryKey) Sign(h hash.Hash256) (common.Signature, error) {
	bs, err := ecrypto.Sign(h[:], ac.privkey)
	if err != nil {
		return common.Signature{}, err
	}
	var sig common.Signature
	copy(sig[:], bs)
	return sig, nil
}

// SignWithPassphrase TODO TEMP
func (ac *MemoryKey) SignWithPassphrase(h hash.Hash256, passphrase []byte) (common.Signature, error) {
	return common.Signature{}, nil
}

// Verify TODO
func (ac *MemoryKey) Verify(h hash.Hash256, sig common.Signature) bool {
	return ecrypto.VerifySignature(ac.pubkey[:], h[:], sig[:])
}

// Bytes TODO
func (ac *MemoryKey) Bytes() []byte {
	return ac.privkey.D.Bytes()
}

// WriteTo TODO
func (ac *MemoryKey) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := util.WriteBytes(w, ac.privkey.D.Bytes()); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom TODO
func (ac *MemoryKey) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if bs, n, err := util.ReadBytes(r); err != nil {
		return read, err
	} else {
		read += n
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
	return read, nil
}
