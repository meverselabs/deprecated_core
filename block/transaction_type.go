package block

import (
	"git.fleta.io/fleta/core/transaction"
	"git.fleta.io/fleta/core/transaction/tx_account"
	"git.fleta.io/fleta/core/transaction/tx_utxo"
)

// TransactionType TODO
type TransactionType uint8

// transaction_type transaction types
const (
	//Account Transaction
	TransferTransctionType          = TransactionType(10)
	TaggedTransferTransctionType    = TransactionType(11)
	WithdrawTransctionType          = TransactionType(18)
	BurnTransctionType              = TransactionType(19)
	SingleAccountTransctionType     = TransactionType(20)
	MultiSigAccountTransctionType   = TransactionType(21)
	FormulationTransctionType       = TransactionType(30)
	RevokeFormulationTransctionType = TransactionType(31)
	//UTXO transaction
	AssignTransctionType  = TransactionType(81)
	DepositTransctionType = TransactionType(91)
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
	case WithdrawTransctionType:
		return "WithdrawTransctionType"
	case AssignTransctionType:
		return "AssignTransctionType"
	case DepositTransctionType:
		return "DepositTransctionType"
	default:
		return "UnknownTransactionType"
	}
}

// TypeOfTransaction TODO
func TypeOfTransaction(tx transaction.Transaction) (TransactionType, error) {
	switch tx.(type) {
	case *tx_account.Transfer:
		return TransferTransctionType, nil
	case *tx_account.TaggedTransfer:
		return TaggedTransferTransctionType, nil
	case *tx_account.Burn:
		return BurnTransctionType, nil
	case *tx_account.SingleAccount:
		return SingleAccountTransctionType, nil
	case *tx_account.MultiSigAccount:
		return MultiSigAccountTransctionType, nil
	case *tx_account.Formulation:
		return FormulationTransctionType, nil
	case *tx_account.RevokeFormulation:
		return RevokeFormulationTransctionType, nil
	case *tx_account.Withdraw:
		return WithdrawTransctionType, nil
	case *tx_utxo.Assign:
		return AssignTransctionType, nil
	case *tx_utxo.Deposit:
		return DepositTransctionType, nil
	default:
		return 0, ErrUnknownTransactionType
	}
}

// NewTransactionByType TODO
func NewTransactionByType(t TransactionType) (transaction.Transaction, error) {
	switch t {
	case TransferTransctionType:
		return new(tx_account.Transfer), nil
	case TaggedTransferTransctionType:
		return new(tx_account.TaggedTransfer), nil
	case BurnTransctionType:
		return new(tx_account.Burn), nil
	case SingleAccountTransctionType:
		return new(tx_account.SingleAccount), nil
	case MultiSigAccountTransctionType:
		return new(tx_account.MultiSigAccount), nil
	case FormulationTransctionType:
		return new(tx_account.Formulation), nil
	case RevokeFormulationTransctionType:
		return new(tx_account.RevokeFormulation), nil
	case WithdrawTransctionType:
		return new(tx_account.Withdraw), nil
	case AssignTransctionType:
		return new(tx_utxo.Assign), nil
	case DepositTransctionType:
		return new(tx_utxo.Deposit), nil
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
