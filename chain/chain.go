package chain

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/rank"
	"git.fleta.io/fleta/common/store"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/amount"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/level"
	"git.fleta.io/fleta/core/transaction"
)

// Provider TODO
type Provider interface {
	Height() uint32
	RewardValue() amount.Amount
	Block(height uint32) (*block.Block, error)
	BlockByHash(h hash.Hash256) (*block.Block, error)
	BlockHash(height uint32) (hash.Hash256, error)
	BlockSigned(height uint32) (*block.Signed, error)
	BlockSignedByHash(h hash.Hash256) (*block.Signed, error)
	Transactions(height uint32) ([]transaction.Transaction, error)
	Transaction(height uint32, index uint16) (transaction.Transaction, error)
	Unspent(height uint32, index uint16, n uint16) (*UTXO, error)
}

// Chain TODO
type Chain interface {
	Provider
	Init() error
	Close()
	ConnectBlock(b *block.Block, s *block.Signed, Top *rank.Rank) (map[uint64]bool, error)
}

// Base TODO
type Base struct {
	blockStore        store.Store
	utxoStore         store.Store
	height            uint32
	processedHeight   uint32
	utxoCache         map[uint64]*transaction.TxOut
	cacheStore        store.Store
	maxUTXOCacheCount int
	ticker            *time.Ticker
}

// NewBase TODO
func NewBase(blockStore store.Store, utxoStore store.Store, cacheCount int) (*Base, error) {
	var height uint32
	if bh, err := blockStore.Get([]byte("height")); err != nil {
	} else {
		height = util.BytesToUint32(bh)
	}
	var processedHeight uint32
	if v, err := utxoStore.Get([]byte("height")); err != nil {
	} else {
		processedHeight = util.BytesToUint32(v)
	}

	tempDir := "./temp"
	os.MkdirAll(tempDir, os.ModeDir)
	tempPath := filepath.Join(tempDir, "utxoCache.dat")
	os.Remove(tempPath)
	cacheStore, err := store.NewBunt(tempPath)
	if err != nil {
		return nil, err
	}
	cn := &Base{
		blockStore:        blockStore,
		utxoStore:         utxoStore,
		cacheStore:        cacheStore,
		height:            height,
		processedHeight:   processedHeight,
		utxoCache:         map[uint64]*transaction.TxOut{},
		maxUTXOCacheCount: cacheCount,
	}
	return cn, nil
}

// Init load UTXO to cache from store
func (cn *Base) Init() error {
	keys, values, err := cn.utxoStore.Scan(nil)
	if err != nil {
		return err
	}
	for i, k := range keys {
		v := values[i]
		if !bytes.Equal(k, []byte("height")) {
			vout := new(transaction.TxOut)
			if _, err := vout.ReadFrom(bytes.NewReader(v)); err != nil {
				return err
			}
			id := util.BytesToUint64(k)
			if err := cn.cacheUTXO(id, vout); err != nil {
				return err
			}
		}
	}
	for cn.processedHeight < cn.Height() {
		if err := cn.processUTXO(true); err != nil {
			return err
		}
	}

	cn.ticker = time.NewTicker(UTXOCacheWriteOutTime)
	go func() {
		for {
			select {
			case <-cn.ticker.C:
				cn.processUTXO(false)
			}
		}
	}()
	return nil
}

// RewardValue TODO
func (cn *Base) RewardValue() amount.Amount {
	return 1
}

// Close TODO
func (cn *Base) Close() {
	for cn.processedHeight < cn.Height() {
		if err := cn.processUTXO(false); err != nil {
			break
		}
	}
	cn.blockStore.Close()
	cn.utxoStore.Close()
	cn.cacheStore.Close()
}

func (cn *Base) processUTXO(isInit bool) error {
	height := cn.processedHeight + 1
	b, err := cn.Block(height)
	if err != nil {
		return err
	}

	spentHash := map[uint64]bool{}
	unspentHash := map[uint64]*transaction.TxOut{}
	for idx, tx := range b.Transactions {
		for _, vin := range tx.Vin() {
			if vin.IsCoinbase() {
			} else {
				spentHash[vin.ID()] = true
			}
		}
		for n, vout := range tx.Vout() {
			unspentHash[transaction.MarshalID(height, uint16(idx), uint16(n))] = vout
		}
	}
	for id, vout := range unspentHash {
		bid := util.Uint64ToBytes(id)
		var buffer bytes.Buffer
		if _, err := vout.WriteTo(&buffer); err != nil {
			return err
		}
		if err := cn.utxoStore.Set(bid, buffer.Bytes()); err != nil {
			return err
		}
		if isInit {
			if err := cn.cacheUTXO(id, vout); err != nil {
				return err
			}
		}
	}
	for id := range spentHash {
		bid := util.Uint64ToBytes(id)
		if err := cn.utxoStore.Delete(bid); err != nil {
			return err
		}
		if isInit {
			if err := cn.deleteUTXO(id); err != nil {
				return err
			}
		}
	}

	if err := cn.utxoStore.Set([]byte("height"), util.Uint32ToBytes(height)); err != nil {
		return err
	}
	cn.processedHeight = height
	return nil
}

type profile struct {
	sync.Mutex
	BeginHash    map[string]time.Time
	DurationHash map[string]time.Duration
}

// Begin TODO
func (p *profile) Begin(name string) {
	p.Lock()
	defer p.Unlock()

	p.BeginHash[name] = time.Now()
	log.Println(name, "BEGIN")
}

// End TODO
func (p *profile) End(name string) {
	p.Lock()
	defer p.Unlock()

	p.DurationHash[name] = time.Now().Sub(p.BeginHash[name])
	log.Println(name, "END")
}

// End TODO
func (p *profile) Print() {
	p.Lock()
	defer p.Unlock()

	for name, t := range p.DurationHash {
		log.Println(name, t/time.Millisecond)
	}
}

// ConnectBlock TODO
func (cn *Base) ConnectBlock(b *block.Block, s *block.Signed, Top *rank.Rank) (map[uint64]bool, error) {
	var prevHash hash.Hash256
	if cn.Height() == 0 {
		prevHash = GenesisHash
	} else {
		h, err := cn.BlockHash(cn.Height())
		if err != nil {
			return nil, err
		}
		prevHash = h
	}
	if !b.Header.HashPrevBlock.Equal(prevHash) {
		return nil, ErrMismatchHashPrevBlock
	}

	if err := ValidateBlockSigned(b, s, Top); err != nil {
		return nil, err
	}
	height := cn.Height() + 1

	txHashes := make([]hash.Hash256, 0, len(b.Transactions))

	spentHash := map[uint64]bool{}
	unspentHash := map[uint64]*transaction.TxOut{}
	for idx, tx := range b.Transactions {
		result, err := ValidateTransaction(cn, tx, b.TransactionSignatures[idx], uint16(idx))
		if err != nil {
			return nil, err
		}
		txHashes = append(txHashes, result.TxHash)

		for k, v := range result.SpentHash {
			spentHash[k] = v
		}
		for k, v := range result.UnspentHash {
			unspentHash[k] = v
		}
	}

	root, err := level.BuildLevelRoot(txHashes)
	if err != nil {
		return nil, err
	}
	if !b.Header.HashLevelRoot.Equal(hash.TwoHash(prevHash, root)) {
		return nil, ErrMismatchHashLevelRoot
	}

	if err := cn.writeBlock(height, b, s); err != nil {
		return nil, err
	}

	for id, vout := range unspentHash {
		if err := cn.cacheUTXO(id, vout); err != nil {
			return nil, err
		}
	}
	for id := range spentHash {
		if err := cn.deleteUTXO(id); err != nil {
			return nil, err
		}
	}

	if err := cn.blockStore.Set([]byte("height"), util.Uint32ToBytes(height)); err != nil {
		return nil, err
	}
	cn.height = height

	return spentHash, nil
}

// Height TODO
func (cn *Base) Height() uint32 {
	return cn.height
}

// Block TODO
func (cn *Base) Block(height uint32) (*block.Block, error) {
	if height > cn.Height() {
		return nil, ErrExceedChainHeight
	}
	if v, err := cn.blockStore.Get(toHeightBlockKey(height)); err != nil {
		return nil, err
	} else {
		b := new(block.Block)
		if _, err := b.ReadFrom(bytes.NewReader(v)); err != nil {
			return nil, err
		}
		return b, nil
	}
}

// BlockByHash TODO
func (cn *Base) BlockByHash(h hash.Hash256) (*block.Block, error) {
	if height, err := cn.blockStore.Get(toHashBlockHeightKey(h)); err != nil {
		return nil, err
	} else {
		return cn.Block(util.BytesToUint32(height))
	}
}

// BlockHash TODO
func (cn *Base) BlockHash(height uint32) (hash.Hash256, error) {
	if height > cn.Height() {
		return hash.Hash256{}, ErrExceedChainHeight
	}
	if v, err := cn.blockStore.Get(toHeightBlockHashKey(height)); err != nil {
		return hash.Hash256{}, err
	} else {
		var h hash.Hash256
		if _, err := h.ReadFrom(bytes.NewReader(v)); err != nil {
			return hash.Hash256{}, err
		}
		return h, nil
	}
}

// BlockSigned TODO
func (cn *Base) BlockSigned(height uint32) (*block.Signed, error) {
	if height > cn.Height() {
		return nil, ErrExceedChainHeight
	}
	if v, err := cn.blockStore.Get(toHeightBlockSignedKey(height)); err != nil {
		return nil, err
	} else {
		s := new(block.Signed)
		if _, err := s.ReadFrom(bytes.NewReader(v)); err != nil {
			return nil, err
		}
		return s, nil
	}
}

// BlockSignedByHash TODO
func (cn *Base) BlockSignedByHash(h hash.Hash256) (*block.Signed, error) {
	if height, err := cn.blockStore.Get(toHashBlockHeightKey(h)); err != nil {
		return nil, err
	} else {
		return cn.BlockSigned(util.BytesToUint32(height))
	}
}

// Transactions TODO
func (cn *Base) Transactions(height uint32) ([]transaction.Transaction, error) {
	if height > cn.Height() {
		return nil, ErrExceedChainHeight
	}
	if b, err := cn.Block(height); err != nil {
		return nil, err
	} else {
		return b.Transactions, nil
	}
}

// Transaction TODO
func (cn *Base) Transaction(height uint32, index uint16) (transaction.Transaction, error) {
	if height > cn.Height() {
		return nil, ErrExceedChainHeight
	}
	if txs, err := cn.Transactions(height); err != nil {
		return nil, err
	} else {
		if index >= uint16(len(txs)) {
			return nil, ErrExceedTransactionIndex
		}
		return txs[index], nil
	}
}

// Unspent TODO
func (cn *Base) Unspent(height uint32, index uint16, n uint16) (*UTXO, error) {
	if height > cn.Height() {
		return nil, ErrExceedChainHeight
	}
	id := transaction.MarshalID(height, index, n)
	utxo := &UTXO{
		TxIn: transaction.TxIn{
			Height: height,
			Index:  index,
			N:      n,
		},
	}
	if vout, has := cn.utxoCache[id]; !has {
		if v, err := cn.cacheStore.Get(util.Uint64ToBytes(id)); err != nil {
			return nil, err
		} else if _, err := utxo.TxOut.ReadFrom(bytes.NewReader(v)); err != nil {
			return nil, err
		}
	} else {
		utxo.TxOut = *vout
	}
	return utxo, nil
}

func (cn *Base) cacheUTXO(id uint64, vout *transaction.TxOut) error {
	if len(cn.utxoCache) >= cn.maxUTXOCacheCount {
		for id, u := range cn.utxoCache {
			delete(cn.utxoCache, id)

			bid := util.Uint64ToBytes(id)
			var buffer bytes.Buffer
			if _, err := u.WriteTo(&buffer); err != nil {
				return err
			}
			if err := cn.cacheStore.Set(bid, buffer.Bytes()); err != nil {
				return err
			}
			break
		}
	}
	cn.utxoCache[id] = vout
	return nil
}

func (cn *Base) deleteUTXO(id uint64) error {
	if _, has := cn.utxoCache[id]; !has {
		bid := util.Uint64ToBytes(id)
		if err := cn.cacheStore.Delete(bid); err != nil {
			if err != store.ErrNotExistKey {
				return err
			}
		}
	} else {
		delete(cn.utxoCache, id)
	}
	return nil
}

func (cn *Base) writeBlock(height uint32, b *block.Block, s *block.Signed) error {
	{
		var buffer bytes.Buffer
		if _, err := b.WriteTo(&buffer); err != nil {
			return err
		} else if err := cn.blockStore.Set(toHeightBlockKey(height), buffer.Bytes()); err != nil {
			return err
		}
	}

	{
		var buffer bytes.Buffer
		if _, err := s.WriteTo(&buffer); err != nil {
			return err
		} else if err := cn.blockStore.Set(toHeightBlockSignedKey(height), buffer.Bytes()); err != nil {
			return err
		}
	}

	if h, err := b.Header.Hash(); err != nil {
		return err
	} else if err := cn.blockStore.Set(toHeightBlockHashKey(height), h[:]); err != nil {
		return err
	} else if err := cn.blockStore.Set(toHashBlockHeightKey(h), util.Uint32ToBytes(height)); err != nil {
		return err
	}
	return nil
}
