package block

import (
	"git.fleta.io/fleta/core/transaction"
	"git.fleta.io/fleta/core/transaction/advanced"
)

// TransactionType TODO
type TransactionType uint8

// transaction_type transaction types
const (
	TradeTransctionType       = TransactionType(iota)
	FormulationTransctionType = TransactionType(iota)
)

func (t TransactionType) String() string {
	switch t {
	case TradeTransctionType:
		return "TradeTransctionType"
	case FormulationTransctionType:
		return "FormulationTransctionType"
	default:
		return "UnknownTransactionType"
	}
}

// TypeNameOfTransaction TODO
func TypeNameOfTransaction(tx transaction.Transaction) string {
	t, err := TypeOfTransaction(tx)
	if err != nil {
		return "UnknownTransactionType"
	} else {
		return t.String()
	}
}

// TypeOfTransaction TODO
func TypeOfTransaction(tx transaction.Transaction) (TransactionType, error) {
	switch tx.(type) {
	case *advanced.Trade:
		return TradeTransctionType, nil
	case *advanced.Formulation:
		return FormulationTransctionType, nil
	default:
		return 0, ErrUnknownTransactionType
	}
}

// NewTransactionByType TODO
func NewTransactionByType(t TransactionType) (transaction.Transaction, error) {
	switch t {
	case TradeTransctionType:
		return new(advanced.Trade), nil
	case FormulationTransctionType:
		return new(advanced.Formulation), nil
	default:
		return nil, ErrUnknownTransactionType
	}
}
