package wallet

import (
	"io"

	"github.com/satori/go.uuid"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/wallet/key"
)

// KeyHolder TODO
type KeyHolder struct {
	uuid string
	key  key.Key
}

// NewKeyHolder TODO
func NewKeyHolder(k key.Key) *KeyHolder {
	kh := &KeyHolder{
		uuid: uuid.Must(uuid.NewV1()).String(),
		key:  k,
	}
	return kh
}

// UUID TODO
func (kh *KeyHolder) UUID() string {
	return kh.uuid
}

// IsLock TODO
func (kh *KeyHolder) IsLock() bool {
	return kh.key.IsLock()
}

// PublicKey TODO
func (kh *KeyHolder) PublicKey() common.PublicKey {
	return kh.key.PublicKey()
}

// Sign TODO
func (kh *KeyHolder) Sign(h hash.Hash256) (common.Signature, error) {
	return kh.key.Sign(h)
}

// SignWithPassphrase TODO TEMP
func (kh *KeyHolder) SignWithPassphrase(h hash.Hash256, passphrase []byte) (common.Signature, error) {
	return kh.key.SignWithPassphrase(h, passphrase)
}

// Verify TODO
func (kh *KeyHolder) Verify(h hash.Hash256, sig common.Signature) bool {
	return kh.key.Verify(h, sig)
}

// WriteTo TODO
func (kh *KeyHolder) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := util.WriteString(w, kh.uuid); err != nil {
		return wrote, err
	} else {
		wrote += int64(n)
	}
	if t, err := TypeOfKey(kh.key); err != nil {
		return 0, err
	} else if n, err := util.WriteUint8(w, uint8(t)); err != nil {
		return wrote, err
	} else {
		wrote += int64(n)
	}
	if n, err := kh.key.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += int64(n)
	}
	return wrote, nil
}

// ReadFrom TODO
func (kh *KeyHolder) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if str, n, err := util.ReadString(r); err != nil {
		return read, err
	} else {
		read += int64(n)
		kh.uuid = str
	}
	if v, n, err := util.ReadUint8(r); err != nil {
		return read, err
	} else {
		read += int64(n)
		if k, err := NewKeyByType(KeyType(v)); err != nil {
			return 0, err
		} else {
			kh.key = k
		}
	}
	if n, err := kh.key.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += int64(n)
	}
	return read, nil
}
