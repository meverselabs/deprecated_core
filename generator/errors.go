package generator

import "errors"

// generator errors
var (
	ErrInvalidGeneratorAddress = errors.New("invalid generator address")
	ErrInvalidBlockVersion     = errors.New("invalid block version")
	ErrInvalidGenTimeThreshold = errors.New("invalid gen time threshold")
)
