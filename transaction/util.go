package transaction

import "git.fleta.io/fleta/common"

// IsMainChain TODO
func IsMainChain(ChainCoord *common.Coordinate) bool {
	return ChainCoord.Height == 0 && ChainCoord.Index == 0
}
