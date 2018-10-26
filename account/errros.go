package account

import (
	"errors"
)

// chain errors
var (
	ErrMismatchSignaturesCount = errors.New("mismatch signatures count")
)
