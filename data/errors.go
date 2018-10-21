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
