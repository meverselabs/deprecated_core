package advanced

import (
	"errors"
)

// advanced errors
var (
	ErrExceedAddressCount   = errors.New("exceed address count")
	ErrExceedTxOutCount     = errors.New("exceed tx out count")
	ErrExceedSignitureCount = errors.New("exceed signiture count")
)
