package consensus

import "github.com/fletaio/common"

var (
	tagStaking = []byte{1, 0}
)

func toStakingKey(addr common.Address) []byte {
	bs := make([]byte, 2+common.AddressSize)
	copy(bs, tagStaking)
	copy(bs[2:], addr[:])
	return bs
}

func fromStakingKey(bs []byte) common.Address {
	var addr common.Address
	copy(addr[:], bs[2:])
	return addr
}
