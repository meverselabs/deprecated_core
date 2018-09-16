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

// IsExistAccount TODO
func (ctx *ValidationContext) IsExistAccount(cn Provider, addr common.Address) (bool, error) {
	if _, err := ctx.loadAccount(cn, addr, false); err != nil {
		if err != store.ErrNotExistKey {
			return false, err
		} else {
			return false, nil
		}
	} else {
		return true, nil
	}
}

// LoadAccount TODO
func (ctx *ValidationContext) LoadAccount(cn Provider, addr common.Address) (*account.Account, error) {
	return ctx.loadAccount(cn, addr, true)
}

func (ctx *ValidationContext) loadAccount(cn Provider, addr common.Address, checkLock bool) (*account.Account, error) {
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
			switch acc.Address.Type() {
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
func ValidateTransaction(cn Chain, tx transaction.Transaction, signers []common.PublicHash) error {
	ctx := NewValidationContext()
	txHash, err := tx.Hash()
	if err != nil {
		return err
	}
	ctx.CurrentTxHash = txHash
	return validateTransaction(ctx, cn, tx, signers, 0, false)
}

// validateTransactionWithResult TODO
func validateTransactionWithResult(ctx *ValidationContext, cn Chain, tx transaction.Transaction, signers []common.PublicHash, idx uint16) error {
	return validateTransaction(ctx, cn, tx, signers, idx, true)
}

// validateTransaction TODO
func validateTransaction(ctx *ValidationContext, cn Provider, t transaction.Transaction, signers []common.PublicHash, idx uint16, bResult bool) error {
	if !cn.Coordinate().Equal(t.Coordinate()) {
		return ErrMismatchCoordinate
	}

	height := cn.Height() + 1

	signerHash := map[string]bool{}
	for _, signer := range signers {
		signerHash[string(signer[:])] = true
	}
	if len(signers) != len(signerHash) {
		return ErrDuplicatedPublicKey
	}

	fromAcc, err := ctx.LoadAccount(cn, t.From())
	if err != nil {
		return err
	}
	if t.Seq() != fromAcc.Seq+1 {
		return ErrInvalidSequence
	}
	if err := ValidateSigners(ctx, cn, fromAcc, signerHash); err != nil {
		return err
	}

	Fee := cn.Fee(t)
	if fromAcc.Balance.Less(Fee) {
		return ErrInsuffcientBalance
	}
	fromAcc.Balance = fromAcc.Balance.Sub(Fee)
	fromAcc.Seq++

	switch tx := t.(type) {
	case *advanced.Trade:
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

			toAcc, err := ctx.LoadAccount(cn, vout.Address)
			if err != nil {
				return err
			}
			toAcc.Balance = toAcc.Balance.Add(vout.Amount)
		}
	case *advanced.TaggedTrade:
		if tx.Amount.IsZero() {
			return ErrInvalidAmount
		}
		if tx.Amount.Less(cn.Config().DustAmount) {
			return ErrTooSmallAmount
		}

		if fromAcc.Balance.Less(tx.Amount) {
			return ErrInsuffcientBalance
		}
		fromAcc.Balance = fromAcc.Balance.Sub(tx.Amount)

		toAcc, err := ctx.LoadAccount(cn, tx.Address)
		if err != nil {
			return err
		}
		toAcc.Balance = toAcc.Balance.Add(tx.Amount)
	case *advanced.Burn:
		if fromAcc.Balance.Less(tx.Amount) {
			return ErrInsuffcientBalance
		}
		fromAcc.Balance = fromAcc.Balance.Sub(tx.Amount)
	case *advanced.SingleAccount:
		addr := common.NewAddress(SingleAccountType, height, idx)
		if is, err := ctx.IsExistAccount(cn, addr); err != nil {
			return err
		} else if is {
			return ErrExistAddress
		} else {
			acc := CreateAccount(cn, addr, []common.PublicHash{tx.KeyHash})
			ctx.AccountHash[string(addr[:])] = acc
		}
	case *advanced.MultiSigAccount:
		if len(tx.KeyHashes) < 2 {
			return ErrInvalidMultiSigKeyCount
		}
		if tx.Required == 0 || int(tx.Required) > len(tx.KeyHashes) {
			return ErrInvalidMultiSigRequired
		}

		addr := common.NewAddress(MultiSigAccountType, height, idx)
		if is, err := ctx.IsExistAccount(cn, addr); err != nil {
			return err
		} else if is {
			return ErrExistAddress
		} else {
			acc := CreateAccount(cn, addr, tx.KeyHashes)
			ctx.AccountHash[string(addr[:])] = acc
			ctx.AccountDataHash[string(toAccountDataKey(addr, "Required"))] = []byte{byte(tx.Required)}
		}
	case *advanced.Formulation:
		addr := common.NewAddress(FormulationAccountType, height, idx)
		if is, err := ctx.IsExistAccount(cn, addr); err != nil {
			return err
		} else if is {
			return ErrExistAddress
		} else {
			acc := CreateAccount(cn, addr, signers)
			ctx.AccountHash[string(addr[:])] = acc
			ctx.AccountDataHash[string(toAccountDataKey(addr, "PublicKey"))] = tx.PublicKey[:]
		}
	case *advanced.RevokeFormulation:
		formulationAcc, err := ctx.LoadAccount(cn, tx.FormulationAddress)
		if err != nil {
			return err
		}
		if err := ValidateSigners(ctx, cn, formulationAcc, signerHash); err != nil {
			return err
		}
		fromAcc.Balance = fromAcc.Balance.Add(formulationAcc.Balance).Add(cn.Config().FormulationCost)

		ctx.DeleteAccountHash[string(tx.FormulationAddress[:])] = formulationAcc
	}

	if fromAcc.Address.Type() == LockedAccountType && fromAcc.Balance.IsZero() {
		ctx.DeleteAccountHash[string(fromAcc.Address[:])] = fromAcc
	}
	return nil
}

// ValidateSigners TODO
func ValidateSigners(ctx *ValidationContext, cn Provider, acc *account.Account, signerHash map[string]bool) error {
	matchCount := 0
	for _, addr := range acc.KeyHashes {
		if signerHash[string(addr[:])] {
			matchCount++
		}
	}
	switch acc.Address.Type() {
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
		if matchCount != len(acc.KeyHashes) {
			return ErrInvalidTransactionSignature
		}
	}
	return nil
}
