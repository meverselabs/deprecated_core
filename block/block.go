package block

import (
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/transaction"
)

// Block TODO
type Block struct {
	Header                Header
	Transactions          []transaction.Transaction //MAXLEN : 65535
	TransactionSignatures [][]common.Signature      //MAXLEN : 65536
}

// WriteTo TODO
func (b *Block) WriteTo(w io.Writer) (int64, error) {
	if len(b.Transactions) > 65535 {
		return 0, ErrExceedTransactionCount
	}
	if len(b.Transactions) != len(b.TransactionSignatures) {
		return 0, ErrMismatchSignaturesCount
	}

	var wrote int64
	if n, err := b.Header.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}

	if n, err := util.WriteUint16(w, uint16(len(b.Transactions))); err != nil {
		return wrote, err
	} else {
		wrote += n
		for _, tx := range b.Transactions {
			if t, err := TypeOfTransaction(tx); err != nil {
				return wrote, err
			} else {
				if n, err := util.WriteUint8(w, uint8(t)); err != nil {
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
		}
		for _, sigs := range b.TransactionSignatures {
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

// ReadFrom TODO
func (b *Block) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := b.Header.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}

	if Len, n, err := util.ReadUint16(r); err != nil {
		return read, err
	} else {
		read += n
		b.Transactions = make([]transaction.Transaction, 0, Len)
		for i := 0; i < int(Len); i++ {
			if t, n, err := util.ReadUint8(r); err != nil {
				return read, err
			} else {
				read += n
				if tx, err := NewTransactionByType(TransactionType(t)); err != nil {
					return read, err
				} else {
					if n, err := tx.ReadFrom(r); err != nil {
						return read, err
					} else {
						read += n
						b.Transactions = append(b.Transactions, tx)
					}
				}
			}
		}
		b.TransactionSignatures = make([][]common.Signature, 0, Len)
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
				b.TransactionSignatures = append(b.TransactionSignatures, sigs)
			}
		}
	}
	return read, nil
}