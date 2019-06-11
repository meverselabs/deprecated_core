package consensus

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/fletaio/common"
	"github.com/fletaio/common/hash"
	"github.com/fletaio/common/util"
	"github.com/fletaio/core/amount"
	"github.com/fletaio/core/data"
	"github.com/fletaio/core/transaction"
)

func init() {
	data.RegisterTransaction("consensus.RevokeFormulation", func(t transaction.Type) transaction.Transaction {
		return &RevokeFormulation{
			Base: transaction.Base{
				Type_: t,
			},
		}
	}, func(loader data.Loader, t transaction.Transaction, signers []common.PublicHash) error {
		tx := t.(*RevokeFormulation)
		if tx.Seq() <= loader.Seq(tx.From()) {
			return ErrInvalidSequence
		}

		acc, err := loader.Account(tx.From())
		if err != nil {
			return err
		}
		frAcc, is := acc.(*FormulationAccount)
		if !is {
			return ErrInvalidAccountType
		}
		if err := loader.Accounter().Validate(loader, frAcc, signers); err != nil {
			return err
		}
		return nil
	}, func(ctx *data.Context, Fee *amount.Amount, t transaction.Transaction, coord *common.Coordinate) (ret interface{}, rerr error) {
		tx := t.(*RevokeFormulation)
		sn := ctx.Snapshot()
		defer ctx.Revert(sn)

		if tx.Seq() != ctx.Seq(tx.From())+1 {
			return nil, ErrInvalidSequence
		}
		ctx.AddSeq(tx.From())

		heritorAcc, err := ctx.Account(tx.From())
		if err != nil {
			return nil, err
		}

		acc, err := ctx.Account(tx.From())
		if err != nil {
			return nil, err
		}
		frAcc, is := acc.(*FormulationAccount)
		if !is {
			return nil, ErrInvalidAccountType
		}
		switch frAcc.FormulationType {
		case AlphaFormulatorType:
			fallthrough
		case SigmaFormulatorType:
			fallthrough
		case OmegaFormulatorType:
			if err := frAcc.SubBalance(Fee); err != nil {
				return nil, err
			}
			heritorAcc.AddBalance(frAcc.Amount)
			heritorAcc.AddBalance(frAcc.Balance())
		case HyperFormulatorType:
			if err := frAcc.SubBalance(Fee); err != nil {
				return nil, err
			}
			heritorAcc.AddBalance(frAcc.Amount)
			heritorAcc.AddBalance(frAcc.Balance())

			keys, err := ctx.AccountDataKeys(tx.From())
			if err != nil {
				return nil, err
			}
			for _, k := range keys {
				bs := ctx.AccountData(tx.From(), k)
				if len(bs) == 0 {
					return nil, ErrInvalidStakingAddress
				}
				StakingAmount := amount.NewAmountFromBytes(bs)
				if frAcc.StakingAmount.Less(StakingAmount) {
					return nil, ErrCriticalStakingAmount
				}
				frAcc.StakingAmount.Sub(StakingAmount)

				if StakingAccount, err := ctx.Account(FromKeyToAddress(bs)); err != nil {
					if err != data.ErrNotExistAccount {
						return nil, err
					}
				} else {
					StakingAccount.AddBalance(StakingAmount)
				}
			}
			if !frAcc.StakingAmount.IsZero() {
				return nil, ErrCriticalStakingAmount
			}
		default:

		}
		ctx.DeleteAccount(acc)

		ctx.Commit(sn)
		return nil, nil
	})
}

// RevokeFormulation is a consensus.RevokeFormulation
// It is used to remove formulation account and get back staked coin
type RevokeFormulation struct {
	transaction.Base
	Seq_    uint64
	From_   common.Address
	Heritor common.Address
}

// IsUTXO returns false
func (tx *RevokeFormulation) IsUTXO() bool {
	return false
}

// From returns the creator of the transaction
func (tx *RevokeFormulation) From() common.Address {
	return tx.From_
}

// Seq returns the sequence of the transaction
func (tx *RevokeFormulation) Seq() uint64 {
	return tx.Seq_
}

// Hash returns the hash value of it
func (tx *RevokeFormulation) Hash() hash.Hash256 {
	return hash.DoubleHashByWriterTo(tx)
}

// WriteTo is a serialization function
func (tx *RevokeFormulation) WriteTo(w io.Writer) (int64, error) {
	var wrote int64
	if n, err := tx.Base.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := util.WriteUint64(w, tx.Seq_); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := tx.From_.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	if n, err := tx.Heritor.WriteTo(w); err != nil {
		return wrote, err
	} else {
		wrote += n
	}
	return wrote, nil
}

// ReadFrom is a deserialization function
func (tx *RevokeFormulation) ReadFrom(r io.Reader) (int64, error) {
	var read int64
	if n, err := tx.Base.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if v, n, err := util.ReadUint64(r); err != nil {
		return read, err
	} else {
		read += n
		tx.Seq_ = v
	}
	if n, err := tx.From_.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	if n, err := tx.Heritor.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	return read, nil
}

// MarshalJSON is a marshaler function
func (tx *RevokeFormulation) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString(`{`)
	buffer.WriteString(`"type":`)
	if bs, err := json.Marshal(tx.Type_); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`,`)
	buffer.WriteString(`"timestamp":`)
	if bs, err := json.Marshal(tx.Timestamp_); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`,`)
	buffer.WriteString(`"seq":`)
	if bs, err := json.Marshal(tx.Seq_); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`,`)
	buffer.WriteString(`"from":`)
	if bs, err := tx.From_.MarshalJSON(); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`,`)
	buffer.WriteString(`"heritor":`)
	if bs, err := tx.Heritor.MarshalJSON(); err != nil {
		return nil, err
	} else {
		buffer.Write(bs)
	}
	buffer.WriteString(`}`)
	return buffer.Bytes(), nil
}
