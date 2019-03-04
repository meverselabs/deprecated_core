package transaction

import "github.com/fletaio/common"

// IsMainChain returns that the target chain is the main chain or not
func IsMainChain(ChainCoord *common.Coordinate) bool {
	return ChainCoord.Height == 0 && ChainCoord.Index == 0
}
