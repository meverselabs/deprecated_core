package account

import (
	"errors"
)

// amount errors
var (
	ErrMismatchSignaturesCount = errors.New("mismatch signatures count")
	ErrInsufficientBalance     = errors.New("insufficient balance")
)
