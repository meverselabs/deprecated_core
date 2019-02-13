package consensus

import "errors"

// consensus errors
var (
	ErrInsufficientCandidateCount    = errors.New("insufficient candidate count")
	ErrInsufficientObserverSignature = errors.New("insufficient observer signature")
	ErrDuplicatedObserverSignature   = errors.New("duplicated observer signature")
	ErrInvalidObserverSignature      = errors.New("invalid observer signature")
	ErrInvalidKeyHashCount           = errors.New("invalid key hash count")
	ErrInvalidSequence               = errors.New("invalid sequence")
	ErrInsuffcientBalance            = errors.New("insufficient balance")
	ErrInvalidToAddress              = errors.New("invalid to address")
	ErrInvalidBlockHash              = errors.New("invalid block hash")
)
