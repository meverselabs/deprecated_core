package store

import "errors"

// store errors
var (
	ErrAlreadyExistGenesis = errors.New("already exist genesis")
	ErrInvalidTxInKey      = errors.New("invalid txin key")
	ErrInvalidChainCoord   = errors.New("invalid chain coord")
)
