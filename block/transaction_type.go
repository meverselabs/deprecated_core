package block

import (
	"git.fleta.io/fleta/core/transaction"
	"git.fleta.io/fleta/core/transaction/advanced"
)

// TransactionType TODO
type TransactionType uint8

// transaction_type transaction types
const (
	TransferTransctionType          = TransactionType(10)
	TaggedTransferTransctionType    = TransactionType(11)
	BurnTransctionType              = TransactionType(19)
	SingleAccountTransctionType     = TransactionType(20)
	MultiSigAccountTransctionType   = TransactionType(21)
	FormulationTransctionType       = TransactionType(30)
	RevokeFormulationTransctionType = TransactionType(31)
)

func (t TransactionType) String() string {
	switch t {
	case TransferTransctionType:
		return "TransferTransctionType"
	case TaggedTransferTransctionType:
		return "TaggedTransferTransctionType"
	case BurnTransctionType:
		return "BurnTransctionType"
	case SingleAccountTransctionType:
		return "SingleAccountTransctionType"
	case MultiSigAccountTransctionType:
		return "MultiSigAccountTransctionType"
	case FormulationTransctionType:
		return "FormulationTransctionType"
	case RevokeFormulationTransctionType:
		return "RevokeFormulationTransctionType"
	default:
		return "UnknownTransactionType"
	}
}

// TypeOfTransaction TODO
func TypeOfTransaction(tx transaction.Transaction) (TransactionType, error) {
	switch tx.(type) {
	case *advanced.Transfer:
		return TransferTransctionType, nil
	case *advanced.TaggedTransfer:
		return TaggedTransferTransctionType, nil
	case *advanced.Burn:
		return BurnTransctionType, nil
	case *advanced.SingleAccount:
		return SingleAccountTransctionType, nil
	case *advanced.MultiSigAccount:
		return MultiSigAccountTransctionType, nil
	case *advanced.Formulation:
		return FormulationTransctionType, nil
	case *advanced.RevokeFormulation:
		return RevokeFormulationTransctionType, nil
	default:
		return 0, ErrUnknownTransactionType
	}
}

// NewTransactionByType TODO
func NewTransactionByType(t TransactionType) (transaction.Transaction, error) {
	switch t {
	case TransferTransctionType:
		return new(advanced.Transfer), nil
	case TaggedTransferTransctionType:
		return new(advanced.TaggedTransfer), nil
	case BurnTransctionType:
		return new(advanced.Burn), nil
	case SingleAccountTransctionType:
		return new(advanced.SingleAccount), nil
	case MultiSigAccountTransctionType:
		return new(advanced.MultiSigAccount), nil
	case FormulationTransctionType:
		return new(advanced.Formulation), nil
	case RevokeFormulationTransctionType:
		return new(advanced.RevokeFormulation), nil
	default:
		return nil, ErrUnknownTransactionType
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
