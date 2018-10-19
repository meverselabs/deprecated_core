package accounter

import "errors"

// accounter errors
var (
	ErrUnknownAccountType = errors.New("unknown account type")
	ErrNotExistHandler    = errors.New("not exist handler")
	ErrExistHandler       = errors.New("exist handler")
	ErrNotExistAccounter  = errors.New("not exist accounter")
)
