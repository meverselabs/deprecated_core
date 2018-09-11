package balance

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"time"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/store"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/chain"
	"git.fleta.io/fleta/core/transaction"
)

// Balance TODO
type Balance interface {
	Init(cn chain.Chain) error
	Close(cn chain.Chain)
	ProcessBlock(height uint32, b *block.Block) error
	Unspents(addr common.Address) ([]uint64, error)
}

// Base TODO
type Base struct {
	addrs             []common.Address
	addrCache         map[string]bool
	cacheStore        store.Store
	addrStore         store.Store
	height            uint32
	processedHeight   uint32
	maxAddrCacheCount int
	ticker            *time.Ticker
}

// NewBase TODO
func NewBase(addrs []common.Address, addrStore store.Store, cacheCount int) (*Base, error) {
	var processedHeight uint32
	if v, err := addrStore.Get([]byte("height")); err != nil {
	} else {
		processedHeight = util.BytesToUint32(v)
	}

	tempDir := "./temp"
	tempPath := filepath.Join(tempDir, "addrCache.dat")
	os.Remove(tempPath)
	cacheStore, err := store.NewBunt(tempPath)
	if err != nil {
		return nil, err
	}
	bc := &Base{
		addrs:             addrs,
		addrCache:         map[string]bool{},
		cacheStore:        cacheStore,
		addrStore:         addrStore,
		height:            processedHeight,
		processedHeight:   processedHeight,
		maxAddrCacheCount: cacheCount,
	}
	return bc, nil
}

// Init TODO
func (bc *Base) Init(cn chain.Chain) error {
	keys, _, err := bc.addrStore.Scan(nil)
	if err != nil {
		return err
	}
	for _, k := range keys {
		if !bytes.Equal(k, []byte("height")) {
			if err := bc.cacheAddressID(k); err != nil {
				return err
			}
		}
	}
	for bc.processedHeight < cn.Height() {
		if err := bc.processAddr(cn, true); err != nil {
			return err
		}
	}
	bc.height = bc.processedHeight

	bc.ticker = time.NewTicker(BalanceCacheWriteOutTime)
	go func() {
		for {
			select {
			case <-bc.ticker.C:
				bc.processAddr(cn, false)
			}
		}
	}()
	return nil
}

// Close TODO
func (bc *Base) Close(cn chain.Chain) {
	for bc.processedHeight < cn.Height() {
		if err := bc.processAddr(cn, false); err != nil {
			break
		}
	}
	bc.addrStore.Close()
	bc.cacheStore.Close()
}

func (bc *Base) processAddr(cn chain.Chain, isInit bool) error {
	height := bc.processedHeight + 1
	b, err := cn.Block(height)
	if err != nil {
		return err
	}

	addrHash := map[string]bool{}
	for _, addr := range bc.addrs {
		addrHash[addr.String()] = true
	}
	for idx, t := range b.Transactions {
		switch tx := t.(type) {
		case *transaction.Base:
			for _, vin := range tx.Vin {
				for _, addr := range bc.addrs {
					if addrHash[addr.String()] {
						bid := util.Uint64ToBytes(vin.ID())
						key := append(addr[:], bid...)
						if err := bc.addrStore.Delete(key); err != nil {
							if err != store.ErrNotExistKey {
								return err
							}
						}
						if isInit {
							if err := bc.deleteAddressID(key); err != nil {
								return err
							}
						}
					}
				}
			}
			for n, vout := range tx.Vout {
				for _, addr := range vout.Addresses {
					if addrHash[addr.String()] {
						bid := util.Uint64ToBytes(transaction.MarshalID(height, uint16(idx), uint16(n)))
						key := append(addr[:], bid...)
						if err := bc.addrStore.Set(key, []byte{0}); err != nil {
							if err != store.ErrNotExistKey {
								return err
							}
						}
						if isInit {
							if err := bc.cacheAddressID(key); err != nil {
								return err
							}
						}
					}
				}
			}
		}
	}
	if err := bc.addrStore.Set([]byte("height"), util.Uint32ToBytes(height)); err != nil {
		return err
	}
	bc.processedHeight = height
	return nil
}

// ProcessBlock TODO
func (bc *Base) ProcessBlock(height uint32, b *block.Block) error {
	if height != bc.height+1 {
		return ErrNotProcessTarget
	}
	addrHash := map[string]bool{}
	for _, addr := range bc.addrs {
		addrHash[addr.String()] = true
	}
	for idx, t := range b.Transactions {
		switch tx := t.(type) {
		case *transaction.Base:
			for _, vin := range tx.Vin {
				for _, addr := range bc.addrs {
					if addrHash[addr.String()] {
						bid := util.Uint64ToBytes(vin.ID())
						key := append(addr[:], bid...)
						if err := bc.deleteAddressID(key); err != nil {
							return err
						}
					}
				}
			}
			for n, vout := range tx.Vout {
				for _, addr := range vout.Addresses {
					if addrHash[addr.String()] {
						bid := util.Uint64ToBytes(transaction.MarshalID(height, uint16(idx), uint16(n)))
						key := append(addr[:], bid...)
						if err := bc.cacheAddressID(key); err != nil {
							return err
						}
					}
				}
			}
		}
	}
	bc.height = height
	log.Println("BalanceCache", len(bc.addrCache))
	return nil
}

// Unspents TODO
func (bc *Base) Unspents(addr common.Address) ([]uint64, error) {
	ids := []uint64{}
	for s := range bc.addrCache {
		k := []byte(s)
		if bytes.Equal(k[:len(addr)], addr[:]) {
			ids = append(ids, util.BytesToUint64(k[len(addr):]))
		}
	}
	keys, _, err := bc.cacheStore.Scan(addr[:])
	if err != nil {
		return nil, err
	}
	for _, k := range keys {
		ids = append(ids, util.BytesToUint64(k[len(addr):]))
	}
	return ids, nil
}

func (bc *Base) cacheAddressID(key []byte) error {
	if len(bc.addrCache) >= bc.maxAddrCacheCount {
		for id := range bc.addrCache {
			delete(bc.addrCache, id)

			if err := bc.cacheStore.Set([]byte(id), []byte{0}); err != nil {
				return err
			}
			break
		}
	}
	bc.addrCache[string(key)] = true
	return nil
}

func (bc *Base) deleteAddressID(key []byte) error {
	if _, has := bc.addrCache[string(key)]; !has {
		if err := bc.cacheStore.Delete(key); err != nil {
			if err != store.ErrNotExistKey {
				return err
			}
		}
	} else {
		delete(bc.addrCache, string(key))
	}
	return nil
}
