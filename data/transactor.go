package data

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/amount"
	"git.fleta.io/fleta/core/transaction"
)

// Transactor TODO
type Transactor struct {
	coord           *common.Coordinate
	handlerTypeHash map[transaction.Type]*transactionHandler
	feeHash         map[transaction.Type]*amount.Amount
	typeNameHash    map[string]transaction.Type
	typeHash        map[transaction.Type]*transactionTypeItem
}

// NewTransactor TODO
func NewTransactor(coord *common.Coordinate) *Transactor {
	tran := &Transactor{
		coord:           coord,
		handlerTypeHash: map[transaction.Type]*transactionHandler{},
		feeHash:         map[transaction.Type]*amount.Amount{},
		typeNameHash:    map[string]transaction.Type{},
		typeHash:        map[transaction.Type]*transactionTypeItem{},
	}
	return tran
}

// ChainCoord TODO
func (tran *Transactor) ChainCoord() *common.Coordinate {
	return tran.coord
}

// Validate TODO
func (tran *Transactor) Validate(loader Loader, tx transaction.Transaction, signers []common.PublicHash) error {
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
func (tran *Transactor) Execute(ctx *Context, tx transaction.Transaction, coord *common.Coordinate) (interface{}, error) {
	t := tx.Type()
	if !tran.coord.Equal(ctx.ChainCoord()) {
		return nil, ErrInvalidChainCoordinate
	}
	if !tran.coord.Equal(tx.ChainCoord()) {
		return nil, ErrInvalidChainCoordinate
	}

	if item, has := tran.handlerTypeHash[t]; !has {
		return nil, ErrNotExistHandler
	} else {
		if ret, err := item.Executor(ctx, tran.feeHash[t].Clone(), tx, coord); err != nil {
			return nil, err
		} else {
			return ret, nil
		}
	}
}

// RegisterType TODO
func (tran *Transactor) RegisterType(Name string, t transaction.Type, Fee *amount.Amount) error {
	item, err := loadTransactionHandler(Name)
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
func (tran *Transactor) AddType(Type transaction.Type, Name string, Factory TransactionFactory) {
	tran.typeHash[Type] = &transactionTypeItem{
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

// TypeByName TODO
func (tran *Transactor) TypeByName(name string) (transaction.Type, error) {
	if t, has := tran.typeNameHash[name]; has {
		return t, nil
	} else {
		return 0, ErrUnknownTransactionType
	}
}

var transactionHandlerHash = map[string]*transactionHandler{}

// RegisterTransaction TODO
func RegisterTransaction(Name string, Factory TransactionFactory, Validator TransactionValidator, Executor TransactionExecutor) error {
	if _, has := transactionHandlerHash[Name]; has {
		return ErrExistHandler
	}
	transactionHandlerHash[Name] = &transactionHandler{
		Factory:   Factory,
		Validator: Validator,
		Executor:  Executor,
	}
	return nil
}

func loadTransactionHandler(Name string) (*transactionHandler, error) {
	if _, has := transactionHandlerHash[Name]; !has {
		return nil, ErrNotExistHandler
	}
	return transactionHandlerHash[Name], nil
}

type transactionHandler struct {
	Factory   TransactionFactory
	Validator TransactionValidator
	Executor  TransactionExecutor
}

type transactionTypeItem struct {
	Type    transaction.Type
	Name    string
	Factory TransactionFactory
}

// TransactionFactory TODO
type TransactionFactory func(t transaction.Type) transaction.Transaction

// TransactionValidator TODO
type TransactionValidator func(loader Loader, tx transaction.Transaction, signers []common.PublicHash) error

// TransactionExecutor TODO
type TransactionExecutor func(ctx *Context, Fee *amount.Amount, tx transaction.Transaction, coord *common.Coordinate) (interface{}, error)
