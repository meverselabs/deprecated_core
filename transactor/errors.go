package transactor

import "errors"

// transactor errors
var (
	ErrUnknownTransactionType = errors.New("unknown transaction type")
	ErrNotExistHandler        = errors.New("not exist handler")
	ErrExistHandler           = errors.New("exist handler")
	ErrNotExistTransactor     = errors.New("not exist transactor")
	ErrInvalidChainCoordinate = errors.New("invalid chain coordinate")
)
