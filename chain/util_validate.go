package chain

import (
	"runtime"
	"sync"
	"sync/atomic"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/rank"
	"git.fleta.io/fleta/core/amount"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/transaction"
)

// ValidateBlockSigned TODO
func ValidateBlockSigned(b *block.Block, s *block.Signed, Top *rank.Rank) error {
	if len(s.ObserverSignatures) != ObserverSignatureRequired {
		return ErrInvalidBlockSignatureCount
	}
	h, err := b.Header.Hash()
	if err != nil {
		return err
	}
	if !h.Equal(s.BlockHash) {
		return ErrMismatchSignedBlockHash
	}

	{
		pubkey, err := common.RecoverPubkey(h, s.GeneratorSignature)
		if err != nil {
			return err
		}
		if !pubkey.Equal(Top.PublicKey) {
			return ErrInvalidGeneratorAddress
		}
	}

	for _, sig := range s.ObserverSignatures {
		pubkey, err := common.RecoverPubkey(h, sig)
		if err != nil {
			return err
		}
		if _, has := ObserverPubkeyHash[string(pubkey[:])]; !has {
			return ErrInvalidObserverPubkey
		}
	}
	return nil
}

// ValidateTransactionSignatures TODO
func ValidateTransactionSignatures(transactions []transaction.Transaction, signatures [][]common.Signature) ([]map[string]bool, []hash.Hash256, error) {
	var wg sync.WaitGroup
	cpuCnt := runtime.NumCPU()
	if len(transactions) < 500 {
		cpuCnt = 1
	}
	txCnt := len(transactions) / cpuCnt
	addrHashes := make([]map[string]bool, len(transactions))
	txHashes := make([]hash.Hash256, len(transactions))
	if len(transactions)%cpuCnt != 0 {
		txCnt++
	}
	errs := make(chan error, cpuCnt)
	var iHasError int64
	for i := 0; i < cpuCnt; i++ {
		lastCnt := (i + 1) * txCnt
		if lastCnt > len(transactions) {
			lastCnt = len(transactions)
		}
		wg.Add(1)
		go func(sidx int, txs []transaction.Transaction) {
			defer wg.Done()
			for q, tx := range txs {
				if iHasError > 0 {
					return
				}

				sigs := signatures[sidx+q]
				h, err := tx.Hash()
				if err != nil {
					errs <- err
					return
				}
				txHashes[sidx+q] = h

				addrHash := map[string]bool{}
				for _, sig := range sigs {
					pubkey, err := common.RecoverPubkey(h, sig)
					if err != nil {
						errs <- err
						return
					}
					addr := common.AddressFromPubkey(pubkey).String()
					if addrHash[addr] {
						errs <- ErrDuplicatedAddress
						return
					}
					addrHash[addr] = true
				}
				addrHashes[sidx+q] = addrHash
			}
		}(i*txCnt, transactions[i*txCnt:lastCnt])
	}
	if len(errs) > 0 {
		err := <-errs
		atomic.AddInt64(&iHasError, 1)
		return nil, nil, err
	}
	wg.Wait()
	return addrHashes, txHashes, nil
}

// ValidateResult TODO
type ValidateResult struct {
	TxHash      hash.Hash256
	SpentHash   map[uint64]bool
	UnspentHash map[uint64]*transaction.TxOut
}

// ValidateTransaction TODO
func ValidateTransaction(cn Chain, tx transaction.Transaction, sigs []common.Signature, idx uint16) (*ValidateResult, error) {
	height := cn.Height() + 1

	spentHash := map[uint64]bool{}
	unspentHash := map[uint64]*transaction.TxOut{}

	txHash, err := tx.Hash()
	if err != nil {
		return nil, err
	}
	addrs := make([]common.Address, 0, len(sigs))
	var insum amount.Amount
	hasCoinbase := false
	for _, vin := range tx.Vin() {
		if vin.IsCoinbase() {
			insum += cn.RewardValue()
			hasCoinbase = true
		} else {
			if vin.Height >= height {
				return nil, ErrExceedTransactionInputHeight
			}
			if utxo, err := cn.Unspent(vin.Height, vin.Index, vin.N); err != nil {
				return nil, err
			} else {
				spentHash[vin.ID()] = true
				insum += utxo.Amount

				for i, addr := range utxo.Addresses {
					if i >= len(addrs) {
						sig := sigs[i]
						pubkey, err := common.RecoverPubkey(txHash, sig)
						if err != nil {
							return nil, err
						}
						addrs = append(addrs, common.AddressFromPubkey(pubkey))
					}
					sigAddr := addrs[i]
					if !addr.Equal(sigAddr) {
						return nil, ErrMismatchAddress
					}
				}
			}
		}
	}

	if hasCoinbase {
		if len(tx.Vin()) > 1 {
			return nil, ErrInvalidCoinbaseTransaction
		}
	}
	var outsum amount.Amount
	for n, vout := range tx.Vout() {
		if vout.Amount == 0 {
			return nil, ErrInvalidAmount
		}
		unspentHash[transaction.MarshalID(height, idx, uint16(n))] = vout
		outsum += vout.Amount
	}
	if outsum > insum {
		return nil, ErrExceedTransactionInputValue
	}
	if hasCoinbase {
		if insum != outsum {
			return nil, ErrInvalidTransactionFee
		}
	} else {
		fee := insum - outsum
		calcultedFee := amount.CaclulateFee(len(tx.Vin()), len(tx.Vout()))
		if fee != calcultedFee {
			return nil, ErrInvalidTransactionFee
		}
	}
	result := &ValidateResult{
		TxHash:      txHash,
		SpentHash:   spentHash,
		UnspentHash: unspentHash,
	}
	return result, nil
}
