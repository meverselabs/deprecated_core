package formulator

import "errors"

// errors
var (
	ErrInvalidTimestamp     = errors.New("invalid timestamp")
	ErrNotAllowedPublicHash = errors.New("not allowed public hash")
)
