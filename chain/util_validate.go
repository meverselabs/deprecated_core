package chain

import (
	"runtime"
	"sync"
	"sync/atomic"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/rank"
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
