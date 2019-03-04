package block

import (
	"bytes"
	"io"

	"github.com/fletaio/common"
	"github.com/fletaio/common/util"
	"github.com/fletaio/core/data"
	"github.com/fletaio/core/transaction"
)

// Body is the set of transactions with validation informations
type Body struct {
	Transactions          []transaction.Transaction //MAXLEN : 65535
	TransactionSignatures [][]common.Signature      //MAXLEN : 65536
	Tran                  *data.Transactor
}

// WriteTo is a serialization function
func (bb *Body) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := util.WriteUint16(w, uint16(len(bb.Transactions))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, tx := range bb.Transactions {
			if n, err := util.WriteUint8(w, uint8(tx.Type())); err != nil {
				return wrote, err
			} else {
				wrote += n
				if n, err := tx.WriteTo(w); err != nil {
					return wrote, err
				} else {
					wrote += n
				}
			}
		}
		for _, sigs := range bb.TransactionSignatures {
			wrote += n
			if n, err := util.WriteUint8(w, uint8(len(sigs))); err != nil {
				return wrote, err
			} else {
				wrote += n
				for _, sig := range sigs {
					if n, err := sig.WriteTo(w); err != nil {
						return wrote, err
					} else {
						wrote += n
					}
				}
			}
		}
	}
	return wrote, nil
}

// ReadFrom is a deserialization function
func (bb *Body) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if Len, n, err := util.ReadUint16(r); err != nil {
		return read, err
	} else {
		read += n
		bb.Transactions = make([]transaction.Transaction, 0, Len)
		for i := 0; i < int(Len); i++ {
			if t, n, err := util.ReadUint8(r); err != nil {
				return read, err
			} else {
				read += n
				if tx, err := bb.Tran.NewByType(transaction.Type(t)); err != nil {
					return read, err
				} else {
					if n, err := tx.ReadFrom(r); err != nil {
						return read, err
					} else {
						read += n
						bb.Transactions = append(bb.Transactions, tx)
					}
				}
			}
		}
		bb.TransactionSignatures = make([][]common.Signature, 0, Len)
		for i := 0; i < int(Len); i++ {
			if SLen, n, err := util.ReadUint8(r); err != nil {
				return read, err
			} else {
				read += n
				sigs := []common.Signature{}
				for j := 0; j < int(SLen); j++ {
					var sig common.Signature
					if n, err := sig.ReadFrom(r); err != nil {
						return read, err
					} else {
						read += n
						sigs = append(sigs, sig)
					}
				}
				bb.TransactionSignatures = append(bb.TransactionSignatures, sigs)
			}
		}
	}
	return read, nil
}

// MarshalJSON is a marshaler function
func (bb *Body) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString(`{`)
	buffer.WriteString(`"transactions":`)
	buffer.WriteString(`[`)
	for i, tx := range bb.Transactions {
		if i > 0 {
			buffer.WriteString(`,`)
		}
		if bs, err := tx.MarshalJSON(); err != nil {
			return nil, err
		} else {
			buffer.Write(bs)
		}
	}
	buffer.WriteString(`]`)
	buffer.WriteString(`,`)
	buffer.WriteString(`"signatures":`)
	buffer.WriteString(`[`)
	for i, sigs := range bb.TransactionSignatures {
		if i > 0 {
			buffer.WriteString(`,`)
		}
		buffer.WriteString(`[`)
		for j, sig := range sigs {
			if j > 0 {
				buffer.WriteString(`,`)
			}
			if bs, err := sig.MarshalJSON(); err != nil {
				return nil, err
			} else {
				buffer.Write(bs)
			}
		}
		buffer.WriteString(`]`)
	}
	buffer.WriteString(`]`)
	buffer.WriteString(`}`)
	return buffer.Bytes(), nil
}
