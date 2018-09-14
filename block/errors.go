package block

import (
	"errors"
)

// block errors
var (
	ErrUnknownTransactionType        = errors.New("unknown transaction type")
	ErrExceedTransactionCount        = errors.New("exceed transaction count")
	ErrExceedSignatureCount          = errors.New("exceed signature count")
	ErrMismatchSignaturesCount       = errors.New("mismatch signatures count")
	ErrExceedTimeoutCount            = errors.New("exceed timeout count")
	ErrExceedTableAppendMessageCount = errors.New("exceed table append message count")
)
