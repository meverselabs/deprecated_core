package chain

import "errors"

// chain errors
var (
	ErrDirtyContext              = errors.New("dirty context")
	ErrNotExistChainCoordinate   = errors.New("not exist chain coordinate")
	ErrInvalidChainCoordinate    = errors.New("invalid chain coordinate")
	ErrInvalidHashLevelRoot      = errors.New("invalid hash level root")
	ErrInvalidGenesisHash        = errors.New("invalid genesis hash")
	ErrNotExistConsensusSaveData = errors.New("invalid consensus save data")
	ErrExpiredContextHeight      = errors.New("expired context height")
	ErrExpiredContextBlockHash   = errors.New("expired context block hash")
	ErrInvalidAppendBlockHeight  = errors.New("invalid append block height")
	ErrInvalidAppendBlockHash    = errors.New("invalid append block hash")
	ErrInvalidAppendContextHash  = errors.New("invalid append context hash")
	ErrClosedChain               = errors.New("closed chain")
)
