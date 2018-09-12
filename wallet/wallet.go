package wallet

import (
	"bytes"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/store"
)

// Wallet TODO
type Wallet interface {
	Keys() []*Key
	CreateKey(name string) (*Key, error)
	KeysByName(name string) []*Key
	KeyByPublicKey(pubkey common.PublicKey) (*Key, error)
	UpdateKeyName(pubkey common.PublicKey, name string) error
	DeleteKey(pubkey common.PublicKey) error
}

// Base TODO
type Base struct {
	KeyStore store.Store
	Keys_    []*Key
}

// NewBase TODO
func NewBase(KeyStore store.Store) (*Base, error) {
	wa := &Base{
		KeyStore: KeyStore,
		Keys_:    []*Key{},
	}
	_, values, err := KeyStore.Scan(nil)
	if err != nil {
		return nil, err
	}
	for _, v := range values {
		ac := new(Key)
		ac.PrivateKey = new(common.PrivateKey)
		if _, err := ac.ReadFrom(bytes.NewReader(v)); err != nil {
			return nil, err
		}
		wa.Keys_ = append(wa.Keys_, ac)
	}
	return wa, nil
}

// Keys TODO
func (wa *Base) Keys() []*Key {
	return wa.Keys_
}

// CreateKey TODO
func (wa *Base) CreateKey(name string) (*Key, error) {
	ac, err := NewKey(name)
	if err != nil {
		return nil, err
	}
	if err := wa.saveKey(ac); err != nil {
		return nil, err
	}
	wa.Keys_ = append(wa.Keys_, ac)
	return ac, nil
}

// KeysByName TODO
func (wa *Base) KeysByName(name string) []*Key {
	list := []*Key{}
	for _, ac := range wa.Keys_ {
		if ac.Name == name {
			list = append(list, ac)
		}
	}
	return list
}

// KeyByPublicKey TODO
func (wa *Base) KeyByPublicKey(pubkey common.PublicKey) (*Key, error) {
	for _, ac := range wa.Keys_ {
		if ac.PublicKey().Equal(pubkey) {
			return ac, nil
		}
	}
	return nil, ErrNotExistKey
}

// UpdateKeyName TODO
func (wa *Base) UpdateKeyName(pubkey common.PublicKey, name string) error {
	ac, err := wa.KeyByPublicKey(pubkey)
	if err != nil {
		return err
	}
	ac.Name = name
	if err := wa.saveKey(ac); err != nil {
		return err
	}
	return nil
}

// DeleteKey TODO
func (wa *Base) DeleteKey(pubkey common.PublicKey) error {
	if err := wa.KeyStore.Delete(pubkey[:]); err != nil {
		return err
	}

	Keys := []*Key{}
	for _, ac := range wa.Keys_ {
		if !ac.PublicKey().Equal(pubkey) {
			Keys = append(Keys, ac)
		}
	}
	wa.Keys_ = Keys
	return nil
}

func (wa *Base) saveKey(ac *Key) error {
	var buffer bytes.Buffer
	if _, err := ac.WriteTo(&buffer); err != nil {
		return err
	}
	if err := wa.KeyStore.Set([]byte(ac.PublicKey().String()), buffer.Bytes()); err != nil {
		return err
	}
	return nil
}
