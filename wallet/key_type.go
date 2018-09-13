package wallet

import (
	"git.fleta.io/fleta/core/wallet/key"
)

// KeyType TODO
type KeyType uint8

// key_type key types
const (
	FileKeyType = KeyType(10)
)

func (t KeyType) String() string {
	switch t {
	case FileKeyType:
		return "FileKeyType"
	default:
		return "UnknownKeyType"
	}
}

// TypeOfKey TODO
func TypeOfKey(k key.Key) (KeyType, error) {
	switch k.(type) {
	case *key.FileKey:
		return FileKeyType, nil
	default:
		return 0, ErrUnknownKeyType
	}
}

// NewKeyByType TODO
func NewKeyByType(t KeyType) (key.Key, error) {
	switch t {
	case FileKeyType:
		return new(key.FileKey), nil
	default:
		return nil, ErrUnknownKeyType
	}
}

// TypeNameOfKey TODO
func TypeNameOfKey(k key.Key) string {
	t, err := TypeOfKey(k)
	if err != nil {
		return "UnknownKeyType"
	} else {
		return t.String()
	}
}
