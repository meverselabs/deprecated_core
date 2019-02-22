package block

import (
	"encoding/json"
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/data"
	"git.fleta.io/fleta/core/transaction"
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

// UnmarshalJSON is a unmarshaler function
func (bb *Body) UnmarshalJSON(bs []byte) error {
	return json.Unmarshal(bs, &bb)
}

// MarshalJSON is a marshaler function
func (bb *Body) MarshalJSON() ([]byte, error) {
	return json.Marshal(bb)
}
