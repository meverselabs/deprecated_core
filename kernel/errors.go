package kernel

import "errors"

// kernel errors
var (
	ErrInvalidChainCoordinate    = errors.New("invalid chain coordinate")
	ErrInvalidHashLevelRoot      = errors.New("invalid hash level root")
	ErrInvalidGenesisHash        = errors.New("invalid genesis hash")
	ErrNotExistConsensusSaveData = errors.New("invalid consensus save data")
	ErrDirtyContext              = errors.New("dirty context")
	ErrInvalidGenerateRequest    = errors.New("invalid generate request")
	ErrNotFormulator             = errors.New("not formulator")
	ErrClosed                    = errors.New("closed")
)
