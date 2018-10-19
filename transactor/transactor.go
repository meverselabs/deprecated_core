package transactor

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/amount"
	"git.fleta.io/fleta/core/data"
	"git.fleta.io/fleta/core/transaction"
)

var transactorHash = map[uint64]*Transactor{}

// GetTransactor TODO
func GetTransactor(coord *common.Coordinate) *Transactor {
	if tran, _ := ByCoord(coord); tran != nil {
		return tran
	}
	tran := newTransactor(coord)
	transactorHash[coord.ID()] = tran
	return tran
}

// ByCoord TODO
func ByCoord(coord *common.Coordinate) (*Transactor, error) {
	if act, has := transactorHash[coord.ID()]; !has {
		return nil, ErrNotExistTransactor
	} else {
		return act, nil
	}
}

// Transactor TODO
type Transactor struct {
	coord           *common.Coordinate
	handlerTypeHash map[transaction.Type]*handler
	feeHash         map[transaction.Type]*amount.Amount
	typeNameHash    map[string]transaction.Type
	typeHash        map[transaction.Type]*typeItem
}

func newTransactor(coord *common.Coordinate) *Transactor {
	tran := &Transactor{
		coord:           coord,
		handlerTypeHash: map[transaction.Type]*handler{},
		feeHash:         map[transaction.Type]*amount.Amount{},
		typeNameHash:    map[string]transaction.Type{},
		typeHash:        map[transaction.Type]*typeItem{},
	}
	return tran
}

// Validate TODO
func (tran *Transactor) Validate(loader data.Loader, tx transaction.Transaction, signers []common.PublicHash) error {
	if !tran.coord.Equal(loader.ChainCoord()) {
		return ErrInvalidChainCoordinate
	}
	if !tran.coord.Equal(tx.ChainCoord()) {
		return ErrInvalidChainCoordinate
	}

	if item, has := tran.handlerTypeHash[tx.Type()]; !has {
		return ErrNotExistHandler
	} else {
		if err := item.Validator(loader, tx, signers); err != nil {
			return err
		}
		return nil
	}
}

// Execute TODO
func (tran *Transactor) Execute(ctx *data.Context, tx transaction.Transaction, coord *common.Coordinate) error {
	t := tx.Type()
	if !tran.coord.Equal(ctx.ChainCoord()) {
		return ErrInvalidChainCoordinate
	}
	if !tran.coord.Equal(tx.ChainCoord()) {
		return ErrInvalidChainCoordinate
	}

	if item, has := tran.handlerTypeHash[t]; !has {
		return ErrNotExistHandler
	} else {
		if err := item.Executor(ctx, tran.feeHash[t].Clone(), tx, coord); err != nil {
			return err
		}
		return nil
	}
}

// RegisterType TODO
func (tran *Transactor) RegisterType(Name string, t transaction.Type, Fee *amount.Amount) error {
	item, err := loadHandler(Name)
	if err != nil {
		return err
	}
	tran.AddType(t, Name, item.Factory)
	tran.handlerTypeHash[t] = item
	tran.feeHash[t] = Fee
	tran.typeNameHash[Name] = t
	return nil
}

// AddType TODO
func (tran *Transactor) AddType(Type transaction.Type, Name string, Factory Factory) {
	tran.typeHash[Type] = &typeItem{
		Type:    Type,
		Name:    Name,
		Factory: Factory,
	}
}

// NewByType TODO
func (tran *Transactor) NewByType(t transaction.Type) (transaction.Transaction, error) {
	if item, has := tran.typeHash[t]; has {
		tx := item.Factory(t)
		tx.SetType(t)
		return tx, nil
	} else {
		return nil, ErrUnknownTransactionType
	}
}

// NewByTypeName TODO
func (tran *Transactor) NewByTypeName(name string) (transaction.Transaction, error) {
	if t, has := tran.typeNameHash[name]; has {
		return tran.NewByType(t)
	} else {
		return nil, ErrUnknownTransactionType
	}
}

var handlerHash = map[string]*handler{}

// RegisterHandler TODO
func RegisterHandler(Name string, Factory Factory, Validator Validator, Executor Executor) error {
	if _, has := handlerHash[Name]; has {
		return ErrExistHandler
	}
	handlerHash[Name] = &handler{
		Factory:   Factory,
		Validator: Validator,
		Executor:  Executor,
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
	Executor  Executor
}

type typeItem struct {
	Type    transaction.Type
	Name    string
	Factory Factory
}

// Factory TODO
type Factory func(t transaction.Type) transaction.Transaction

// Validator TODO
type Validator func(loader data.Loader, tx transaction.Transaction, signers []common.PublicHash) error

// Executor TODO
type Executor func(ctx *data.Context, Fee *amount.Amount, tx transaction.Transaction, coord *common.Coordinate) error
