package data

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/amount"
	"git.fleta.io/fleta/core/transaction"
)

// Transactor provide transaction's handlers of the target chain
type Transactor struct {
	coord           *common.Coordinate
	handlerTypeHash map[transaction.Type]*transactionHandler
	feeHash         map[transaction.Type]*amount.Amount
	typeNameHash    map[string]transaction.Type
	typeHash        map[transaction.Type]*transactionTypeItem
}

// NewTransactor returns a Transactor
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

// ChainCoord returns the coordinate of the target chain
func (tran *Transactor) ChainCoord() *common.Coordinate {
	return tran.coord
}

// Validate supports the validation of the transaction with signers
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

// Execute updates the context using the transaction and the coordinate of it
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

// RegisterType add the transaction type with handler loaded by the name from the global transaction registry
func (tran *Transactor) RegisterType(Name string, t transaction.Type, Fee *amount.Amount) error {
	item, err := loadTransactionHandler(Name)
	if err != nil {
		return err
	}
	tran.typeHash[t] = &transactionTypeItem{
		Type:    t,
		Name:    Name,
		Factory: item.Factory,
	}
	tran.handlerTypeHash[t] = item
	tran.feeHash[t] = Fee
	tran.typeNameHash[Name] = t
	return nil
}

// NewByType generate an transaction instance by the type
func (tran *Transactor) NewByType(t transaction.Type) (transaction.Transaction, error) {
	if item, has := tran.typeHash[t]; has {
		tx := item.Factory(t)
		tx.SetType(t)
		return tx, nil
	} else {
		return nil, ErrUnknownTransactionType
	}
}

// NewByTypeName generate an transaction instance by the name
func (tran *Transactor) NewByTypeName(name string) (transaction.Transaction, error) {
	if t, has := tran.typeNameHash[name]; has {
		return tran.NewByType(t)
	} else {
		return nil, ErrUnknownTransactionType
	}
}

// TypeByName returns the type by the name
func (tran *Transactor) TypeByName(name string) (transaction.Type, error) {
	if t, has := tran.typeNameHash[name]; has {
		return t, nil
	} else {
		return 0, ErrUnknownTransactionType
	}
}

var transactionHandlerHash = map[string]*transactionHandler{}

// RegisterTransaction register transaction handlers to the global account registry
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

// TransactionFactory is a function type to generate an account instance by the type
type TransactionFactory func(t transaction.Type) transaction.Transaction

// TransactionValidator is a function type to support the validation of the transaction with signers
type TransactionValidator func(loader Loader, tx transaction.Transaction, signers []common.PublicHash) error

// TransactionExecutor is a function type to update the context using the transaction and the coordinate of it
type TransactionExecutor func(ctx *Context, Fee *amount.Amount, tx transaction.Transaction, coord *common.Coordinate) (interface{}, error)
