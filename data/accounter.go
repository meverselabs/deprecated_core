package data

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/account"
)

// Accounter TODO
type Accounter struct {
	coord           *common.Coordinate
	handlerTypeHash map[account.Type]*accountHandler
	typeNameHash    map[string]account.Type
	typeHash        map[account.Type]*accountTypeItem
}

// NewAccounter TODO
func NewAccounter(coord *common.Coordinate) *Accounter {
	act := &Accounter{
		coord:           coord,
		handlerTypeHash: map[account.Type]*accountHandler{},
		typeNameHash:    map[string]account.Type{},
		typeHash:        map[account.Type]*accountTypeItem{},
	}
	return act
}

// ChainCoord TODO
func (act *Accounter) ChainCoord() *common.Coordinate {
	return act.coord
}

// Validate TODO
func (act *Accounter) Validate(loader Loader, acc account.Account, signers []common.PublicHash) error {
	if item, has := act.handlerTypeHash[acc.Type()]; !has {
		return ErrNotExistHandler
	} else {
		if err := item.Validator(loader, acc, signers); err != nil {
			return err
		}
		return nil
	}
}

// RegisterType TODO
func (act *Accounter) RegisterType(Name string, t account.Type) error {
	item, err := loadAccountHandler(Name)
	if err != nil {
		return err
	}
	act.AddType(t, Name, item.Factory)
	act.handlerTypeHash[t] = item
	act.typeNameHash[Name] = t
	return nil
}

// AddType TODO
func (act *Accounter) AddType(Type account.Type, Name string, Factory AccountFactory) {
	act.typeHash[Type] = &accountTypeItem{
		Type:    Type,
		Name:    Name,
		Factory: Factory,
	}
}

// NewByType TODO
func (act *Accounter) NewByType(t account.Type) (account.Account, error) {
	if item, has := act.typeHash[t]; has {
		acc := item.Factory(t)
		acc.SetType(t)
		return acc, nil
	} else {
		return nil, ErrUnknownAccountType
	}
}

// NewByTypeName TODO
func (act *Accounter) NewByTypeName(name string) (account.Account, error) {
	if t, has := act.typeNameHash[name]; has {
		return act.NewByType(t)
	} else {
		return nil, ErrUnknownAccountType
	}
}

// TypeByName TODO
func (act *Accounter) TypeByName(name string) (account.Type, error) {
	if t, has := act.typeNameHash[name]; has {
		return t, nil
	} else {
		return 0, ErrUnknownAccountType
	}
}

var accounterHandlerHash = map[string]*accountHandler{}

// RegisterAccount TODO
func RegisterAccount(Name string, Factory AccountFactory, Validator AccountValidator) error {
	if _, has := accounterHandlerHash[Name]; has {
		return ErrExistHandler
	}
	accounterHandlerHash[Name] = &accountHandler{
		Factory:   Factory,
		Validator: Validator,
	}
	return nil
}

func loadAccountHandler(Name string) (*accountHandler, error) {
	if _, has := accounterHandlerHash[Name]; !has {
		return nil, ErrNotExistHandler
	}
	return accounterHandlerHash[Name], nil
}

type accountHandler struct {
	Factory   AccountFactory
	Validator AccountValidator
}

type accountTypeItem struct {
	Type    account.Type
	Name    string
	Factory AccountFactory
}

// AccountFactory TODO
type AccountFactory func(t account.Type) account.Account

// AccountValidator TODO
type AccountValidator func(loader Loader, acc account.Account, signers []common.PublicHash) error
