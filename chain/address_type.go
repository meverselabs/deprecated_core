package chain

import (
	"git.fleta.io/fleta/common"
)

// account address types
const (
	SingleAddressType      = common.AddressType(10)
	LockedAddressType      = common.AddressType(19)
	MultiSigAddressType    = common.AddressType(20)
	FormulationAddressType = common.AddressType(30)
)

// NameOfAddressType TODO
func NameOfAddressType(t common.AddressType) string {
	switch t {
	case SingleAddressType:
		return "SingleAddressType"
	case LockedAddressType:
		return "LockedAddressType"
	case MultiSigAddressType:
		return "MultiSigAddressType"
	case FormulationAddressType:
		return "FormulationAddressType"
	default:
		return "UnknownAddressType"
	}
}
