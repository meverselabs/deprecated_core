package block

import (
	"errors"
)

var (
	// ErrUnknownTransactionType TODO
	ErrUnknownTransactionType = errors.New("unknown transaction type")
	// ErrExceedTransactionCount TODO
	ErrExceedTransactionCount = errors.New("exceed transaction count")
	// ErrExceedSignatureCount TODO
	ErrExceedSignatureCount = errors.New("exceed signature count")
	// ErrMismatchSignaturesPubkeys TODO
	ErrMismatchSignaturesCount = errors.New("mismatch signatures count")
)
