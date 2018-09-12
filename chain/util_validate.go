package chain

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/store"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/chain/account"
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
	AccountHash map[string]*account.Account
}

// NewValidationContext TODO
func NewValidationContext() *ValidationContext {
	ctx := &ValidationContext{
		AccountHash: map[string]*account.Account{},
	}
	return ctx
}

// ValidateTransaction TODO
func ValidateTransaction(cn Chain, tx transaction.Transaction, singers []common.PublicKey) error {
	ctx := NewValidationContext()
	return validateTransaction(ctx, cn, tx, singers, 0, false)
}

// validateTransactionWithResult TODO
func validateTransactionWithResult(ctx *ValidationContext, cn Chain, tx transaction.Transaction, singers []common.PublicKey, idx uint16) error {
	return validateTransaction(ctx, cn, tx, singers, idx, true)
}

// validateTransaction TODO
func validateTransaction(ctx *ValidationContext, cn Provider, t transaction.Transaction, singers []common.PublicKey, idx uint16, bResult bool) error {
	Fee := cn.Fee(t)
	switch tx := t.(type) {
	case *advanced.Trade:
		fromAcc, has := ctx.AccountHash[string(tx.From[:])]
		if !has {
			acc, err := cn.Account(tx.From)
			if err != nil {
				return err
			}
			fromAcc = acc
			ctx.AccountHash[string(tx.From[:])] = fromAcc
		}
		if len(fromAcc.PublicKeys) != len(singers) {
			return ErrMismatchSignaturesCount
		}
		for i, singer := range singers {
			if !singer.Equal(fromAcc.PublicKeys[i]) {
				return ErrInvalidTransactionSignature
			}
		}

		if fromAcc.Balance.Less(Fee) {
			return ErrInsuffcientBalance
		}
		fromAcc.Balance = fromAcc.Balance.Sub(Fee)

		for _, vout := range tx.Vout {
			if vout.Amount.IsZero() {
				return ErrInvalidAmount
			}
			if vout.Amount.Less(cn.Config().DustAmount) {
				return ErrTooSmallAmount
			}

			if fromAcc.Balance.Less(vout.Amount) {
				return ErrInsuffcientBalance
			}
			fromAcc.Balance = fromAcc.Balance.Sub(vout.Amount)

			toAcc, has := ctx.AccountHash[string(vout.Address[:])]
			if !has {
				acc, err := cn.Account(tx.From)
				if err != nil {
					return err
				}
				fromAcc = acc
				ctx.AccountHash[string(tx.From[:])] = fromAcc
			}
			toAcc.Balance = toAcc.Balance.Add(vout.Amount)
		}
	case *advanced.Formulation:
		fromAcc, has := ctx.AccountHash[string(tx.From[:])]
		if !has {
			acc, err := cn.Account(tx.From)
			if err != nil {
				return err
			}
			fromAcc = acc
			ctx.AccountHash[string(tx.From[:])] = fromAcc
		}
		if len(fromAcc.PublicKeys) != len(singers) {
			return ErrMismatchSignaturesCount
		}
		for i, singer := range singers {
			if !singer.Equal(fromAcc.PublicKeys[i]) {
				return ErrInvalidTransactionSignature
			}
		}

		if fromAcc.Balance.Less(Fee) {
			return ErrInsuffcientBalance
		}
		fromAcc.Balance = fromAcc.Balance.Sub(Fee)

		//TODO : update formulator information
	case *advanced.MultiSigAccount:
		fromAcc, has := ctx.AccountHash[string(tx.From[:])]
		if !has {
			acc, err := cn.Account(tx.From)
			if err != nil {
				return err
			}
			fromAcc = acc
			ctx.AccountHash[string(tx.From[:])] = fromAcc
		}
		if len(fromAcc.PublicKeys) != len(singers) {
			return ErrMismatchSignaturesCount
		}
		for i, singer := range singers {
			if !singer.Equal(fromAcc.PublicKeys[i]) {
				return ErrInvalidTransactionSignature
			}
		}

		if fromAcc.Balance.Less(Fee) {
			return ErrInsuffcientBalance
		}
		fromAcc.Balance = fromAcc.Balance.Sub(Fee)

		// TODO : make MultiSigAccountt address
		var addr common.Address
		if _, has := ctx.AccountHash[string(addr[:])]; !has {
			if _, err := cn.Account(addr); err != nil {
				if err != store.ErrNotExistKey {
					return err
				} else {
					acc := &account.Account{}
					ctx.AccountHash[string(addr[:])] = acc
				}
			} else {
				return ErrExistAddress
			}
		} else {
			return ErrExistAddress
		}
	}
	return nil
}
