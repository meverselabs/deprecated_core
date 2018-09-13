package chain

import (
	"git.fleta.io/fleta/common"
)

// account address types
const (
	KeyAccountType         = common.AddressType(10)
	LockedAccountType      = common.AddressType(18)
	MultiSigAccountType    = common.AddressType(20)
	FormulationAccountType = common.AddressType(30)
)
