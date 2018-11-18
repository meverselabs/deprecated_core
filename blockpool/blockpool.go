package blockpool

import (
	"sync"

	"git.fleta.io/fleta/common/queue"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/data"
)

// PoolItem is the block item that will be processed
type PoolItem struct {
	Block          *block.Block
	ObserverSigned *block.ObserverSigned
	Context        *data.Context
	Callback       AppendCallback
}

// BlockPool is the pool to provide block by the height
type BlockPool struct {
	sync.Mutex
	queue *queue.SortedQueue
}

// NewBlockPool returns a BlockPool
func NewBlockPool() *BlockPool {
	bp := &BlockPool{
		queue: queue.NewSortedQueue(),
	}
	return bp
}

// Append add the block data
func (bp *BlockPool) Append(b *block.Block, s *block.ObserverSigned, ctx *data.Context, cb AppendCallback) error {
	bp.Lock()
	defer bp.Unlock()

	item := &PoolItem{
		Block:          b,
		ObserverSigned: s,
		Context:        ctx,
		Callback:       cb,
	}
	bp.queue.Insert(item, uint64(b.Header.Height))
	return nil
}

// Pop get the block data of the height and remove all blocks that smaller than the target height
func (bp *BlockPool) Pop(Height uint32) *PoolItem {
	bp.Lock()
	defer bp.Unlock()

	v := bp.queue.Peek()
	if v != nil {
		item := v.(*PoolItem)
		for item.Block.Header.Height < Height {
			bp.queue.Pop()
			v = bp.queue.Peek()
			if v == nil {
				return nil
			}
			item = v.(*PoolItem)
		}
		if item.Block.Header.Height == Height {
			bp.queue.Pop()
			return item
		}
	}
	return nil
}

// AppendCallback is a function to be called when the block is processed
type AppendCallback func() error
