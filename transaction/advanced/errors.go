package advanced

import (
	"errors"
)

// advanced errors
var (
	ErrExceedPublicHashCount  = errors.New("exceed public hash count")
	ErrExceedTransferOutCount = errors.New("exceed transfer out count")
	ErrExceedSignitureCount   = errors.New("exceed signiture count")
)
