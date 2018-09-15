package chain

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/store"
	"git.fleta.io/fleta/common/util"
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
	CurrentTxHash     hash.Hash256
	AccountHash       map[string]*account.Account
	DeleteAccountHash map[string]*account.Account
	AccountDataHash   map[string][]byte
}

// NewValidationContext TODO
func NewValidationContext() *ValidationContext {
	ctx := &ValidationContext{
		AccountHash:       map[string]*account.Account{},
		DeleteAccountHash: map[string]*account.Account{},
		AccountDataHash:   map[string][]byte{},
	}
	return ctx
}

// LoadAccount TODO
func (ctx *ValidationContext) LoadAccount(cn Provider, addr common.Address, checkLock bool) (*account.Account, error) {
	if _, has := ctx.DeleteAccountHash[string(addr[:])]; has {
		return nil, ErrDeletedAccount
	}

	targetAcc, has := ctx.AccountHash[string(addr[:])]
	if !has {
		acc, err := cn.Account(addr)
		if err != nil {
			return nil, err
		}
		if checkLock {
			switch acc.Type {
			case LockedAccountType:
				bs, err := ctx.AccountData(cn, acc.Address, "UnlockHeight")
				if err != nil {
					return nil, err
				}
				if cn.Height() < util.BytesToUint32(bs) {
					return nil, ErrLockedAccount
				}
			}
		}
		targetAcc = acc
		ctx.AccountHash[string(addr[:])] = targetAcc
	}
	return targetAcc, nil
}

// AccountData TODO
func (ctx *ValidationContext) AccountData(cn Provider, addr common.Address, name string) ([]byte, error) {
	key := toAccountDataKey(addr, name)
	data, has := ctx.AccountDataHash[string(key)]
	if !has {
		bs, err := cn.AccountData(addr, name)
		if err != nil {
			return nil, err
		}
		data = bs
	}
	return data, nil
}

// ValidateTransaction TODO
func ValidateTransaction(cn Chain, tx transaction.Transaction, signers []common.Address) error {
	ctx := NewValidationContext()
	txHash, err := tx.Hash()
	if err != nil {
		return err
	}
	ctx.CurrentTxHash = txHash
	return validateTransaction(ctx, cn, tx, signers, 0, false)
}

// validateTransactionWithResult TODO
func validateTransactionWithResult(ctx *ValidationContext, cn Chain, tx transaction.Transaction, signers []common.Address, idx uint16) error {
	return validateTransaction(ctx, cn, tx, signers, idx, true)
}

// validateTransaction TODO
func validateTransaction(ctx *ValidationContext, cn Provider, t transaction.Transaction, signers []common.Address, idx uint16, bResult bool) error {
	if !cn.Coordinate().Equal(t.Coordinate()) {
		return ErrMismatchCoordinate
	}

	signerHash := map[string]bool{}
	for _, signer := range signers {
		if signer.Type() != KeyAccountType {
			return ErrInvalidAccountType
		}
		signerHash[string(signer[:])] = true
	}
	if len(signers) != len(signerHash) {
		return ErrDuplicatedAddress
	}

	Fee := cn.Fee(t)
	switch tx := t.(type) {
	case *advanced.Trade:
		fromAcc, err := ctx.LoadAccount(cn, tx.From, true)
		if err != nil {
			return err
		}
		if t.Seq() != fromAcc.Seq+1 {
			return ErrInvalidSequence
		}
		if err := ValidateSigners(ctx, cn, fromAcc, signerHash); err != nil {
			return err
		}

		if fromAcc.Balance.Less(Fee) {
			return ErrInsuffcientBalance
		}
		fromAcc.Balance = fromAcc.Balance.Sub(Fee)
		fromAcc.Seq++

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

			toAcc, err := ctx.LoadAccount(cn, vout.Address, false)
			if err != nil {
				if err != store.ErrNotExistKey {
					return err
				} else {
					toAcc = CreateAccount(cn, vout.Address, []common.Address{vout.Address})
					ctx.AccountHash[string(vout.Address[:])] = toAcc
				}
			}
			toAcc.Balance = toAcc.Balance.Add(vout.Amount)
		}
	case *advanced.Burn:
		fromAcc, err := ctx.LoadAccount(cn, tx.From, false)
		if err != nil {
			return err
		}
		if t.Seq() != fromAcc.Seq+1 {
			return ErrInvalidSequence
		}
		if err := ValidateSigners(ctx, cn, fromAcc, signerHash); err != nil {
			return err
		}

		if fromAcc.Balance.Less(Fee) {
			return ErrInsuffcientBalance
		}
		fromAcc.Balance = fromAcc.Balance.Sub(Fee)
		fromAcc.Seq++

		if fromAcc.Balance.Less(tx.Amount) {
			return ErrInsuffcientBalance
		}
		fromAcc.Balance = fromAcc.Balance.Sub(tx.Amount)
	case *advanced.MultiSigAccount:
		fromAcc, err := ctx.LoadAccount(cn, tx.From, true)
		if err != nil {
			return err
		}
		if t.Seq() != fromAcc.Seq+1 {
			return ErrInvalidSequence
		}
		if err := ValidateSigners(ctx, cn, fromAcc, signerHash); err != nil {
			return err
		}

		if tx.Required == 0 || int(tx.Required) > len(tx.KeyAddresses) {
			return ErrInvalidMultiSigRequired
		}

		if fromAcc.Balance.Less(Fee) {
			return ErrInsuffcientBalance
		}
		fromAcc.Balance = fromAcc.Balance.Sub(Fee)
		fromAcc.Seq++

		addr := common.AddressFromHash(cn.Coordinate(), MultiSigAccountType, ctx.CurrentTxHash, common.ChecksumFromAddresses(tx.KeyAddresses))
		if _, err := ctx.LoadAccount(cn, addr, false); err != nil {
			if err != store.ErrNotExistKey {
				return err
			} else {
				acc := CreateAccount(cn, addr, tx.KeyAddresses)
				ctx.AccountHash[string(addr[:])] = acc
				ctx.AccountDataHash[string(toAccountDataKey(addr, "Required"))] = []byte{byte(tx.Required)}
			}
		} else {
			return ErrExistAddress
		}
	case *advanced.Formulation:
		fromAcc, err := ctx.LoadAccount(cn, tx.From, true)
		if err != nil {
			return err
		}
		if t.Seq() != fromAcc.Seq+1 {
			return ErrInvalidSequence
		}
		if err := ValidateSigners(ctx, cn, fromAcc, signerHash); err != nil {
			return err
		}

		if fromAcc.Balance.Less(Fee) {
			return ErrInsuffcientBalance
		}
		fromAcc.Balance = fromAcc.Balance.Sub(Fee)
		fromAcc.Seq++

		addr := common.AddressFromHash(cn.Coordinate(), FormulationAccountType, ctx.CurrentTxHash, common.ChecksumFromAddresses(signers))
		if _, err := ctx.LoadAccount(cn, addr, false); err != nil {
			if err != store.ErrNotExistKey {
				return err
			} else {
				acc := CreateAccount(cn, addr, signers)
				ctx.AccountHash[string(addr[:])] = acc
				ctx.AccountDataHash[string(toAccountDataKey(addr, "PublicKey"))] = tx.PublicKey[:]
			}
		} else {
			return ErrExistAddress
		}
	case *advanced.RevokeFormulation:
		fromAcc, err := ctx.LoadAccount(cn, tx.From, true)
		if err != nil {
			return err
		}
		if t.Seq() != fromAcc.Seq+1 {
			return ErrInvalidSequence
		}
		if err := ValidateSigners(ctx, cn, fromAcc, signerHash); err != nil {
			return err
		}

		formulationAcc, err := ctx.LoadAccount(cn, tx.FormulationAddress, false)
		if err != nil {
			return err
		}
		if err := ValidateSigners(ctx, cn, formulationAcc, signerHash); err != nil {
			return err
		}

		if fromAcc.Balance.Less(Fee) {
			return ErrInsuffcientBalance
		}
		fromAcc.Balance = fromAcc.Balance.Sub(Fee)
		fromAcc.Seq++
		fromAcc.Balance = fromAcc.Balance.Add(formulationAcc.Balance).Add(cn.Config().FormulationCost)

		ctx.DeleteAccountHash[string(tx.FormulationAddress[:])] = formulationAcc
	}
	return nil
}

// ValidateSigners TODO
func ValidateSigners(ctx *ValidationContext, cn Provider, acc *account.Account, signerHash map[string]bool) error {
	matchCount := 0
	for _, addr := range acc.KeyAddresses {
		if signerHash[string(addr[:])] {
			matchCount++
		}
	}
	switch acc.Type {
	case MultiSigAccountType:
		bs, err := ctx.AccountData(cn, acc.Address, "Required")
		if err != nil {
			return err
		}
		Required := int(uint8(bs[0]))
		if matchCount != Required {
			return ErrInvalidTransactionSignature
		}
	default:
		if matchCount != len(acc.KeyAddresses) {
			return ErrInvalidTransactionSignature
		}
	}
	return nil
}
