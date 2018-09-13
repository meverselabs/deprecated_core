package wallet

import (
	"bytes"
	"strconv"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/store"
)

// Wallet TODO
type Wallet interface {
	AddKeyHolder(kh *KeyHolder) error
	KeyHolders() []*KeyHolder
	KeyHolderByID(UUID string) (*KeyHolder, error)
	KeyHolderByPublicKey(pubkey common.PublicKey) (*KeyHolder, error)
	DeleteKeyHolder(UUID string) error
}

// Base TODO
type Base struct {
	KeyStore      store.Store
	KeyHolderHash map[string]*KeyHolder
	publicKeyHash map[string]*KeyHolder
	uuids         []string
}

// NewBase TODO
func NewBase(KeyStore store.Store) (*Base, error) {
	wa := &Base{
		KeyStore:      KeyStore,
		KeyHolderHash: map[string]*KeyHolder{},
		publicKeyHash: map[string]*KeyHolder{},
		uuids:         []string{},
	}
	_, values, err := KeyStore.Scan(nil)
	if err != nil {
		return nil, err
	}
	for _, v := range values {
		if len(v) > 0 {
			kh := new(KeyHolder)
			if _, err := kh.ReadFrom(bytes.NewReader(v)); err != nil {
				return nil, err
			}
			wa.AddKeyHolder(kh)
		}
	}
	return wa, nil
}

// AddKeyHolder TODO
func (wa *Base) AddKeyHolder(kh *KeyHolder) error {
	var buffer bytes.Buffer
	if _, err := kh.WriteTo(&buffer); err != nil {
		return err
	}
	idx := len(wa.KeyHolderHash)
	id := strconv.Itoa(idx)
	if err := wa.KeyStore.Set([]byte(id), buffer.Bytes()); err != nil {
		return err
	}
	wa.KeyHolderHash[kh.UUID()] = kh
	pubkey := kh.PublicKey()
	wa.publicKeyHash[string(pubkey[:])] = kh
	wa.uuids = append(wa.uuids, kh.UUID())
	return nil
}

// KeyHolders TODO
func (wa *Base) KeyHolders() []*KeyHolder {
	list := make([]*KeyHolder, 0, len(wa.KeyHolderHash))
	for _, UUID := range wa.uuids {
		list = append(list, wa.KeyHolderHash[UUID])
	}
	return list
}

// KeyHolderByID TODO
func (wa *Base) KeyHolderByID(UUID string) (*KeyHolder, error) {
	if kh, has := wa.KeyHolderHash[UUID]; !has {
		return nil, ErrNotExistKeyHolder
	} else {
		return kh, nil
	}
}

// KeyHolderByPublicKey TODO
func (wa *Base) KeyHolderByPublicKey(pubkey common.PublicKey) (*KeyHolder, error) {
	if kh, has := wa.publicKeyHash[string(pubkey[:])]; !has {
		return nil, ErrNotExistKeyHolder
	} else {
		return kh, nil
	}
}

// DeleteKeyHolder TODO
func (wa *Base) DeleteKeyHolder(UUID string) error {
	if kh, err := wa.KeyHolderByID(UUID); err != nil {
		return err
	} else {
		delete(wa.KeyHolderHash, kh.UUID())
		pubkey := kh.PublicKey()
		delete(wa.publicKeyHash, string(pubkey[:]))
		list := make([]string, 0, len(wa.KeyHolderHash))
		for _, v := range wa.uuids {
			if v != UUID {
				list = append(list, v)
			}
		}
		wa.uuids = list
	}
	return nil
}
