package wallet

import (
	"bytes"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/store"
)

// Wallet TODO
type Wallet interface {
	Accounts() []*Account
	CreateAccount(name string) (*Account, error)
	AccountsByName(name string) []*Account
	AccountByAddress(addr common.Address) (*Account, error)
	UpdateAccountName(addr common.Address, name string) error
	DeleteAccount(addr common.Address) error
}

// Base TODO
type Base struct {
	accountStore store.Store
	accounts     []*Account
}

// NewBase TODO
func NewBase(accountStore store.Store) (*Base, error) {
	wa := &Base{
		accountStore: accountStore,
		accounts:     []*Account{},
	}
	_, values, err := accountStore.Scan(nil)
	if err != nil {
		return nil, err
	}
	for _, v := range values {
		ac := new(Account)
		ac.PrivateKey = new(common.PrivateKey)
		if _, err := ac.ReadFrom(bytes.NewReader(v)); err != nil {
			return nil, err
		}
		wa.accounts = append(wa.accounts, ac)
	}
	return wa, nil
}

// Accounts TODO
func (wa *Base) Accounts() []*Account {
	return wa.accounts
}

// CreateAccount TODO
func (wa *Base) CreateAccount(name string) (*Account, error) {
	ac, err := NewAccount(name)
	if err != nil {
		return nil, err
	}
	if err := wa.saveAccount(ac); err != nil {
		return nil, err
	}
	wa.accounts = append(wa.accounts, ac)
	return ac, nil
}

// AccountsByName TODO
func (wa *Base) AccountsByName(name string) []*Account {
	list := []*Account{}
	for _, ac := range wa.accounts {
		if ac.Name == name {
			list = append(list, ac)
		}
	}
	return list
}

// AccountByAddress TODO
func (wa *Base) AccountByAddress(addr common.Address) (*Account, error) {
	for _, ac := range wa.accounts {
		if ac.Address().Equal(addr) {
			return ac, nil
		}
	}
	return nil, ErrNotExistAccount
}

// UpdateAccountName TODO
func (wa *Base) UpdateAccountName(addr common.Address, name string) error {
	ac, err := wa.AccountByAddress(addr)
	if err != nil {
		return err
	}
	ac.Name = name
	if err := wa.saveAccount(ac); err != nil {
		return err
	}
	return nil
}

// DeleteAccount TODO
func (wa *Base) DeleteAccount(addr common.Address) error {
	if err := wa.accountStore.Delete(addr[:]); err != nil {
		return err
	}

	accounts := []*Account{}
	for _, ac := range wa.accounts {
		if !ac.Address().Equal(addr) {
			accounts = append(accounts, ac)
		}
	}
	wa.accounts = accounts
	return nil
}

func (wa *Base) saveAccount(ac *Account) error {
	var buffer bytes.Buffer
	if _, err := ac.WriteTo(&buffer); err != nil {
		return err
	}
	if err := wa.accountStore.Set([]byte(ac.Address().String()), buffer.Bytes()); err != nil {
		return err
	}
	return nil
}
