package chain

import (
	"errors"
)

var (
	// ErrMismatchAddress TODO
	ErrMismatchAddress = errors.New("mismatch address")
	// ErrMismatchOwnerCount TODO
	ErrMismatchOwnerCount = errors.New("mismatch owner count")
	// ErrExceedTransactionInputValue TODO
	ErrExceedTransactionInputValue = errors.New("exceed transaction input value")
	// ErrExceedTransactionInputHeight TODO
	ErrExceedTransactionInputHeight = errors.New("exceed transaction input height")
	// ErrInvalidCoinbaseTransaction TODO
	ErrInvalidCoinbaseTransaction = errors.New("invalid coinbase transcation")
	// ErrInvalidBlockSignatureCount TODO
	ErrInvalidBlockSignatureCount = errors.New("invalid block signature count")
	// ErrInvalidObserverPubkey TODO
	ErrInvalidObserverPubkey = errors.New("invalid observer pubkey")
	// ErrInvalidGeneratorAddress TODO
	ErrInvalidGeneratorAddress = errors.New("invalid generator address")
	// ErrInvalidRankAddress TODO
	ErrInvalidRankAddress = errors.New("invalid rank address")
	// ErrInvalidRankerAddressCount TODO
	ErrInvalidRankerAddressCount = errors.New("invalid ranker address count")
	// ErrDuplicatedAddress TODO
	ErrDuplicatedAddress = errors.New("duplicated address")
	// ErrUnknownTransactionType TODO
	ErrUnknownTransactionType = errors.New("unknown transaction type")
	// ErrExceedTransactionCount TODO
	ErrExceedTransactionCount = errors.New("exceed transaction count")
	// ErrExceedSignatureCount TODO
	ErrExceedSignatureCount = errors.New("exceed signature count")
	// ErrMismatchSignaturesCount TODO
	ErrMismatchSignaturesCount = errors.New("mismatch signatures count")
	// ErrMismatchHashLevelRoot TODO
	ErrMismatchHashLevelRoot = errors.New("mismatch HashLevelRoot")
	// ErrMismatchHashPrevBlock TODO
	ErrMismatchHashPrevBlock = errors.New("mismatch HashPrevBlock")
	// ErrExceedChainHeight TODO
	ErrExceedChainHeight = errors.New("exceed chain height")
	// ErrExceedTransactionIndex TODO
	ErrExceedTransactionIndex = errors.New("exceed transaction index")
	// ErrInsuffcientBalance TODO
	ErrInsuffcientBalance = errors.New("insufficient balance")
	// ErrInvalidUTXO TODO
	ErrInvalidUTXO = errors.New("invalid utxo")
	// ErrStoreCorrupted TODO
	ErrStoreCorrupted = errors.New("store corrupted")
	// ErrExceedCoinbaseCount TODO
	ErrExceedCoinbaseCount = errors.New("exceed coinbase count")
	// ErrInvalidTransactionFee TODO
	ErrInvalidTransactionFee = errors.New("invalid transaction fee")
	// ErrMismatchSignedBlockHash TODO
	ErrMismatchSignedBlockHash = errors.New("mismatch signed block hash")
	// ErrInvalidAmount TODO
	ErrInvalidAmount = errors.New("invalid amount")
)
