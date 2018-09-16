package chain

import (
	"git.fleta.io/fleta/common"
)

// account address types
const (
	SingleAccountType      = common.AddressType(10)
	LockedAccountType      = common.AddressType(18)
	MultiSigAccountType    = common.AddressType(20)
	FormulationAccountType = common.AddressType(30)
)

// NameOfAddressType TODO
func NameOfAddressType(t common.AddressType) string {
	switch t {
	case SingleAccountType:
		return "SingleAccountType"
	case LockedAccountType:
		return "LockedAccountType"
	case MultiSigAccountType:
		return "MultiSigAccountType"
	case FormulationAccountType:
		return "FormulationAccountType"
	default:
		return "UnknownAccountType"
	}
}
