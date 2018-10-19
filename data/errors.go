package data

import "errors"

// transaction errors
var (
	ErrExistAccount = errors.New("exist account")
	ErrDoubleSpent  = errors.New("double spent")
)
