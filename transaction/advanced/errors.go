package advanced

import (
	"errors"
)

// advanced errors
var (
	ErrExceedPublicHashCount = errors.New("exceed public hash count")
	ErrExceedTradeOutCount   = errors.New("exceed trade out count")
	ErrExceedSignitureCount  = errors.New("exceed signiture count")
)
