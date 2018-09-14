package chain

import (
	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
)

const (
	tagHeightBlock           = 1
	toHeightObserverSigned   = 2
	tagHeightBlockHash       = 3
	tagHashBlockHeight       = 4
	tagGenesisAccountAddress = 5
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

func toGenesisAccountAddress(idx uint16) []byte {
	return append(util.Uint16ToBytes(idx), tagGenesisAccountAddress)
}

func toAccountDataKey(addr common.Address, name string) []byte {
	key := make([]byte, len(addr)+len(name))
	copy(key, addr[:])
	copy(key[len(addr):], []byte(name))
	return key
}
