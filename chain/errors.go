package chain

import (
	"errors"
)

// ErrDoubleSpent chain errors
var (
	ErrMismatchAddress               = errors.New("mismatch address")
	ErrMismatchOwnerCount            = errors.New("mismatch owner count")
	ErrExceedTransactionInputValue   = errors.New("exceed transaction input value")
	ErrExceedTransactionInputHeight  = errors.New("exceed transaction input height")
	ErrInvalidCoinbase               = errors.New("invalid coinbase")
	ErrInvalidObserverPubkey         = errors.New("invalid observer pubkey")
	ErrInvalidGeneratorAddress       = errors.New("invalid generator address")
	ErrInvalidRankAddress            = errors.New("invalid rank address")
	ErrInvalidRankerAddressCount     = errors.New("invalid ranker address count")
	ErrDuplicatedAddress             = errors.New("duplicated address")
	ErrUnknownTransactionType        = errors.New("unknown transaction type")
	ErrExceedTransactionCount        = errors.New("exceed transaction count")
	ErrExceedSignatureCount          = errors.New("exceed signature count")
	ErrMismatchSignaturesCount       = errors.New("mismatch signatures count")
	ErrMismatchHashLevelRoot         = errors.New("mismatch HashLevelRoot")
	ErrMismatchHashPrevBlock         = errors.New("mismatch HashPrevBlock")
	ErrExceedChainHeight             = errors.New("exceed chain height")
	ErrExceedTransactionIndex        = errors.New("exceed transaction index")
	ErrInsuffcientBalance            = errors.New("insufficient balance")
	ErrInvalidSequence               = errors.New("invalid sequence")
	ErrStoreCorrupted                = errors.New("store corrupted")
	ErrExceedCoinbaseCount           = errors.New("exceed coinbase count")
	ErrInvalidTransactionFee         = errors.New("invalid transaction fee")
	ErrMismatchSignedBlockHash       = errors.New("mismatch signed block hash")
	ErrMismatchSignedTransactionHash = errors.New("mismatch signed transaction hash")
	ErrInvalidAmount                 = errors.New("invalid amount")
	ErrTooSmallAmount                = errors.New("too small amount")
	ErrDoubleSpent                   = errors.New("double spent")
	ErrExistAddress                  = errors.New("exist address")
)
