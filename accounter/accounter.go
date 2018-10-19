package accounter

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/account"
)

var accounterHash = map[uint64]*Accounter{}

// GetAccounter TODO
func GetAccounter(coord *common.Coordinate) *Accounter {
	if act, _ := ByCoord(coord); act != nil {
		return act
	}
	act := newAccounter(coord)
	accounterHash[coord.ID()] = act
	return act
}

// ByCoord TODO
func ByCoord(coord *common.Coordinate) (*Accounter, error) {
	if act, has := accounterHash[coord.ID()]; !has {
		return nil, ErrNotExistAccounter
	} else {
		return act, nil
	}
}

// Accounter TODO
type Accounter struct {
	coord           *common.Coordinate
	handlerTypeHash map[account.Type]*handler
	typeNameHash    map[string]account.Type
	typeHash        map[account.Type]*typeItem
}

func newAccounter(coord *common.Coordinate) *Accounter {
	act := &Accounter{
		coord:           coord,
		handlerTypeHash: map[account.Type]*handler{},
		typeNameHash:    map[string]account.Type{},
		typeHash:        map[account.Type]*typeItem{},
	}
	return act
}

// Validate TODO
func (act *Accounter) Validate(acc account.Account, signers []common.PublicHash) error {
	if item, has := act.handlerTypeHash[acc.Type()]; !has {
		return ErrNotExistHandler
	} else {
		if err := item.Validator(acc, signers); err != nil {
			return err
		}
		return nil
	}
}

// RegisterType TODO
func (act *Accounter) RegisterType(Name string, t account.Type) error {
	item, err := loadHandler(Name)
	if err != nil {
		return err
	}
	act.AddType(t, Name, item.Factory)
	act.handlerTypeHash[t] = item
	act.typeNameHash[Name] = t
	return nil
}

// AddType TODO
func (act *Accounter) AddType(Type account.Type, Name string, Factory Factory) {
	act.typeHash[Type] = &typeItem{
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

var handlerHash = map[string]*handler{}

// RegisterHandler TODO
func RegisterHandler(Name string, Factory Factory, Validator Validator) error {
	if _, has := handlerHash[Name]; has {
		return ErrExistHandler
	}
	handlerHash[Name] = &handler{
		Factory:   Factory,
		Validator: Validator,
	}
	return nil
}

func loadHandler(Name string) (*handler, error) {
	if _, has := handlerHash[Name]; !has {
		return nil, ErrNotExistHandler
	}
	return handlerHash[Name], nil
}

// handler TODO
type handler struct {
	Factory   Factory
	Validator Validator
}

type typeItem struct {
	Type    account.Type
	Name    string
	Factory Factory
}

// Factory TODO
type Factory func(t account.Type) account.Account

// Validator TODO
type Validator func(acc account.Account, signers []common.PublicHash) error
