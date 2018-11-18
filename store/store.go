package store

import (
	"bytes"
	"os"
	"path/filepath"
	"sync"
	"time"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/account"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/data"
	"git.fleta.io/fleta/core/db"
	"git.fleta.io/fleta/core/transaction"
	"github.com/dgraph-io/badger"
)

// Store saves the target chain state
// All updates are executed in one transaction with FileSync option
type Store struct {
	db          *badger.DB
	lockfile    *os.File
	ticker      *time.Ticker
	cache       storeCache
	accounter   *data.Accounter
	transactor  *data.Transactor
	SeqHashLock sync.Mutex
	SeqHash     map[common.Address]uint64
}

type storeCache struct {
	cached          bool
	height          uint32
	heightBlockHash hash.Hash256
}

// NewStore returns a Store
func NewStore(path string, act *data.Accounter, tran *data.Transactor) (*Store, error) {
	if !act.ChainCoord().Equal(tran.ChainCoord()) {
		return nil, ErrInvalidChainCoord
	}

	opts := badger.DefaultOptions
	opts.Dir = path
	opts.ValueDir = path
	opts.Truncate = true
	opts.SyncWrites = true
	lockfilePath := filepath.Join(opts.Dir, "LOCK")
	os.MkdirAll(path, os.ModeDir)

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
		db:         db,
		lockfile:   lockfile,
		ticker:     ticker,
		accounter:  act,
		transactor: tran,
		SeqHash:    map[common.Address]uint64{},
	}, nil
}

// Close terminate and clean store
func (st *Store) Close() {
	st.db.Close()
	st.lockfile.Close()
	st.ticker.Stop()
	st.db = nil
	st.lockfile = nil
	st.ticker = nil
}

// ChainCoord returns the coordinate of the target chain
func (st *Store) ChainCoord() *common.Coordinate {
	return st.accounter.ChainCoord()
}

// Accounter returns the accounter of the target chain
func (st *Store) Accounter() *data.Accounter {
	return st.accounter
}

// Transactor returns the transactor of the target chain
func (st *Store) Transactor() *data.Transactor {
	return st.transactor
}

// TargetHeight returns the height of the processing block
func (st *Store) TargetHeight() uint32 {
	return st.Height() + 1
}

// LastBlockHash returns the last block hash of the target chain
func (st *Store) LastBlockHash() hash.Hash256 {
	h, err := st.BlockHash(st.Height())
	if err != nil {
		// should have not reach
		panic(err)
	}
	return h
}

// Accounts returns all accounts in the store
func (st *Store) Accounts() ([]account.Account, error) {
	list := []account.Account{}
	if err := st.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Seek(tagAccount); it.ValidForPrefix(tagAccount); it.Next() {
			item := it.Item()
			value, err := item.Value()
			if err != nil {
				return err
			}
			acc, err := st.accounter.NewByType(account.Type(value[0]))
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

// Seq returns the sequence of the transaction
func (st *Store) Seq(addr common.Address) uint64 {
	st.SeqHashLock.Lock()
	defer st.SeqHashLock.Unlock()

	if seq, has := st.SeqHash[addr]; has {
		return seq
	} else {
		var seq uint64
		if err := st.db.View(func(txn *badger.Txn) error {
			item, err := txn.Get(toAccountSeqKey(addr))
			if err != nil {
				return err
			}
			value, err := item.Value()
			if err != nil {
				return err
			}
			seq = util.BytesToUint64(value)
			return nil
		}); err != nil {
			return 0
		}
		st.SeqHash[addr] = seq
		return seq
	}
}

// Account returns the account instance of the address from the store
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
		acc, err = st.accounter.NewByType(account.Type(value[0]))
		if err != nil {
			return err
		}
		if _, err := acc.ReadFrom(bytes.NewReader(value[1:])); err != nil {
			return err
		}
		return nil
	}); err != nil {
		if err == db.ErrNotExistKey {
			return nil, data.ErrNotExistAccount
		} else {
			return nil, err
		}
	}
	return acc, nil
}

// IsExistAccount checks that the account of the address is exist or not
func (st *Store) IsExistAccount(addr common.Address) (bool, error) {
	var isExist bool
	if err := st.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(toAccountKey(addr))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return db.ErrNotExistKey
			} else {
				return err
			}
		}
		isExist = !item.IsDeletedOrExpired()
		return nil
	}); err != nil {
		if err == db.ErrNotExistKey {
			return false, nil
		} else {
			return false, err
		}
	}
	return isExist, nil
}

// AccountBalance returns the account balance
func (st *Store) AccountBalance(addr common.Address) (*account.Balance, error) {
	var bc *account.Balance
	if err := st.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(toAccountBalanceKey(addr))
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
		bc = account.NewBalance()
		if _, err := bc.ReadFrom(bytes.NewReader(value)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		if err == db.ErrNotExistKey {
			return nil, data.ErrNotExistAccount
		} else {
			return nil, err
		}
	}
	return bc, nil
}

// AccountDataKeys returns all data keys of the account in the store
func (st *Store) AccountDataKeys(addr common.Address) ([][]byte, error) {
	list := [][]byte{}
	if err := st.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := toAccountDataKey(string(addr[:]))
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			list = append(list, item.Key())
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return list, nil
}

// AccountData returns the account data from the store
func (st *Store) AccountData(addr common.Address, name []byte) []byte {
	key := string(addr[:]) + string(name)
	var data []byte
	if err := st.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(toAccountDataKey(key))
		if err != nil {
			return err
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

// UTXOs returns all UTXOs in the store
func (st *Store) UTXOs() ([]*transaction.UTXO, error) {
	list := []*transaction.UTXO{}
	if err := st.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		for it.Seek(tagUTXO); it.ValidForPrefix(tagUTXO); it.Next() {
			item := it.Item()
			value, err := item.Value()
			if err != nil {
				return err
			}
			utxo := &transaction.UTXO{
				TxIn:  transaction.NewTxIn(fromUTXOKey(item.Key())),
				TxOut: transaction.NewTxOut(),
			}
			if _, err := utxo.TxOut.ReadFrom(bytes.NewReader(value)); err != nil {
				return err
			}
			list = append(list, utxo)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return list, nil
}

// UTXO returns the UTXO from the top store
func (st *Store) UTXO(id uint64) (*transaction.UTXO, error) {
	var utxo *transaction.UTXO
	if err := st.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(toUTXOKey(id))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return data.ErrNotExistUTXO
			} else {
				return err
			}
		}
		value, err := item.Value()
		if err != nil {
			return err
		}
		utxo = &transaction.UTXO{
			TxIn:  transaction.NewTxIn(id),
			TxOut: transaction.NewTxOut(),
		}
		if _, err := utxo.TxOut.ReadFrom(bytes.NewReader(value)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return utxo, nil
}

// BlockHash returns the hash of the block by height
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

// Block returns the block by height
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
		if _, err := b.ReadFromWith(bytes.NewReader(value), st.transactor); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return b, nil
}

// ObserverSigned returns the observer signatures of the block by height
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

// Height returns the current height of the target chain
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

// CustomData returns the custom data by the key from the store
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

// SetCustomData updates the custom data
func (st *Store) SetCustomData(key string, value []byte) error {
	return st.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set(toCustomData(key), value); err != nil {
			return err
		}
		return nil
	})
}

// DeleteCustomData deletes the custom data
func (st *Store) DeleteCustomData(key string) error {
	return st.db.Update(func(txn *badger.Txn) error {
		if err := txn.Delete(toCustomData(key)); err != nil {
			return err
		}
		return nil
	})
}

// StoreGenesis stores the genesis data with custom data
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
	st.SeqHashLock.Lock()
	for k, v := range ctd.SeqHash {
		st.SeqHash[k] = v
	}
	st.SeqHashLock.Unlock()
	return nil
}

// StoreBlock stores the block data with custom data
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
	st.SeqHashLock.Lock()
	for k, v := range ctd.SeqHash {
		st.SeqHash[k] = v
	}
	st.SeqHashLock.Unlock()
	return nil
}

func applyContextData(txn *badger.Txn, ctd *data.ContextData) error {
	for k, v := range ctd.SeqHash {
		if err := txn.Set(toAccountSeqKey(k), util.Uint64ToBytes(v)); err != nil {
			return err
		}
	}
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
		if _, has := ctd.AccountBalanceHash[k]; !has {
			ctd.AccountBalanceHash[k] = account.NewBalance()
		}
	}
	for k := range ctd.DeletedAccountHash {
		if err := txn.Delete(toAccountKey(k)); err != nil {
			return err
		}
		if err := txn.Delete(toAccountBalanceKey(k)); err != nil {
			return err
		}
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := toAccountDataKey(string(k[:]))
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			if err := txn.Delete(item.Key()); err != nil {
				return err
			}
		}
	}
	for k, v := range ctd.AccountBalanceHash {
		var buffer bytes.Buffer
		if _, err := v.WriteTo(&buffer); err != nil {
			return err
		}
		if err := txn.Set(toAccountBalanceKey(k), buffer.Bytes()); err != nil {
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
		if v.TxIn.ID() != k {
			return ErrInvalidTxInKey
		}
		if _, err := v.TxOut.WriteTo(&buffer); err != nil {
			return err
		}
		if err := txn.Set(toUTXOKey(k), buffer.Bytes()); err != nil {
			return err
		}
	}
	for k, v := range ctd.CreatedUTXOHash {
		var buffer bytes.Buffer
		if _, err := v.WriteTo(&buffer); err != nil {
			return err
		}
		if err := txn.Set(toUTXOKey(k), buffer.Bytes()); err != nil {
			return err
		}
	}
	for k := range ctd.DeletedUTXOHash {
		if err := txn.Delete(toUTXOKey(k)); err != nil {
			return err
		}
	}
	return nil
}
