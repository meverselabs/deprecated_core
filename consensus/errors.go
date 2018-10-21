package consensus

import "errors"

// consensus errors
var (
	ErrInvalidPrevBlockHash          = errors.New("invalid prev block hash")
	ErrInvalidPrevBlockHeight        = errors.New("invalid prev block height")
	ErrInsufficientCandidateCount    = errors.New("insufficient candidate count")
	ErrInvalidTopMember              = errors.New("invalid top member")
	ErrInvalidTopSignature           = errors.New("invalid top signature")
	ErrInsufficientObserverSignature = errors.New("insufficient observer signature")
	ErrDuplicatedObserverSignature   = errors.New("duplicated observer signature")
	ErrInvalidObserverSignature      = errors.New("invalid observer signature")
	ErrInvalidKeyHashCount           = errors.New("invalid key hash count")
	ErrInvalidSequence               = errors.New("invalid sequence")
	ErrInsuffcientBalance            = errors.New("insufficient balance")
	ErrInvalidToAddress              = errors.New("invalid to address")
)
