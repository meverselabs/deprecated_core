package consensus

import (
	"bytes"

	"github.com/fletaio/common"
)

var (
	TagStaking     = []byte{1, 0}
	tagAutoStaking = []byte{1, 1}
)

func toStakingKey(addr common.Address) []byte {
	bs := make([]byte, 2+common.AddressSize)
	copy(bs, TagStaking)
	copy(bs[2:], addr[:])
	return bs
}

// FromStakingKey returns staking address if it is staking key
func FromStakingKey(bs []byte) (common.Address, bool) {
	if bytes.HasPrefix(bs, TagStaking) {
		var addr common.Address
		copy(addr[:], bs[2:])
		return addr, true
	} else {
		return common.Address{}, false
	}
}

func toAutoStakingKey(addr common.Address) []byte {
	bs := make([]byte, 2+common.AddressSize)
	copy(bs, tagAutoStaking)
	copy(bs[2:], addr[:])
	return bs
}
