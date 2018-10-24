package data

import "errors"

// transaction errors
var (
	ErrExistAccount    = errors.New("exist account")
	ErrNotExistAccount = errors.New("not exist account")
	ErrExistUTXO       = errors.New("exist utxo")
	ErrNotExistUTXO    = errors.New("not exist utxo")
	ErrDoubleSpent     = errors.New("double spent")
)

// accounter errors
var (
	ErrUnknownAccountType = errors.New("unknown account type")
	ErrNotExistHandler    = errors.New("not exist handler")
	ErrExistHandler       = errors.New("exist handler")
	ErrNotExistAccounter  = errors.New("not exist accounter")
)
