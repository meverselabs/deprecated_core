package txpool

import (
	"encoding/hex"
	"testing"
	"time"

	"git.fleta.io/fleta/common/hash"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/framework/log"

	"git.fleta.io/fleta/core/amount"
	"git.fleta.io/fleta/core/key"
	"git.fleta.io/fleta/core/transaction"

	"git.fleta.io/fleta/extension/account_tx"
	"git.fleta.io/fleta/extension/utxo_tx"
)

type testCtx struct {
	m map[common.Address]uint64
}

func (ctx *testCtx) Seq(addr common.Address) uint64 {
	if v, has := ctx.m[addr]; has {
		return v
	}
	ctx.m[addr] = 0
	return ctx.m[addr]
}

func (ctx *testCtx) updateSeq(tx AccountTransaction, seq uint64) {
	ctx.m[tx.From()] = seq
}

func testSignature(tx transaction.Transaction) (transaction.Transaction, []common.Signature) {
	MemKey := getMemKey("f6d94eb4131bda99277f3bc44fc498527ecd43177872a2b58ee7008225037a18")

	sig, err := MemKey.Sign(tx.Hash())
	if err != nil {
		panic(err)
	}
	return tx, []common.Signature{sig}
}

func testAddr(i int) common.Address {
	return common.NewAddress(common.NewCoordinate(1, 1), common.NewCoordinate(1, 2), uint64(i))
}

func getMemKey(h string) *key.MemoryKey {
	var MemKey *key.MemoryKey
	if bs, err := hex.DecodeString(h); err != nil {
		panic(err)
	} else if key, err := key.NewMemoryKeyFromBytes(bs); err != nil {
		panic(err)
	} else {
		MemKey = key
	}

	return MemKey

}

func testUTXOTx(i byte) transaction.Transaction {
	coord := common.NewCoordinate(0, 0)

	MemKey1 := getMemKey("0ad8e4cd0c22664e977b2a634e3a9ddf7f5985c6a7fa9ec389127bf272bbafbb")
	MemKey2 := getMemKey(hash.Hash([]byte{i}).String())

	t := &utxo_tx.OpenAccount{
		Base: utxo_tx.Base{
			Base: transaction.Base{
				ChainCoord_: coord,
				Timestamp_:  uint64(time.Now().UnixNano()),
				Type_:       transaction.Type(255),
			},
			Vin: []*transaction.TxIn{
				&transaction.TxIn{
					Height: uint32(1),
					Index:  uint16(1),
					N:      uint16(1),
				},
			},
		},
		Vout: []*transaction.TxOut{
			&transaction.TxOut{
				Amount:     amount.NewCoinAmount(10, 10),
				PublicHash: common.NewPublicHash(MemKey1.PublicKey()),
			},
		},
		KeyHash: common.NewPublicHash(MemKey2.PublicKey()),
	}

	return t
}

func testAccTx(seq uint64, from common.Address) transaction.Transaction {
	coord := common.NewCoordinate(0, 0)

	MemKey := getMemKey("0ad8e4cd0c22664e977b2a634e3a9ddf7f5985c6a7fa9ec389127bf272bbafbb")

	t := &account_tx.CreateAccount{
		Base: account_tx.Base{
			Base: transaction.Base{
				ChainCoord_: coord,
				Timestamp_:  uint64(time.Now().UnixNano()),
				Type_:       transaction.Type(255),
			},
			Seq_:  seq,
			From_: from,
		},
		KeyHash: common.NewPublicHash(MemKey.PublicKey()),
	}
	return t
}

func TestAccBasicPushPop2(t *testing.T) {

	MemKey := getMemKey(hash.Hash([]byte{1}).String())
	log.Info(MemKey.PublicKey())
	log.Info(common.NewPublicHash(MemKey.PublicKey()))
	log.Info(common.NewAddress(common.NewCoordinate(1, 1), common.NewCoordinate(1, 2), uint64(1)))

	MemKey = getMemKey(hash.Hash([]byte{2}).String())
	log.Info(MemKey.PublicKey())
	log.Info(common.NewPublicHash(MemKey.PublicKey()))
	log.Info(common.NewAddress(common.NewCoordinate(1, 1), common.NewCoordinate(1, 2), uint64(1)))

	MemKey = getMemKey(hash.Hash([]byte{3}).String())
	log.Info(MemKey.PublicKey())
	log.Info(common.NewPublicHash(MemKey.PublicKey()))
	log.Info(common.NewAddress(common.NewCoordinate(1, 1), common.NewCoordinate(1, 2), uint64(1)))

	MemKey = getMemKey(hash.Hash([]byte{4}).String())
	log.Info(MemKey.PublicKey())
	log.Info(common.NewPublicHash(MemKey.PublicKey()))
	log.Info(common.NewAddress(common.NewCoordinate(1, 1), common.NewCoordinate(1, 2), uint64(1)))

	time.Sleep(time.Second)
}

func TestAccBasicPushPop(t *testing.T) {
	type args struct {
		pushCount int
		popCount  int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{name: "test1", args: args{pushCount: 10, popCount: 10}, want: 0},
		{name: "test2", args: args{pushCount: 10, popCount: 5}, want: 5},
		{name: "test3", args: args{pushCount: 10, popCount: 0}, want: 10},
		{name: "test4", args: args{pushCount: 10000, popCount: 9999}, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txPool := NewTransactionPool()

			length := txPool.Size()

			if length != 0 {
				t.Errorf("init size = %v, want 0", length)
			}

			for i := 0; i < tt.args.pushCount; i++ {
				testAddr := testAddr(i)
				tx := testAccTx(1, testAddr)
				txPool.Push(testSignature(tx))
			}

			length = txPool.Size()

			if tt.args.pushCount != length {
				t.Errorf("pushCount = %v, want %v", tt.args.pushCount, length)
				return
			}

			ctx := &testCtx{
				m: map[common.Address]uint64{},
			}

			for i := 0; i < tt.args.popCount; i++ {
				tx := txPool.Pop(ctx).Transaction
				atx := tx.(*account_tx.CreateAccount)
				ctx.updateSeq(atx, atx.Seq_)
			}

			length = txPool.Size()

			if length != tt.want {
				t.Errorf("length = %v, want %v", length, tt.want)
				return
			}

		})
	}
}

func TestAccPushPopIncreaseSeq(t *testing.T) {
	type args struct {
		pushCount uint64
		popCount  uint64
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{name: "test1", args: args{pushCount: 10, popCount: 10}, want: 0},
		{name: "test2", args: args{pushCount: 10, popCount: 5}, want: 5},
		{name: "test3", args: args{pushCount: 10, popCount: 0}, want: 10},
		{name: "test4", args: args{pushCount: 10000, popCount: 9999}, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txPool := NewTransactionPool()

			length := uint64(txPool.Size())

			if length != 0 {
				t.Errorf("init size = %v, want 0", length)
			}

			testAddr := testAddr(1)
			for i := uint64(1); i <= tt.args.pushCount; i++ {
				tx := testAccTx(i, testAddr)
				txPool.Push(testSignature(tx))
			}

			length = uint64(txPool.Size())

			if tt.args.pushCount != length {
				t.Errorf("pushCount = %v, want %v", tt.args.pushCount, length)
				return
			}

			ctx := &testCtx{
				m: map[common.Address]uint64{},
			}

			for i := uint64(0); i < tt.args.popCount; i++ {
				tx := txPool.Pop(ctx).Transaction
				atx := tx.(*account_tx.CreateAccount)
				ctx.updateSeq(atx, atx.Seq_)
			}

			if txPool.Size() != tt.want {
				t.Errorf("length = %v, want %v", txPool.Size(), tt.want)
				return
			}

		})
	}
}

func TestAccReversePushPop(t *testing.T) {
	type args struct {
		pushCount uint64
		popCount  uint64
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{name: "test1", args: args{pushCount: 10, popCount: 10}, want: 0},
		{name: "test2", args: args{pushCount: 10, popCount: 5}, want: 5},
		{name: "test3", args: args{pushCount: 10, popCount: 0}, want: 10},
		{name: "test4", args: args{pushCount: 10000, popCount: 9999}, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txPool := NewTransactionPool()

			length := uint64(txPool.Size())

			if length != 0 {
				t.Errorf("init size = %v, want 0", length)
			}

			testAddr := testAddr(1)
			for i := uint64(0); i < tt.args.pushCount; i++ {
				tx := testAccTx(tt.args.pushCount-i, testAddr)
				txPool.Push(testSignature(tx))
			}

			length = uint64(txPool.Size())

			if tt.args.pushCount != length {
				t.Errorf("pushCount = %v, want %v", tt.args.pushCount, length)
				return
			}

			ctx := &testCtx{
				m: map[common.Address]uint64{},
			}

			for i := uint64(0); i < tt.args.popCount; i++ {
				tx := txPool.Pop(ctx).Transaction
				atx := tx.(*account_tx.CreateAccount)
				ctx.updateSeq(atx, atx.Seq_)
			}

			if txPool.Size() != tt.want {
				t.Errorf("length = %v, want %v", txPool.Size(), tt.want)
				return
			}

		})
	}
}

func TestAccPushPopRemove(t *testing.T) {

	type args struct {
		pushCount   uint64
		popCount    uint64
		removeCount uint64
		order       int
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		{name: "test1", args: args{pushCount: 10, popCount: 0, removeCount: 10, order: -1}, want: 0},
		{name: "test2", args: args{pushCount: 10, popCount: 5, removeCount: 10, order: -1}, want: 0},
		{name: "test3", args: args{pushCount: 10, popCount: 0, removeCount: 5, order: 1}, want: 5},
		{name: "test4", args: args{pushCount: 10, popCount: 5, removeCount: 5, order: 1}, want: 0},
		{name: "test5", args: args{pushCount: 10, popCount: 5, removeCount: 0, order: 1}, want: 5},
		{name: "test6", args: args{pushCount: 100, popCount: 0, removeCount: 100, order: 1}, want: 0},
		{name: "test7", args: args{pushCount: 100, popCount: 0, removeCount: 1, order: -1}, want: 0},
		{name: "test8", args: args{pushCount: 100, popCount: 10, removeCount: 50, order: 1}, want: 40},
		{name: "test9", args: args{pushCount: 100, popCount: 10, removeCount: 1, order: -1}, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txPool := NewTransactionPool()
			testAddr := testAddr(1)

			txs := make([]transaction.Transaction, 0, tt.args.pushCount)
			for i := uint64(1); i <= tt.args.pushCount; i++ {
				tx := testAccTx(i, testAddr)
				txs = append(txs, tx)
				txPool.Push(testSignature(tx))
			}

			ctx := &testCtx{
				m: map[common.Address]uint64{},
			}

			for i := uint64(0); i < tt.args.popCount; i++ {
				tx := txPool.Pop(ctx).Transaction
				atx := tx.(*account_tx.CreateAccount)
				ctx.updateSeq(atx, atx.Seq_)
			}

			for i := uint64(0); i < tt.args.removeCount; i++ {
				var tx transaction.Transaction
				if tt.args.order == 1 {
					tx = txs[i+tt.args.popCount]
				} else {
					tx = txs[uint64(len(txs))-i-1]
				}
				txPool.Remove(tx)
			}

			length := uint64(txPool.Size())

			if length != tt.want {
				t.Errorf("length = %v, want %v", length, tt.want)
				return
			}

		})
	}
}

func TestUTXOBasicPushPop(t *testing.T) {
	type args struct {
		pushCount int
		popCount  int
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{name: "test1", args: args{pushCount: 10, popCount: 10}, want: 0},
		{name: "test2", args: args{pushCount: 10, popCount: 5}, want: 5},
		{name: "test3", args: args{pushCount: 10, popCount: 0}, want: 10},
		{name: "test4", args: args{pushCount: 10000, popCount: 9999}, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txPool := NewTransactionPool()

			length := txPool.Size()

			if length != 0 {
				t.Errorf("init size = %v, want 0", length)
			}

			for i := 0; i < tt.args.pushCount; i++ {
				tx := testUTXOTx(byte(i))
				txPool.Push(testSignature(tx))
			}

			length = txPool.Size()

			if tt.args.pushCount != length {
				t.Errorf("pushCount = %v, want %v", tt.args.pushCount, length)
				return
			}

			ctx := &testCtx{
				m: map[common.Address]uint64{},
			}

			for i := 0; i < tt.args.popCount; i++ {
				txPool.Pop(ctx)
			}

			length = txPool.Size()

			if length != tt.want {
				t.Errorf("length = %v, want %v", length, tt.want)
				return
			}

		})
	}
}

func TestUTXOPushPopRemove(t *testing.T) {
	type args struct {
		pushCount   uint64
		popCount    uint64
		removeCount uint64
		order       int
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		{name: "test1", args: args{pushCount: 10, popCount: 0, removeCount: 10, order: 1}, want: 0},
		{name: "test2", args: args{pushCount: 10, popCount: 5, removeCount: 10, order: -1}, want: 0},
		{name: "test3", args: args{pushCount: 10, popCount: 0, removeCount: 5, order: 1}, want: 5},
		{name: "test4", args: args{pushCount: 10, popCount: 5, removeCount: 5, order: 1}, want: 0},
		{name: "test5", args: args{pushCount: 10, popCount: 5, removeCount: 0, order: 1}, want: 5},
		{name: "test6", args: args{pushCount: 100, popCount: 0, removeCount: 100, order: 1}, want: 100},
		{name: "test7", args: args{pushCount: 100, popCount: 0, removeCount: 1, order: -1}, want: 0},
		{name: "test8", args: args{pushCount: 100, popCount: 10, removeCount: 50, order: 1}, want: 40},
		{name: "test9", args: args{pushCount: 100, popCount: 10, removeCount: 1, order: -1}, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			txPool := NewTransactionPool()

			txs := make([]transaction.Transaction, 0, tt.args.pushCount)
			for i := uint64(1); i <= tt.args.pushCount; i++ {
				tx := testUTXOTx(byte(i))
				txs = append(txs, tx)
				txPool.Push(testSignature(tx))
			}

			ctx := &testCtx{
				m: map[common.Address]uint64{},
			}

			for i := uint64(0); i < tt.args.popCount; i++ {
				txPool.Pop(ctx)
			}

			for i := uint64(0); i < tt.args.removeCount; i++ {
				var tx transaction.Transaction
				if tt.args.order == 1 {
					tx = txs[i]
				} else {
					tx = txs[uint64(len(txs))-i-1]
				}
				txPool.Remove(tx)
			}

			length := uint64(txPool.Size())

			if length != tt.want {
				t.Errorf("length = %v, want %v", length, tt.want)
				return
			}

		})
	}
}
