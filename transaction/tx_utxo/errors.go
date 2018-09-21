package tx_utxo

import (
	"errors"
)

// advanced errors
var (
	ErrExceedTxInCount = errors.New("exceed txin count")
)
