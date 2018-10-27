package amount

import (
	"io"
	"math"
	"math/big"
	"strconv"
	"strings"

	"git.fleta.io/fleta/common/util"
)

// COIN is 1 coin
var COIN = NewCoinAmount(1, 0)

// FractionalMax represent the max value of under the float point
const FractionalMax = 1000000000000000000

// FractionalCount represent the number of under the float point
const FractionalCount = 18

func init() {
	if math.Pow10(FractionalCount) != FractionalMax {
		panic("Pow10(FractionalCount) is different with FractionalMax")
	}
}

var zeroInt = big.NewInt(0)

// Amount is the precision float value based on the big.Int
type Amount struct {
	*big.Int
}

func newAmount(value int64) *Amount {
	return &Amount{
		Int: big.NewInt(value),
	}
}

// NewCoinAmount returns the amount that is consisted of the integer and the fractional value
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

// NewAmountFromBytes parse the amount from the byte array
func NewAmountFromBytes(bs []byte) *Amount {
	b := newAmount(0)
	b.Int.SetBytes(bs)
	return b
}

// WriteTo is a serialization function
func (am *Amount) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := util.WriteBytes(w, am.Int.Bytes()); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom is a deserialization function
func (am *Amount) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if bs, n, err := util.ReadBytes(r); err != nil {
		return read, err
	} else {
		read += n
		am.Int.SetBytes(bs)
	}
	return read, nil
}

// Clone returns the clonend value of it
func (am *Amount) Clone() *Amount {
	c := newAmount(0)
	c.Int.Add(am.Int, zeroInt)
	return c
}

// Add returns a + b (*immutable)
func (am *Amount) Add(b *Amount) *Amount {
	c := newAmount(0)
	c.Int.Add(am.Int, b.Int)
	return c
}

// Sub returns a - b (*immutable)
func (am *Amount) Sub(b *Amount) *Amount {
	c := newAmount(0)
	c.Int.Sub(am.Int, b.Int)
	return c
}

// DivC returns a / b (*immutable)
func (am *Amount) DivC(b int64) *Amount {
	c := newAmount(0)
	c.Int.Div(am.Int, big.NewInt(b))
	return c
}

// MulC returns a * b (*immutable)
func (am *Amount) MulC(b int64) *Amount {
	c := newAmount(0)
	c.Int.Mul(am.Int, big.NewInt(b))
	return c
}

// IsZero returns a == 0
func (am *Amount) IsZero() bool {
	return am.Int.Cmp(zeroInt) == 0
}

// Less returns a < b
func (am *Amount) Less(b *Amount) bool {
	return am.Int.Cmp(b.Int) < 0
}

// Equal checks compare two values and returns true or false
func (am *Amount) Equal(b *Amount) bool {
	return am.Int.Cmp(b.Int) == 0
}

// String returns the float string of the amount
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

// ParseAmount parse the amount from the float string
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
