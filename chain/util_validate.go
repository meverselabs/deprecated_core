package chain

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/core/amount"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/transaction"
	"git.fleta.io/fleta/core/transaction/advanced"
)

// ValidateBlockGeneratorSignature TODO
func ValidateBlockGeneratorSignature(b *block.Block, GeneratorSignature common.Signature, ExpectedPublicKey common.PublicKey) error {
	h, err := b.Header.Hash()
	if err != nil {
		return err
	}
	{
		pubkey, err := common.RecoverPubkey(h, GeneratorSignature)
		if err != nil {
			return err
		}
		if !pubkey.Equal(ExpectedPublicKey) {
			return ErrInvalidGeneratorAddress
		}
	}
	return nil
}

// ValidationContext TODO
type ValidationContext struct {
	TxHashes    []hash.Hash256
	SpentHash   map[uint64]bool
	UnspentHash map[uint64]*transaction.TxOut
}

// NewValidationContext TODO
func NewValidationContext() *ValidationContext {
	ctx := &ValidationContext{
		TxHashes:    []hash.Hash256{},
		SpentHash:   map[uint64]bool{},
		UnspentHash: map[uint64]*transaction.TxOut{},
	}
	return ctx
}

// ValidateTransaction TODO
func ValidateTransaction(cn Chain, tx transaction.Transaction, sigs []common.Signature) error {
	ctx := NewValidationContext()
	txHash, err := tx.Hash()
	if err != nil {
		return err
	}
	ctx.TxHashes = append(ctx.TxHashes, txHash)
	return validateTransaction(ctx, cn, tx, sigs, 0, false)
}

// validateTransactionWithResult TODO
func validateTransactionWithResult(ctx *ValidationContext, cn Chain, tx transaction.Transaction, sigs []common.Signature, idx uint16) error {
	return validateTransaction(ctx, cn, tx, sigs, idx, true)
}

// validateTransaction TODO
func validateTransaction(ctx *ValidationContext, cn Chain, t transaction.Transaction, sigs []common.Signature, idx uint16, bResult bool) error {
	height := cn.Height() + 1

	TxHash := ctx.TxHashes[len(ctx.TxHashes)-1]

	switch tx := t.(type) {
	case *transaction.Base:
		addrs := make([]common.Address, 0, len(sigs))
		var insum amount.Amount
		hasCoinbase := false
		for _, vin := range tx.Vin {
			if vin.IsCoinbase() {
				insum += cn.RewardValue()
				hasCoinbase = true
			} else {
				if vin.Height >= height {
					return ErrExceedTransactionInputHeight
				}
				if ctx.SpentHash[vin.ID()] {
					return ErrDoubleSpent
				}
				if utxo, err := cn.Unspent(vin.Height, vin.Index, vin.N); err != nil {
					return err
				} else {
					ctx.SpentHash[vin.ID()] = true
					insum += utxo.Amount

					for i, addr := range utxo.Addresses {
						if i >= len(addrs) {
							sig := sigs[i]
							pubkey, err := common.RecoverPubkey(TxHash, sig)
							if err != nil {
								return err
							}
							addrs = append(addrs, common.AddressFromPubkey(pubkey))
						}
						sigAddr := addrs[i]
						if !addr.Equal(sigAddr) {
							return ErrMismatchAddress
						}
					}
				}
			}
		}

		if hasCoinbase {
			if len(tx.Vin) > 1 {
				return ErrInvalidCoinbase
			}
		}
		var outsum amount.Amount
		for n, vout := range tx.Vout {
			if vout.Amount == 0 {
				return ErrInvalidAmount
			}
			ctx.UnspentHash[transaction.MarshalID(height, idx, uint16(n))] = vout
			outsum += vout.Amount
		}
		if outsum > insum {
			return ErrExceedTransactionInputValue
		}
		if hasCoinbase {
			if insum != outsum {
				return ErrInvalidTransactionFee
			}
		} else {
			fee := insum - outsum
			calcultedFee := amount.CaclulateFee(len(tx.Vin), len(tx.Vout))
			if fee != calcultedFee {
				return ErrInvalidTransactionFee
			}
		}
	case *advanced.Formulation:
		addrs := make([]common.Address, 0, len(sigs))
		var insum amount.Amount
		for _, vin := range tx.Vin {
			if vin.IsCoinbase() {
				return ErrInvalidCoinbase
			} else {
				if vin.Height >= height {
					return ErrExceedTransactionInputHeight
				}
				if ctx.SpentHash[vin.ID()] {
					return ErrDoubleSpent
				}
				if utxo, err := cn.Unspent(vin.Height, vin.Index, vin.N); err != nil {
					return err
				} else {
					ctx.SpentHash[vin.ID()] = true
					insum += utxo.Amount

					for i, addr := range utxo.Addresses {
						if i >= len(addrs) {
							sig := sigs[i]
							pubkey, err := common.RecoverPubkey(TxHash, sig)
							if err != nil {
								return err
							}
							addrs = append(addrs, common.AddressFromPubkey(pubkey))
						}
						sigAddr := addrs[i]
						if !addr.Equal(sigAddr) {
							return ErrMismatchAddress
						}
					}
				}
			}
		}

		calcultedFee := amount.CaclulateFee(len(tx.Vin), 1)
		if insum != cn.FormulationAmount()+calcultedFee {
			return ErrInvalidTransactionFee
		}
		//TODO : update formulator information
	}
	return nil
}
