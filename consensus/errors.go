package consensus

import "errors"

// consensus errors
var (
	ErrInvalidKeyHashCount        = errors.New("invalid key hash count")
	ErrInvalidSequence            = errors.New("invalid sequence")
	ErrInsuffcientBalance         = errors.New("insufficient balance")
	ErrInvalidToAddress           = errors.New("invalid to address")
	ErrInvalidBlockHash           = errors.New("invalid block hash")
	ErrInvalidPhase               = errors.New("invalid phase")
	ErrExistAddress               = errors.New("exist address")
	ErrExceedCandidateCount       = errors.New("exceed candidate count")
	ErrInsufficientCandidateCount = errors.New("insufficient candidate count")
)
