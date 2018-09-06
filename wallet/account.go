package wallet

import (
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/util"
)

// Account TODO
type Account struct {
	*common.PrivateKey
	Name string // MAXLEN : 65535
}

// NewAccount TODO
func NewAccount(name string) (*Account, error) {
	privKey, err := common.NewPrivateKey()
	if err != nil {
		return nil, err
	}

	ac := &Account{
		Name:       name,
		PrivateKey: privKey,
	}
	return ac, nil
}

// NewAccountFromPrivateKey TODO
func NewAccountFromPrivateKey(name string, privKey *common.PrivateKey) (*Account, error) {
	ac := &Account{
		Name:       name,
		PrivateKey: privKey,
	}
	return ac, nil
}

// NewAccountFromBytes TODO
func NewAccountFromBytes(name string, pk []byte) (*Account, error) {
	privKey, err := common.NewPrivateKeyFromBytes(pk)
	if err != nil {
		return nil, err
	}

	ac := &Account{
		Name:       name,
		PrivateKey: privKey,
	}
	return ac, nil
}

// WriteTo TODO
func (ac *Account) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := util.WriteString(w, ac.Name); err != nil {
		return wrote, err
	} else {
		wrote += int64(n)
	}
	if n, err := ac.PrivateKey.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += int64(n)
	}
	return wrote, nil
}

// ReadFrom TODO
func (ac *Account) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if str, n, err := util.ReadString(r); err != nil {
		return read, err
	} else {
		read += int64(n)
		ac.Name = str
	}
	if n, err := ac.PrivateKey.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += int64(n)
	}
	return read, nil
}
