package data

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/fletaio/common"
	"github.com/fletaio/common/util"
	"github.com/fletaio/core/amount"
)

// LockedBalance defines locked balance
type LockedBalance struct {
	Address      common.Address
	Amount       *amount.Amount
	UnlockHeight uint32
}

// WriteTo is a serialization function
func (lb *LockedBalance) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := lb.Address.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := lb.Amount.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint32(w, lb.UnlockHeight); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom is a deserialization function
func (lb *LockedBalance) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := lb.Address.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := lb.Amount.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if v, n, err := util.ReadUint32(r); err != nil {
		return read, err
	} else {
		read += n
		lb.UnlockHeight = v
	}
	return read, nil
}

// MarshalJSON is a marshaler function
func (lb *LockedBalance) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString(`{`)
	buffer.WriteString(`"address":`)
	if bs, err := lb.Address.MarshalJSON(); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`,`)
	buffer.WriteString(`"amount":`)
	if bs, err := lb.Amount.MarshalJSON(); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`,`)
	buffer.WriteString(`"unlock_height":`)
	if bs, err := json.Marshal(lb.UnlockHeight); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`}`)
	return buffer.Bytes(), nil
}
