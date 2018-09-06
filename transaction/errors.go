package transaction

import (
	"errors"
)

var (
	// ErrExceedTransactionCount TODO
	ErrExceedTransactionCount = errors.New("exceed transaction count")
	// ErrExceedSignitureCount TODO
	ErrExceedSignitureCount = errors.New("exceed signiture count")
)
