package txpool

import (
	"errors"
	"sync"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/queue"
	"git.fleta.io/fleta/core/transaction"
	"git.fleta.io/fleta/extension/account_tx"
)

// TransactionPool errors
var (
	ErrEmptyQueue            = errors.New("empty queue")
	ErrNotAccountTransaction = errors.New("not account transaction")
)

// TransactionPool TODO
type TransactionPool struct {
	sync.Mutex
	turnQ         *queue.Queue
	numberQ       *queue.Queue
	utxoQ         *queue.LinkedQueue
	turnOutHash   map[bool]int
	numberOutHash map[common.Address]int
	bucketHash    map[common.Address]*queue.SortedQueue
}

// NewTransactionPool TODO
func NewTransactionPool() *TransactionPool {
	tp := &TransactionPool{
		turnQ:         queue.NewQueue(),
		numberQ:       queue.NewQueue(),
		utxoQ:         queue.NewLinkedQueue(),
		turnOutHash:   map[bool]int{},
		numberOutHash: map[common.Address]int{},
		bucketHash:    map[common.Address]*queue.SortedQueue{},
	}
	return tp
}

// Push TODO
func (tp *TransactionPool) Push(t transaction.Transaction, sigs []common.Signature) error {
	tp.Lock()
	defer tp.Unlock()

	item := &PoolItem{
		Transaction: t,
		TxHash:      t.Hash(),
		Signatures:  sigs,
	}
	if t.IsUTXO() {
		tp.utxoQ.Push(t.Hash(), item)
		tp.turnQ.Push(true)
		return nil
	} else {
		tx, is := t.(account_tx.AccountTransaction)
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
		return nil
	}
}

// Remove TODO
func (tp *TransactionPool) Remove(t transaction.Transaction) {
	tp.Lock()
	defer tp.Unlock()

	if t.IsUTXO() {
		if tp.utxoQ.Remove(t.Hash()) != nil {
			tp.turnOutHash[true]++
		}
	} else {
		tx := t.(account_tx.AccountTransaction)
		addr := tx.From()
		if q, has := tp.bucketHash[addr]; has {
			for {
				item := q.Peek().(*PoolItem)
				if tx.Seq() < item.Transaction.(account_tx.AccountTransaction).Seq() {
					break
				}
				q.Pop()
				tp.turnOutHash[false]++
				tp.numberOutHash[addr]++
			}
			if q.Size() == 0 {
				delete(tp.bucketHash, addr)
			}
		}
	}
}

// Pop TODO
func (tp *TransactionPool) Pop(SeqCache SeqCache) *PoolItem {
	tp.Lock()
	defer tp.Unlock()

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
		return tp.utxoQ.Pop().(*PoolItem)
	} else {
		remain := tp.numberQ.Size()
		ignoreHash := map[common.Address]bool{}
		for {
			var addr common.Address
			for {
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
			item := q.Peek().(*PoolItem)
			lastSeq := SeqCache.Seq(addr)
			if item.Transaction.(account_tx.AccountTransaction).Seq() != lastSeq+1 {
				ignoreHash[addr] = true
				tp.numberQ.Push(addr)
				if remain > 0 {
					continue
				} else {
					return nil
				}
			}
			q.Pop()
			if q.Size() == 0 {
				delete(tp.bucketHash, addr)
			}
			return item
		}
	}
}

// PoolItem TODO
type PoolItem struct {
	Transaction transaction.Transaction
	TxHash      hash.Hash256
	Signatures  []common.Signature
}
