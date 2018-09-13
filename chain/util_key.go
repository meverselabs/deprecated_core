package chain

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
)

const (
	tagHeightBlock         = iota
	toHeightObserverSigned = iota
	tagHeightBlockHash     = iota
	tagHashBlockHeight     = iota
)

func toHeightBlockKey(height uint32) []byte {
	return append(util.Uint32ToBytes(height), tagHeightBlock)
}

func toHeightObserverSignedKey(height uint32) []byte {
	return append(util.Uint32ToBytes(height), toHeightObserverSigned)
}

func toHeightBlockHashKey(height uint32) []byte {
	return append(util.Uint32ToBytes(height), tagHeightBlockHash)
}

func toHashBlockHeightKey(h hash.Hash256) []byte {
	return append(h[:], tagHashBlockHeight)
}

func toAccountDataKey(addr common.Address, name string) []byte {
	key := make([]byte, len(addr)+len(name))
	copy(key, addr[:])
	copy(key[len(addr):], []byte(name))
	return key
}
