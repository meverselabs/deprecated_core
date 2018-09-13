package chain

import (
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/chain/account/data"
)

// account address types
const (
	KeyAccountType         = common.AddressType(10)
	LockedAccountType      = common.AddressType(18)
	MultiSigAccountType    = common.AddressType(20)
	FormulationAccountType = common.AddressType(30)
)

// LockedAccountData TODO
type LockedAccountData struct {
	*data.Base
	UnlockHeight uint32
}

// WriteTo TODO
func (d *LockedAccountData) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := d.Base.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint32(w, d.UnlockHeight); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom TODO
func (d *LockedAccountData) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := d.Base.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if v, n, err := util.ReadUint32(r); err != nil {
		return read, err
	} else {
		read += n
		d.UnlockHeight = v
	}
	return read, nil
}
