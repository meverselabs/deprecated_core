package kernel

import "errors"

// store errors
var (
	ErrInvalidTxInKey            = errors.New("invalid txin key")
	ErrInvalidChainCoord         = errors.New("invalid chain coord")
	ErrInvalidSignatureCount     = errors.New("invalid signature count")
	ErrKernelClosed              = errors.New("kernel closed")
	ErrStoreClosed               = errors.New("store closed")
	ErrDirtyContext              = errors.New("dirty context")
	ErrInvalidLevelRootHash      = errors.New("invalid level root hash")
	ErrNotExistConsensusSaveData = errors.New("invalid consensus save data")
	ErrNotExistRewardSaveData    = errors.New("invalid reward save data")
	ErrExpiredContextHeight      = errors.New("expired context height")
	ErrExpiredContextBlockHash   = errors.New("expired context block hash")
	ErrInvalidAppendBlockHeight  = errors.New("invalid append block height")
	ErrInvalidAppendBlockHash    = errors.New("invalid append block hash")
	ErrInvalidAppendContextHash  = errors.New("invalid append context hash")
	ErrInvalidTopSignature       = errors.New("invalid top signature")
	ErrInvalidUTXO               = errors.New("invalid utxo")
	ErrProcessingTransaction     = errors.New("processing transaction")
	ErrNotFormulator             = errors.New("not formulator")
	ErrPastSeq                   = errors.New("past seq")
	ErrTooFarSeq                 = errors.New("too far seq")
	ErrTxQueueOverflowed         = errors.New("tx queue overflowed")
)
