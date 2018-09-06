package amount

import (
	"bytes"
	"io"
	"strconv"

	"git.fleta.io/fleta/common/util"
)

// COIN TODO
const COIN = Amount(100000000)

// Amount TODO
type Amount uint64

// WriteTo TODO
func (amount *Amount) WriteTo(w io.Writer) (int64, error) {
	return util.WriteUint64(w, uint64(*amount))
}

// ReadFrom TODO
func (amount *Amount) ReadFrom(r io.Reader) (int64, error) {
	if v, n, err := util.ReadUint64(r); err != nil {
		return n, err
	} else {
		(*amount) = Amount(v)
		return n, nil
	}
}

// MarshalJSON TODO
func (amount Amount) MarshalJSON() ([]byte, error) {
	v := uint64(amount)
	var buffer bytes.Buffer
	buffer.WriteString(strconv.FormatUint(v/uint64(COIN), 10))
	buffer.WriteString(".")
	p := strconv.FormatUint(v%uint64(COIN), 10)
	for i := 0; i < 8-len(p); i++ {
		buffer.WriteString("0")
	}
	buffer.WriteString(p)
	return buffer.Bytes(), nil
}

// Debug TODO
func (amount Amount) Debug() (string, error) {
	if bs, err := amount.MarshalJSON(); err != nil {
		return "", err
	} else {
		return string(bs), err
	}
}
