package transaction

import "errors"

// transaction errors
var (
	ErrExceedSignatureCount = errors.New("exceed signature count")
	ErrExceedTxOutCount     = errors.New("exceed txout count")
)
