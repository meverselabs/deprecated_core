package kernel

import "errors"

// kernel errors
var (
	ErrNotTopMember  = errors.New("not top member")
	ErrNotFormulator = errors.New("not formulator")
	ErrClosedKernel  = errors.New("closed kernel")
)
