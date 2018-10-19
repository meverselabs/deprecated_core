package transaction

import "errors"

// transaction errors
var (
	ErrExceedSignatureCount = errors.New("exceed signature count")
)
