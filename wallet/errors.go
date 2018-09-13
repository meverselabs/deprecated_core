package wallet

import (
	"errors"
)

// wallet errors
var (
	ErrNotExistKeyHolder = errors.New("not exist key holder")
	ErrUnknownKeyType    = errors.New("unknown key type")
)
