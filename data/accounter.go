package data

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/account"
)

// Accounter provide account's handlers of the target chain
type Accounter struct {
	coord          *common.Coordinate
	handlerTypeMap map[account.Type]*accountHandler
	typeNameMap    map[string]account.Type
	typeMap        map[account.Type]*accountTypeItem
}

// NewAccounter returns a Accounter
func NewAccounter(coord *common.Coordinate) *Accounter {
	act := &Accounter{
		coord:          coord,
		handlerTypeMap: map[account.Type]*accountHandler{},
		typeNameMap:    map[string]account.Type{},
		typeMap:        map[account.Type]*accountTypeItem{},
	}
	return act
}

// ChainCoord returns the coordinate of the target chain
func (act *Accounter) ChainCoord() *common.Coordinate {
	return act.coord
}

// Validate supports the validation of the account with signers
func (act *Accounter) Validate(loader Loader, acc account.Account, signers []common.PublicHash) error {
	if item, has := act.handlerTypeMap[acc.Type()]; !has {
		return ErrNotExistHandler
	} else {
		if err := item.Validator(loader, acc, signers); err != nil {
			return err
		}
		return nil
	}
}

// RegisterType add the account type with handler loaded by the name from the global account registry
func (act *Accounter) RegisterType(Name string, t account.Type) error {
	item, err := loadAccountHandler(Name)
	if err != nil {
		return err
	}
	act.typeMap[t] = &accountTypeItem{
		Type:    t,
		Name:    Name,
		Factory: item.Factory,
	}
	act.handlerTypeMap[t] = item
	act.typeNameMap[Name] = t
	return nil
}

// NewByType generate an account instance by the type
func (act *Accounter) NewByType(t account.Type) (account.Account, error) {
	if item, has := act.typeMap[t]; has {
		acc := item.Factory(t)
		acc.SetType(t)
		return acc, nil
	} else {
		return nil, ErrUnknownAccountType
	}
}

// NewByTypeName generate an account instance by the name
func (act *Accounter) NewByTypeName(name string) (account.Account, error) {
	if t, has := act.typeNameMap[name]; has {
		return act.NewByType(t)
	} else {
		return nil, ErrUnknownAccountType
	}
}

// TypeByName returns the type by the name
func (act *Accounter) TypeByName(name string) (account.Type, error) {
	if t, has := act.typeNameMap[name]; has {
		return t, nil
	} else {
		return 0, ErrUnknownAccountType
	}
}

// NameByType returns the name by the type
func (act *Accounter) NameByType(t account.Type) (string, error) {
	if item, has := act.typeMap[t]; has {
		return item.Name, nil
	} else {
		return "", ErrUnknownTransactionType
	}
}

var accounterHandlerMap = map[string]*accountHandler{}

// RegisterAccount register account handlers to the global account registry
func RegisterAccount(Name string, Factory AccountFactory, Validator AccountValidator) error {
	if _, has := accounterHandlerMap[Name]; has {
		return ErrExistHandler
	}
	accounterHandlerMap[Name] = &accountHandler{
		Factory:   Factory,
		Validator: Validator,
	}
	return nil
}

func loadAccountHandler(Name string) (*accountHandler, error) {
	if _, has := accounterHandlerMap[Name]; !has {
		return nil, ErrNotExistHandler
	}
	return accounterHandlerMap[Name], nil
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

// AccountFactory is a function type to generate an account instance by the type
type AccountFactory func(t account.Type) account.Account

// TransactionValidator is a function type to support the validation of the account with signers
type AccountValidator func(loader Loader, acc account.Account, signers []common.PublicHash) error
