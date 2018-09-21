package amount

import (
	"io"
	"math/big"
	"strconv"
	"strings"

	"git.fleta.io/fleta/common/util"
)

// COIN TODO
var COIN = NewCoinAmount(1, 0)

// FractionalMax TODO
const FractionalMax = 1000000000000000000

// FractionalCount TODO
const FractionalCount = 18

var zeroInt = big.NewInt(0)

// Amount TODO
type Amount struct {
	*big.Int
}

// newAmount TODO
func newAmount(value int64) *Amount {
	return &Amount{
		Int: big.NewInt(value),
	}
}

// NewCoinAmount TODO
func NewCoinAmount(i uint64, f uint64) *Amount {
	if i == 0 {
		return newAmount(int64(f))
	} else if f == 0 {
		bi := newAmount(int64(i))
		return bi.MulC(FractionalMax)
	} else {
		bi := newAmount(int64(i))
		bf := newAmount(int64(f))
		return bi.MulC(FractionalMax).Add(bf)
	}
}

// WriteTo TODO
func (am *Amount) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := util.WriteBytes8(w, am.Int.Bytes()); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom TODO
func (am *Amount) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if bs, n, err := util.ReadBytes8(r); err != nil {
		return read, err
	} else {
		read += n
		am.Int.SetBytes(bs)
	}
	return read, nil
}

// Clone TODO
func (am *Amount) Clone() *Amount {
	c := newAmount(0)
	c.Int.Add(am.Int, zeroInt)
	return c
}

// Add TODO
func (am *Amount) Add(b *Amount) *Amount {
	c := newAmount(0)
	c.Int.Add(am.Int, b.Int)
	return c
}

// Sub TODO
func (am *Amount) Sub(b *Amount) *Amount {
	c := newAmount(0)
	c.Int.Sub(am.Int, b.Int)
	return c
}

// DivC TODO
func (am *Amount) DivC(b int64) *Amount {
	c := newAmount(0)
	c.Int.Div(am.Int, big.NewInt(b))
	return c
}

// MulC TODO
func (am *Amount) MulC(b int64) *Amount {
	c := newAmount(0)
	c.Int.Mul(am.Int, big.NewInt(b))
	return c
}

// IsZero TODO
func (am *Amount) IsZero() bool {
	return am.Int.Cmp(zeroInt) == 0
}

// Less TODO
func (am *Amount) Less(b *Amount) bool {
	return am.Int.Cmp(b.Int) < 0
}

// Equal TODO
func (am *Amount) Equal(b *Amount) bool {
	return am.Int.Cmp(b.Int) == 0
}

// String TODO
func (am *Amount) String() string {
	if am.IsZero() {
		return "0"
	}
	str := am.Int.String()
	if len(str) < FractionalCount {
		return "0." + formatFractional(str)
	} else {
		si := str[:len(str)-FractionalCount]
		sf := strings.TrimRight(str[len(str)-FractionalCount:], "0")
		if len(sf) > 0 {
			return si + "." + sf
		} else {
			return si
		}
	}
}

// ParseAmount TODO
func ParseAmount(str string) (*Amount, error) {
	ls := strings.SplitN(str, ".", 2)
	switch len(ls) {
	case 1:
		pi, err := strconv.ParseUint(ls[0], 10, 64)
		if err != nil {
			return nil, ErrInvalidFormat
		}
		return NewCoinAmount(pi, 0), nil
	case 2:
		pi, err := strconv.ParseUint(ls[0], 10, 64)
		if err != nil {
			return nil, ErrInvalidFormat
		}
		pf, err := strconv.ParseUint(padFractional(ls[1]), 10, 64)
		if err != nil {
			return nil, ErrInvalidFormat
		}
		return NewCoinAmount(pi, pf), nil
	default:
		return nil, ErrInvalidFormat
	}
}
