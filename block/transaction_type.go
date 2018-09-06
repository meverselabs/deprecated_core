package block

import (
	"git.fleta.io/fleta/core/transaction"
)

// TransactionType TODO
type TransactionType uint8

const (
	// BaseTransctionType TODO
	BaseTransctionType = TransactionType(iota)
	// SignedTransctionType TODO
	SignedTransctionType = TransactionType(iota)
)

// TypeOfTransaction TODO
func TypeOfTransaction(tx transaction.Transaction) (TransactionType, error) {
	switch tx.(type) {
	case *transaction.Base:
		return BaseTransctionType, nil
	default:
		return 0, ErrUnknownTransactionType
	}
}

// NewTransactionByType TODO
func NewTransactionByType(t TransactionType) (transaction.Transaction, error) {
	switch t {
	case BaseTransctionType:
		return new(transaction.Base), nil
	default:
		return nil, ErrUnknownTransactionType
	}
}
