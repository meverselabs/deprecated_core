package txpool

import (
	"sync"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/queue"
	"git.fleta.io/fleta/core/transaction"
)

// AccountTransaction is an interface that defines common functions of account model based transactions
type AccountTransaction interface {
	Seq() uint64
	From() common.Address
}

// TransactionPool provides a transaction queue
// User can push transaction regardless of UTXO model based transactions or account model based transactions
// If the sequence of the account model based transaction is not reached to the next of the last sequence, it doens't poped
type TransactionPool struct {
	sync.Mutex
	turnQ         *queue.Queue
	numberQ       *queue.Queue
	utxoQ         *queue.LinkedQueue
	txidHash      map[hash.Hash256]bool
	turnOutHash   map[bool]int
	numberOutHash map[common.Address]int
	bucketHash    map[common.Address]*queue.SortedQueue
}

// NewTransactionPool returns a TransactionPool
func NewTransactionPool() *TransactionPool {
	tp := &TransactionPool{
		turnQ:         queue.NewQueue(),
		numberQ:       queue.NewQueue(),
		utxoQ:         queue.NewLinkedQueue(),
		txidHash:      map[hash.Hash256]bool{},
		turnOutHash:   map[bool]int{},
		numberOutHash: map[common.Address]int{},
		bucketHash:    map[common.Address]*queue.SortedQueue{},
	}
	return tp
}

// IsExist checks that the transaction hash is inserted or not
func (tp *TransactionPool) IsExist(TxHash hash.Hash256) bool {
	tp.Lock()
	defer tp.Unlock()

	return tp.txidHash[TxHash]
}

// Size returns the size of TxPool
func (tp *TransactionPool) Size() int {
	tp.Lock()
	defer tp.Unlock()

	sum := 0
	for _, v := range tp.turnOutHash {
		sum += v
	}
	return tp.turnQ.Size() - sum
}

// Push inserts the transaction and signatures of it by base model and sequence
// An UTXO model based transaction will be handled by FIFO
// An account model based transaction will be sorted by the sequence value
func (tp *TransactionPool) Push(t transaction.Transaction, sigs []common.Signature) error {
	tp.Lock()
	defer tp.Unlock()

	TxHash := t.Hash()
	if tp.txidHash[TxHash] {
		return ErrExistTransaction
	}

	item := &PoolItem{
		Transaction: t,
		TxHash:      TxHash,
		Signatures:  sigs,
	}
	if t.IsUTXO() {
		tp.utxoQ.Push(t.Hash(), item)
		tp.turnQ.Push(true)
	} else {
		tx, is := t.(AccountTransaction)
		if !is {
			return ErrNotAccountTransaction
		}
		addr := tx.From()
		q, has := tp.bucketHash[addr]
		if !has {
			q = queue.NewSortedQueue()
			tp.bucketHash[addr] = q
		}
		q.Insert(item, tx.Seq())
		tp.numberQ.Push(addr)
		tp.turnQ.Push(false)
	}
	tp.txidHash[TxHash] = true
	return nil
}

// Remove deletes the target transaction from the queue
// If it is an account model based transaction, it will be sorted by the sequence in the address
func (tp *TransactionPool) Remove(t transaction.Transaction) {
	tp.Lock()
	defer tp.Unlock()

	TxHash := t.Hash()
	if t.IsUTXO() {
		if tp.utxoQ.Remove(TxHash) != nil {
			tp.turnOutHash[true]++
			delete(tp.txidHash, TxHash)
		}
	} else {
		tx := t.(AccountTransaction)
		addr := tx.From()
		if q, has := tp.bucketHash[addr]; has {
			for {
				if q.Size() == 0 {
					break
				}
				v, _ := q.Peek()
				item := v.(*PoolItem)
				if tx.Seq() < item.Transaction.(AccountTransaction).Seq() {
					break
				}
				q.Pop()
				delete(tp.txidHash, item.Transaction.Hash())
				tp.turnOutHash[false]++
				tp.numberOutHash[addr]++
			}
			if q.Size() == 0 {
				delete(tp.bucketHash, addr)
			}
		}
	}
}

// Pop returns and removes the proper transaction
func (tp *TransactionPool) Pop(SeqCache SeqCache) *PoolItem {
	tp.Lock()
	defer tp.Unlock()

	return tp.UnsafePop(SeqCache)
}

// UnsafePop returns and removes the proper transaction without mutex locking
func (tp *TransactionPool) UnsafePop(SeqCache SeqCache) *PoolItem {
	var bTurn bool
	for {
		turn := tp.turnQ.Pop()
		if turn == nil {
			return nil
		}
		bTurn = turn.(bool)
		tout := tp.turnOutHash[bTurn]
		if tout > 0 {
			tp.turnOutHash[bTurn] = tout - 1
			continue
		}
		break
	}
	if bTurn {
		item := tp.utxoQ.Pop().(*PoolItem)
		delete(tp.txidHash, item.Transaction.Hash())
		return item
	} else {
		remain := tp.numberQ.Size()
		ignoreHash := map[common.Address]bool{}
		for {
			var addr common.Address
			for {
				if tp.numberQ.Size() == 0 {
					return nil
				}
				addr = tp.numberQ.Pop().(common.Address)
				remain--
				nout := tp.numberOutHash[addr]
				if nout > 0 {
					nout--
					if nout == 0 {
						delete(tp.numberOutHash, addr)
					} else {
						tp.numberOutHash[addr] = nout
					}
					continue
				}
				break
			}
			if ignoreHash[addr] {
				tp.numberQ.Push(addr)
				if remain > 0 {
					continue
				} else {
					return nil
				}
			}
			q := tp.bucketHash[addr]
			v, _ := q.Peek()
			item := v.(*PoolItem)
			lastSeq := SeqCache.Seq(addr)
			if item.Transaction.(AccountTransaction).Seq() != lastSeq+1 {
				ignoreHash[addr] = true
				tp.numberQ.Push(addr)
				if remain > 0 {
					continue
				} else {
					return nil
				}
			}
			q.Pop()
			delete(tp.txidHash, item.Transaction.Hash())
			if q.Size() == 0 {
				delete(tp.bucketHash, addr)
			}
			return item
		}
	}
}

// PoolItem represents the item of the queue
type PoolItem struct {
	Transaction transaction.Transaction
	TxHash      hash.Hash256
	Signatures  []common.Signature
}
