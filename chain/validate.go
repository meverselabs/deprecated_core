package chain

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/core/amount"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/chain/account"
	"git.fleta.io/fleta/core/transaction"
	"git.fleta.io/fleta/core/transaction/tx_account"
	"git.fleta.io/fleta/core/transaction/tx_utxo"
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

	signerHash := map[common.PublicHash]bool{}
	for _, signer := range signers {
		signerHash[signer] = true
	}
	if len(signers) != len(signerHash) {
		return ErrDuplicatedPublicKey
	}

	switch tx := t.(type) {
	case *tx_account.Transfer:
		fromAcc, err := ctx.LoadAccount(cn, tx.From)
		if err != nil {
			return err
		}
		if tx.Seq != fromAcc.Seq+1 {
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

		if fromAcc.Address.Type() == LockedAddressType && fromAcc.Balance.IsZero() {
			ctx.DeleteAccountHash[fromAcc.Address] = fromAcc
		}
	case *tx_account.TaggedTransfer:
		fromAcc, err := ctx.LoadAccount(cn, tx.From)
		if err != nil {
			return err
		}
		if tx.Seq != fromAcc.Seq+1 {
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

		if fromAcc.Address.Type() == LockedAddressType && fromAcc.Balance.IsZero() {
			ctx.DeleteAccountHash[fromAcc.Address] = fromAcc
		}
	case *tx_account.Burn:
		fromAcc, err := ctx.LoadAccount(cn, tx.From)
		if err != nil {
			return err
		}
		if tx.Seq != fromAcc.Seq+1 {
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

		if fromAcc.Balance.Less(tx.Amount) {
			return ErrInsuffcientBalance
		}
		fromAcc.Balance = fromAcc.Balance.Sub(tx.Amount)

		if fromAcc.Address.Type() == LockedAddressType && fromAcc.Balance.IsZero() {
			ctx.DeleteAccountHash[fromAcc.Address] = fromAcc
		}
	case *tx_account.SingleAccount:
		fromAcc, err := ctx.LoadAccount(cn, tx.From)
		if err != nil {
			return err
		}
		if tx.Seq != fromAcc.Seq+1 {
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

		addr := common.NewAddress(SingleAddressType, height, idx)
		if is, err := ctx.IsExistAccount(cn, addr); err != nil {
			return err
		} else if is {
			return ErrExistAddress
		} else {
			acc := CreateAccount(cn, addr, []common.PublicHash{tx.KeyHash})
			ctx.AccountHash[addr] = acc
		}

		if fromAcc.Address.Type() == LockedAddressType && fromAcc.Balance.IsZero() {
			ctx.DeleteAccountHash[fromAcc.Address] = fromAcc
		}
	case *tx_account.MultiSigAccount:
		fromAcc, err := ctx.LoadAccount(cn, tx.From)
		if err != nil {
			return err
		}
		if tx.Seq != fromAcc.Seq+1 {
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

		if len(tx.KeyHashes) < 2 {
			return ErrInvalidMultiSigKeyCount
		}
		if tx.Required == 0 || int(tx.Required) > len(tx.KeyHashes) {
			return ErrInvalidMultiSigRequired
		}

		addr := common.NewAddress(MultiSigAddressType, height, idx)
		if is, err := ctx.IsExistAccount(cn, addr); err != nil {
			return err
		} else if is {
			return ErrExistAddress
		} else {
			acc := CreateAccount(cn, addr, tx.KeyHashes)
			ctx.AccountHash[addr] = acc
			ctx.AccountDataHash[string(toAccountDataKey(addr, "Required"))] = []byte{byte(tx.Required)}
		}

		if fromAcc.Address.Type() == LockedAddressType && fromAcc.Balance.IsZero() {
			ctx.DeleteAccountHash[fromAcc.Address] = fromAcc
		}
	case *tx_account.Formulation:
		fromAcc, err := ctx.LoadAccount(cn, tx.From)
		if err != nil {
			return err
		}
		if tx.Seq != fromAcc.Seq+1 {
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

		addr := common.NewAddress(FormulationAddressType, height, idx)
		if is, err := ctx.IsExistAccount(cn, addr); err != nil {
			return err
		} else if is {
			return ErrExistAddress
		} else {
			acc := CreateAccount(cn, addr, signers)
			ctx.AccountHash[addr] = acc
			ctx.AccountDataHash[string(toAccountDataKey(addr, "PublicKey"))] = tx.PublicKey[:]
		}

		if fromAcc.Address.Type() == LockedAddressType && fromAcc.Balance.IsZero() {
			ctx.DeleteAccountHash[fromAcc.Address] = fromAcc
		}
	case *tx_account.RevokeFormulation:
		fromAcc, err := ctx.LoadAccount(cn, tx.From)
		if err != nil {
			return err
		}
		if tx.Seq != fromAcc.Seq+1 {
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

		formulationAcc, err := ctx.LoadAccount(cn, tx.FormulationAddress)
		if err != nil {
			return err
		}
		if err := ValidateSigners(ctx, cn, formulationAcc, signerHash); err != nil {
			return err
		}
		fromAcc.Balance = fromAcc.Balance.Add(formulationAcc.Balance).Add(cn.Config().FormulationCost)

		ctx.DeleteAccountHash[tx.FormulationAddress] = formulationAcc

		if fromAcc.Address.Type() == LockedAddressType && fromAcc.Balance.IsZero() {
			ctx.DeleteAccountHash[fromAcc.Address] = fromAcc
		}
	case *tx_account.Withdraw:
		fromAcc, err := ctx.LoadAccount(cn, tx.From)
		if err != nil {
			return err
		}
		if tx.Seq != fromAcc.Seq+1 {
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

		for n, vout := range tx.Vout {
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

			ctx.UTXOHash[transaction.MarshalID(height, idx, uint16(n))] = vout
		}

		if fromAcc.Address.Type() == LockedAddressType && fromAcc.Balance.IsZero() {
			ctx.DeleteAccountHash[fromAcc.Address] = fromAcc
		}
	case *tx_utxo.Assign:
		if len(signers) > 1 {
			return ErrExceedSignatureCount
		}

		insum := amount.NewCoinAmount(0, 0)
		for _, vin := range tx.Vin {
			if vin.Height >= height {
				return ErrExceedTransactionInputHeight
			}
			if utxo, err := cn.Unspent(vin.Height, vin.Index, vin.N); err != nil {
				return err
			} else {
				if !utxo.PublicHash.Equal(signers[0]) {
					return ErrMismatchPublicHash
				}
				ctx.SpentUTXOHash[vin.ID()] = true
				insum = insum.Add(utxo.Amount)
			}
		}

		outsum := amount.NewCoinAmount(0, 0)
		for n, vout := range tx.Vout {
			if vout.Amount.IsZero() {
				return ErrInvalidAmount
			}
			if vout.Amount.Less(cn.Config().DustAmount) {
				return ErrTooSmallAmount
			}
			ctx.UTXOHash[transaction.MarshalID(height, idx, uint16(n))] = vout
			outsum = outsum.Add(vout.Amount)
		}
		if insum.Less(outsum) {
			return ErrExceedTransactionInputValue
		}
		PaidFee := insum.Sub(outsum)
		Fee := cn.Fee(t)
		if !PaidFee.Equal(Fee) {
			return ErrInvalidTransactionFee
		}
	case *tx_utxo.Deposit:
		if len(signers) > 1 {
			return ErrExceedSignatureCount
		}
		if tx.Amount.IsZero() {
			return ErrInvalidAmount
		}
		if tx.Amount.Less(cn.Config().DustAmount) {
			return ErrTooSmallAmount
		}

		insum := amount.NewCoinAmount(0, 0)
		for _, vin := range tx.Vin {
			if vin.Height >= height {
				return ErrExceedTransactionInputHeight
			}
			if utxo, err := cn.Unspent(vin.Height, vin.Index, vin.N); err != nil {
				return err
			} else {
				if !utxo.PublicHash.Equal(signers[0]) {
					return ErrMismatchPublicHash
				}
				ctx.SpentUTXOHash[vin.ID()] = true
				insum = insum.Add(utxo.Amount)
			}
		}

		outsum := tx.Amount.Clone()
		for n, vout := range tx.Vout {
			if vout.Amount.IsZero() {
				return ErrInvalidAmount
			}
			if vout.Amount.Less(cn.Config().DustAmount) {
				return ErrTooSmallAmount
			}
			ctx.UTXOHash[transaction.MarshalID(height, idx, uint16(n))] = vout
			outsum = outsum.Add(vout.Amount)
		}
		if insum.Less(outsum) {
			return ErrExceedTransactionInputValue
		}
		PaidFee := insum.Sub(outsum)
		Fee := cn.Fee(t)
		if !PaidFee.Equal(Fee) {
			return ErrInvalidTransactionFee
		}

		toAcc, err := ctx.LoadAccount(cn, tx.Address)
		if err != nil {
			return err
		}
		toAcc.Balance = toAcc.Balance.Add(tx.Amount)
	case *tx_utxo.OpenAccount:
		if len(signers) > 1 {
			return ErrExceedSignatureCount
		}

		insum := amount.NewCoinAmount(0, 0)
		for _, vin := range tx.Vin {
			if vin.Height >= height {
				return ErrExceedTransactionInputHeight
			}
			if utxo, err := cn.Unspent(vin.Height, vin.Index, vin.N); err != nil {
				return err
			} else {
				if !utxo.PublicHash.Equal(signers[0]) {
					return ErrMismatchPublicHash
				}
				ctx.SpentUTXOHash[vin.ID()] = true
				insum = insum.Add(utxo.Amount)
			}
		}

		outsum := amount.NewCoinAmount(0, 0)
		for n, vout := range tx.Vout {
			if vout.Amount.IsZero() {
				return ErrInvalidAmount
			}
			if vout.Amount.Less(cn.Config().DustAmount) {
				return ErrTooSmallAmount
			}
			ctx.UTXOHash[transaction.MarshalID(height, idx, uint16(n))] = vout
			outsum = outsum.Add(vout.Amount)
		}
		if insum.Less(outsum) {
			return ErrExceedTransactionInputValue
		}
		PaidFee := insum.Sub(outsum)
		Fee := cn.Fee(t)
		if !PaidFee.Equal(Fee) {
			return ErrInvalidTransactionFee
		}

		addr := common.NewAddress(SingleAddressType, height, idx)
		if is, err := ctx.IsExistAccount(cn, addr); err != nil {
			return err
		} else if is {
			return ErrExistAddress
		} else {
			acc := CreateAccount(cn, addr, []common.PublicHash{tx.KeyHash})
			ctx.AccountHash[addr] = acc
		}
	}
	return nil
}

// ValidateSigners TODO
func ValidateSigners(ctx *ValidationContext, cn Provider, acc *account.Account, signerHash map[common.PublicHash]bool) error {
	matchCount := 0
	for _, addr := range acc.KeyHashes {
		if signerHash[addr] {
			matchCount++
		}
	}
	switch acc.Address.Type() {
	case MultiSigAddressType:
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
