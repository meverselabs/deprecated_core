package consensus

import "errors"

// consensus errors
var (
	ErrInvalidKeyHashCount           = errors.New("invalid key hash count")
	ErrInvalidSequence               = errors.New("invalid sequence")
	ErrInsuffcientBalance            = errors.New("insufficient balance")
	ErrInvalidToAddress              = errors.New("invalid to address")
	ErrInvalidBlockHash              = errors.New("invalid block hash")
	ErrInvalidPhase                  = errors.New("invalid phase")
	ErrExistAddress                  = errors.New("exist address")
	ErrExistAccountName              = errors.New("exist account name")
	ErrInvalidAccountName            = errors.New("invaild account name")
	ErrExceedCandidateCount          = errors.New("exceed candidate count")
	ErrInsufficientCandidateCount    = errors.New("insufficient candidate count")
	ErrInvalidMaxBlocksPerFormulator = errors.New("invalid max blocks per formulator")
)
