package transaction

import (
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/amount"
)

// TxOut TODO
type TxOut struct {
	Amount    amount.Amount
	Addresses []common.Address
}

// WriteTo TODO
func (out *TxOut) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := out.Amount.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint8(w, uint8(len(out.Addresses))); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	for _, addr := range out.Addresses {
		if n, err := addr.WriteTo(w); err != nil {
			return wrote, err
		} else {
			wrote += n
		}
	}
	return wrote, nil
}

// ReadFrom TODO
func (out *TxOut) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := out.Amount.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}

	if Len, n, err := util.ReadUint8(r); err != nil {
		return read, err
	} else {
		read += n
		for i := 0; i < int(Len); i++ {
			var addr common.Address
			if n, err := addr.ReadFrom(r); err != nil {
				return read, err
			} else {
				read += n
				out.Addresses = append(out.Addresses, addr)
			}
		}
	}
	return read, nil
}
