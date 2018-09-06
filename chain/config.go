package chain

import (
	"time"

	"git.fleta.io/fleta/common/hash"
)

var (
	// ObserverPubkeyHash TODO
	ObserverPubkeyHash = map[string]bool{}
	// GenesisHash SHA256("fleta : fatest blockchain system")
	GenesisHash = hash.MustParseHex("666c657461203a2066617465737420626c6f636b636861696e2073797374656d")
)

const (
	// ObserverCount TODO
	ObserverCount = 5
	// ObserverSignatureRequired TODO
	ObserverSignatureRequired = (ObserverCount)/2 + 1
	// UTXOCacheWriteOutTime TODO
	UTXOCacheWriteOutTime = time.Second * 30
)
