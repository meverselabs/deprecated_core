package kernel

import "errors"

// store errors
var (
	ErrInvalidTxInKey            = errors.New("invalid txin key")
	ErrInvalidChainCoord         = errors.New("invalid chain coord")
	ErrInvalidSignatureCount     = errors.New("invalid signature count")
	ErrKernelClosed              = errors.New("kernel closed")
	ErrDirtyContext              = errors.New("dirty context")
	ErrNotExistChainCoordinate   = errors.New("not exist chain coordinate")
	ErrInvalidChainCoordinate    = errors.New("invalid chain coordinate")
	ErrInvalidLevelRootHash      = errors.New("invalid level root hash")
	ErrNotExistConsensusSaveData = errors.New("invalid consensus save data")
	ErrExpiredContextHeight      = errors.New("expired context height")
	ErrExpiredContextBlockHash   = errors.New("expired context block hash")
	ErrInvalidAppendBlockHeight  = errors.New("invalid append block height")
	ErrInvalidAppendBlockHash    = errors.New("invalid append block hash")
	ErrInvalidAppendContextHash  = errors.New("invalid append context hash")
)
