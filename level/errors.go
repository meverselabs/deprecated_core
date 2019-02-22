package level

import (
	"errors"
)

// level errors
var (
	ErrExceedHashCount  = errors.New("exceed hash count")
	ErrInvalidHashCount = errors.New("invalid hash count")
)
