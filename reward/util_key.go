package reward

import (
	"bytes"

	"github.com/fletaio/common"
)

var (
	tagPowerSum = []byte{1, 0}
)

func toPowerSumKey(addr common.Address) []byte {
	bs := make([]byte, 2+common.AddressSize)
	copy(bs, tagPowerSum)
	copy(bs[2:], addr[:])
	return bs
}

func getPowerSumKey(bs []byte) (common.Address, bool) {
	if bytes.HasPrefix(bs, tagPowerSum) {
		var addr common.Address
		copy(addr[:], bs[2:])
		return addr, true
	} else {
		return common.Address{}, false
	}
}
