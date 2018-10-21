package store

import (
	"encoding/binary"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
)

var (
	tagHeightBlock          = []byte{1, 0}
	tagHeightBlockHash      = []byte{1, 1}
	tagHashBlockHeight      = []byte{1, 2}
	tagHeightObserverSigned = []byte{2, 0}
	tagAccount              = []byte{3, 0}
	tagAccountSeq           = []byte{3, 1}
	tagAccountData          = []byte{3, 2}
	tagUTXO                 = []byte{4, 0}
	tagCustomData           = []byte{5, 0}
)

func toHeightBlockKey(height uint32) []byte {
	bs := make([]byte, 6)
	copy(bs, tagHeightBlock)
	binary.LittleEndian.PutUint32(bs[2:], height)
	return bs
}

func toHeightObserverSignedKey(height uint32) []byte {
	bs := make([]byte, 6)
	copy(bs, tagHeightObserverSigned)
	binary.LittleEndian.PutUint32(bs[2:], height)
	return bs
}

func toHeightBlockHashKey(height uint32) []byte {
	bs := make([]byte, 6)
	copy(bs, tagHeightBlockHash)
	binary.LittleEndian.PutUint32(bs[2:], height)
	return bs
}

func toHashBlockHeightKey(h hash.Hash256) []byte {
	bs := make([]byte, 34)
	copy(bs, tagHashBlockHeight)
	copy(bs[2:], h[:])
	return bs
}

func toAccountKey(addr common.Address) []byte {
	bs := make([]byte, 2+common.AddressSize)
	copy(bs, tagAccount)
	copy(bs[2:], addr[:])
	return bs
}

func toAccountSeqKey(addr common.Address) []byte {
	bs := make([]byte, 2+common.AddressSize)
	copy(bs, tagAccountSeq)
	copy(bs[2:], addr[:])
	return bs
}

func toAccountDataKey(key string) []byte {
	bs := make([]byte, 2+len(key))
	copy(bs, tagAccountData)
	copy(bs[2:], []byte(key))
	return bs
}

func toUTXOKey(id uint64) []byte {
	bs := make([]byte, 10)
	copy(bs, tagUTXO)
	binary.LittleEndian.PutUint64(bs[2:], id)
	return bs
}

func fromUTXOKey(bs []byte) uint64 {
	return binary.LittleEndian.Uint64(bs[2:])
}

func toCustomData(key string) []byte {
	bs := make([]byte, 1+len(key))
	copy(bs, tagCustomData)
	copy(bs[2:], []byte(key))
	return bs
}
