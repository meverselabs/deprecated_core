package store

import (
	"encoding/binary"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
)

const (
	tagHeightBlock          = byte(0)
	tagHeightBlockHash      = byte(1)
	tagHashBlockHeight      = byte(2)
	tagHeightObserverSigned = byte(3)
	tagAccount              = byte(4)
	tagAccountData          = byte(5)
	tagTxInUTXO             = byte(6)
	tagCustomData           = byte(7)
)

func toHeightBlockKey(height uint32) []byte {
	bs := make([]byte, 5)
	bs[0] = tagHeightBlock
	binary.LittleEndian.PutUint32(bs[1:], height)
	return bs
}

func toHeightObserverSignedKey(height uint32) []byte {
	bs := make([]byte, 5)
	bs[0] = tagHeightObserverSigned
	binary.LittleEndian.PutUint32(bs[1:], height)
	return bs
}

func toHeightBlockHashKey(height uint32) []byte {
	bs := make([]byte, 5)
	bs[0] = tagHeightBlockHash
	binary.LittleEndian.PutUint32(bs[1:], height)
	return bs
}

func toHashBlockHeightKey(h hash.Hash256) []byte {
	bs := make([]byte, 33)
	bs[0] = tagHashBlockHeight
	copy(bs[1:], h[:])
	return bs
}

func toAccountKey(addr common.Address) []byte {
	bs := make([]byte, 1+common.AddressSize)
	bs[0] = tagAccount
	copy(bs[1:], addr[:])
	return bs
}

func toAccountDataKey(key string) []byte {
	bs := make([]byte, 1+len(key))
	bs[0] = tagAccountData
	copy(bs[1:], []byte(key))
	return bs
}

func toTxInUTXOKey(id uint64) []byte {
	bs := make([]byte, 9)
	bs[0] = tagTxInUTXO
	binary.LittleEndian.PutUint64(bs[1:], id)
	return bs
}

func toCustomData(key string) []byte {
	bs := make([]byte, 1+len(key))
	bs[0] = tagCustomData
	copy(bs[1:], []byte(key))
	return bs
}
