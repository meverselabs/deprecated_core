package event

import (
	"github.com/fletaio/common"
)

// UnmarshalID returns the block height, the transaction index in the block, the output index in the transaction
func UnmarshalID(id uint64) (*common.Coordinate, uint16) {
	return common.NewCoordinate(uint32(id>>32), uint16(id>>16)), uint16(id)
}

// MarshalID returns the packed id
func MarshalID(coord *common.Coordinate, index uint16) uint64 {
	return uint64(coord.Height)<<32 | uint64(coord.Index)<<16 | uint64(index)
}
