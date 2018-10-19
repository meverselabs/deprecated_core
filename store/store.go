package store

import (
	"bytes"
	"os"
	"path/filepath"
	"time"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/account"
	"git.fleta.io/fleta/core/accounter"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/data"
	"git.fleta.io/fleta/core/db"
	"git.fleta.io/fleta/core/transaction"
	"github.com/dgraph-io/badger"
)

// Store TODO
type Store struct {
	db       *badger.DB
	lockfile *os.File
	ticker   *time.Ticker
	cache    storeCache
	coord    *common.Coordinate
	SeqHash  map[common.Address]uint64
}

type storeCache struct {
	cached          bool
	height          uint32
	heightBlockHash hash.Hash256
}

// NewStore TODO
func NewStore(coord *common.Coordinate) (*Store, error) {
	path := "./" + coord.String()
	opts := badger.DefaultOptions
	opts.Dir = path
	opts.ValueDir = path
	opts.Truncate = true
	opts.SyncWrites = true
	lockfilePath := filepath.Join(opts.Dir, "LOCK")

	os.Remove(lockfilePath)

	db, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	lockfile, err := os.OpenFile(lockfilePath, os.O_EXCL, 0)
	if err != nil {
		return nil, err
	}

	{
	again:
		if err := db.RunValueLogGC(0.7); err != nil {
		} else {
			goto again
		}
	}

	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for range ticker.C {
		again:
			if err := db.RunValueLogGC(0.7); err != nil {
			} else {
				goto again
			}
		}
	}()

	return &Store{
		db:       db,
		lockfile: lockfile,
		ticker:   ticker,
		coord:    coord,
		SeqHash:  map[common.Address]uint64{},
	}, nil
}

// ChainCoord TODO
func (st *Store) ChainCoord() *common.Coordinate {
	return st.coord
}

// Accounts TODO
func (st *Store) Accounts() ([]account.Account, error) {
	list := []account.Account{}
	if err := st.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte{tagAccount}
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			value, err := item.Value()
			if err != nil {
				return err
			}
			act, err := accounter.ByCoord(st.coord)
			if err != nil {
				return err
			}
			acc, err := act.NewByType(account.Type(value[0]))
			if err != nil {
				return err
			}
			if _, err := acc.ReadFrom(bytes.NewReader(value[1:])); err != nil {
				return err
			}
			list = append(list, acc)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return list, nil
}

// Seq TODO
func (st *Store) Seq(addr common.Address) uint64 {
	if seq, has := st.SeqHash[addr]; has {
		return seq
	} else {
		if acc, err := st.Account(addr); err != nil {
			return 0
		} else {
			seq := acc.Seq()
			st.SeqHash[addr] = seq
			return seq
		}
	}
}

// Account TODO
func (st *Store) Account(addr common.Address) (account.Account, error) {
	var acc account.Account
	if err := st.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(toAccountKey(addr))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return db.ErrNotExistKey
			} else {
				return err
			}
		}
		value, err := item.Value()
		if err != nil {
			return err
		}
		act, err := accounter.ByCoord(st.coord)
		if err != nil {
			return err
		}
		acc, err = act.NewByType(account.Type(value[0]))
		if err != nil {
			return err
		}
		if _, err := acc.ReadFrom(bytes.NewReader(value[1:])); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return acc, nil
}

// AccountData TODO
func (st *Store) AccountData(key string) []byte {
	var data []byte
	if err := st.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(toAccountDataKey(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return db.ErrNotExistKey
			} else {
				return err
			}
		}
		value, err := item.Value()
		if err != nil {
			return err
		}
		data = value
		return nil
	}); err != nil {
		return nil
	}
	return data
}

// UTXO TODO
func (st *Store) UTXO(id uint64) (*transaction.UTXO, error) {
	var utxo *transaction.UTXO
	if err := st.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(toTxInUTXOKey(id))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return db.ErrNotExistKey
			} else {
				return err
			}
		}
		value, err := item.Value()
		if err != nil {
			return err
		}
		utxo = transaction.NewUTXO()
		if _, err := utxo.ReadFrom(bytes.NewReader(value)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return utxo, nil
}

// BlockHash TODO
func (st *Store) BlockHash(height uint32) (hash.Hash256, error) {
	if st.cache.cached {
		if st.cache.height == height {
			return st.cache.heightBlockHash, nil
		}
	}

	var h hash.Hash256
	if err := st.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(toHeightBlockHashKey(height))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return db.ErrNotExistKey
			} else {
				return err
			}
		}
		value, err := item.Value()
		if err != nil {
			return err
		}
		if _, err := h.ReadFrom(bytes.NewReader(value)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return hash.Hash256{}, err
	}
	return h, nil
}

// Block TODO
func (st *Store) Block(height uint32) (*block.Block, error) {
	var b *block.Block
	if err := st.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(toHeightBlockKey(height))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return db.ErrNotExistKey
			} else {
				return err
			}
		}
		value, err := item.Value()
		if err != nil {
			return err
		}
		b = new(block.Block)
		if _, err := b.ReadFrom(bytes.NewReader(value)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return b, nil
}

// ObserverSigned TODO
func (st *Store) ObserverSigned(height uint32) (*block.ObserverSigned, error) {
	var s *block.ObserverSigned
	if err := st.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(toHeightBlockKey(height))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return db.ErrNotExistKey
			} else {
				return err
			}
		}
		value, err := item.Value()
		if err != nil {
			return err
		}
		s = new(block.ObserverSigned)
		if _, err := s.ReadFrom(bytes.NewReader(value)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return s, nil
}

// Height TODO
func (st *Store) Height() uint32 {
	if st.cache.cached {
		return st.cache.height
	}

	var height uint32
	st.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("height"))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return db.ErrNotExistKey
			} else {
				return err
			}
		}
		value, err := item.Value()
		if err != nil {
			return err
		}
		height = util.BytesToUint32(value)
		return nil
	})
	return height
}

// CustomData TODO
func (st *Store) CustomData(key string) []byte {
	var bs []byte
	st.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(toCustomData(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return db.ErrNotExistKey
			} else {
				return err
			}
		}
		value, err := item.Value()
		if err != nil {
			return err
		}
		bs = value
		return nil
	})
	return bs
}

// SetCustomData TODO
func (st *Store) SetCustomData(key string, value []byte) error {
	return st.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set(toCustomData(key), value); err != nil {
			return err
		}
		return nil
	})
}

// DeleteCustomData TODO
func (st *Store) DeleteCustomData(key string) error {
	return st.db.Update(func(txn *badger.Txn) error {
		if err := txn.Delete(toCustomData(key)); err != nil {
			return err
		}
		return nil
	})
}

// StoreGenesis TODO
func (st *Store) StoreGenesis(ctd *data.ContextData, GenesisHash hash.Hash256, customHash map[string][]byte) error {
	if _, err := st.BlockHash(0); err != nil {
		if err != db.ErrNotExistKey {
			return err
		}
	} else {
		return ErrAlreadyExistGenesis
	}
	if err := st.db.Update(func(txn *badger.Txn) error {
		{
			if err := txn.Set(toHeightBlockHashKey(0), GenesisHash[:]); err != nil {
				return err
			}
			bsHeight := util.Uint32ToBytes(0)
			if err := txn.Set(toHashBlockHeightKey(GenesisHash), bsHeight); err != nil {
				return err
			}
			if err := txn.Set([]byte("height"), bsHeight); err != nil {
				return err
			}
		}
		if err := applyContextData(txn, ctd); err != nil {
			return err
		}
		for k, v := range customHash {
			if err := txn.Set(toCustomData(k), v); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	st.cache.height = 0
	st.cache.heightBlockHash = GenesisHash
	st.cache.cached = true
	for k, v := range ctd.SeqHash {
		st.SeqHash[k] = v
	}
	return nil
}

// StoreBlock TODO
func (st *Store) StoreBlock(ctd *data.ContextData, b *block.Block, s *block.ObserverSigned, customHash map[string][]byte) error {
	blockHash := b.Header.Hash()
	if err := st.db.Update(func(txn *badger.Txn) error {
		{
			var buffer bytes.Buffer
			if _, err := b.WriteTo(&buffer); err != nil {
				return err
			}
			if err := txn.Set(toHeightBlockKey(b.Header.Height), buffer.Bytes()); err != nil {
				return err
			}
		}
		{
			if err := txn.Set(toHeightBlockHashKey(b.Header.Height), blockHash[:]); err != nil {
				return err
			}
			bsHeight := util.Uint32ToBytes(b.Header.Height)
			if err := txn.Set(toHashBlockHeightKey(blockHash), bsHeight); err != nil {
				return err
			}
			if err := txn.Set([]byte("height"), bsHeight); err != nil {
				return err
			}
		}
		{
			var buffer bytes.Buffer
			if _, err := s.WriteTo(&buffer); err != nil {
				return err
			}
			if err := txn.Set(toHeightObserverSignedKey(b.Header.Height), buffer.Bytes()); err != nil {
				return err
			}
		}
		if err := applyContextData(txn, ctd); err != nil {
			return err
		}
		for k, v := range customHash {
			if err := txn.Set(toCustomData(k), v); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}
	st.cache.height = b.Header.Height
	st.cache.heightBlockHash = blockHash
	st.cache.cached = true
	for k, v := range ctd.SeqHash {
		st.SeqHash[k] = v
	}
	return nil
}

func applyContextData(txn *badger.Txn, ctd *data.ContextData) error {
	for k, v := range ctd.AccountHash {
		var buffer bytes.Buffer
		buffer.WriteByte(byte(v.Type()))
		if _, err := v.WriteTo(&buffer); err != nil {
			return err
		}
		if err := txn.Set(toAccountKey(k), buffer.Bytes()); err != nil {
			return err
		}
	}
	for k, v := range ctd.CreatedAccountHash {
		var buffer bytes.Buffer
		buffer.WriteByte(byte(v.Type()))
		if _, err := v.WriteTo(&buffer); err != nil {
			return err
		}
		if err := txn.Set(toAccountKey(k), buffer.Bytes()); err != nil {
			return err
		}
	}
	for k := range ctd.DeletedAccountHash {
		if err := txn.Delete(toAccountKey(k)); err != nil {
			return err
		}
	}
	for k, v := range ctd.AccountDataHash {
		if err := txn.Set(toAccountDataKey(k), []byte(v)); err != nil {
			return err
		}
	}
	for k := range ctd.DeletedAccountDataHash {
		if err := txn.Delete(toAccountDataKey(k)); err != nil {
			return err
		}
	}
	for k, v := range ctd.UTXOHash {
		var buffer bytes.Buffer
		if _, err := v.WriteTo(&buffer); err != nil {
			return err
		}
		if err := txn.Set(toTxInUTXOKey(k), buffer.Bytes()); err != nil {
			return err
		}
	}
	for k, v := range ctd.CreatedUTXOHash {
		var buffer bytes.Buffer
		if _, err := v.WriteTo(&buffer); err != nil {
			return err
		}
		if err := txn.Set(toTxInUTXOKey(k), buffer.Bytes()); err != nil {
			return err
		}
	}
	for k := range ctd.DeletedUTXOHash {
		if err := txn.Delete(toTxInUTXOKey(k)); err != nil {
			return err
		}
	}
	return nil
}
