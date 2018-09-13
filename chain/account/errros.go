package account

import (
	"errors"
)

// ErrDoubleSpent chain errors
var (
	ErrMismatchSignaturesCount     = errors.New("mismatch signatures count")
	ErrInvalidTransactionSignature = errors.New("invalid transaction signature")
)
