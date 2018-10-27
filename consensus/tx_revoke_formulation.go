package consensus

import (
	"bytes"
	"io"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/amount"
	"git.fleta.io/fleta/core/data"
	"git.fleta.io/fleta/core/transaction"
)

func init() {
	data.RegisterTransaction("formulation.RevokeFormulation", func(t transaction.Type) transaction.Transaction {
		return &RevokeFormulation{
			Base: transaction.Base{
				ChainCoord_: &common.Coordinate{},
				Type_:       t,
			},
		}
	}, func(loader data.Loader, t transaction.Transaction, signers []common.PublicHash) error {
		tx := t.(*RevokeFormulation)
		if tx.Seq() <= loader.Seq(tx.From()) {
			return ErrInvalidSequence
		}
		if tx.To.Equal(tx.From()) {
			return ErrInvalidToAddress
		}

		fromAcc, err := loader.Account(tx.From())
		if err != nil {
			return err
		}

		if err := loader.Accounter().Validate(loader, fromAcc, signers); err != nil {
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

		fromAcc, err := ctx.Account(tx.From())
		if err != nil {
			return nil, err
		}

		chainCoord := ctx.ChainCoord()
		fromBalance := fromAcc.Balance(chainCoord)
		if fromBalance.Less(Fee) {
			return nil, ErrInsuffcientBalance
		}
		fromBalance = fromBalance.Sub(Fee)
		fromAcc.SetBalance(chainCoord, fromBalance)

		toAcc, err := ctx.Account(tx.To)
		if err != nil {
			return nil, err
		}
		for _, TokenCoord := range fromAcc.TokenCoords() {
			fromBalance := fromAcc.Balance(TokenCoord)
			fromAcc.SetBalance(TokenCoord, amount.NewCoinAmount(0, 0))
			toBalance := toAcc.Balance(TokenCoord)
			toBalance = toBalance.Add(fromBalance)
			toAcc.SetBalance(TokenCoord, toBalance)
		}
		ctx.DeleteAccount(fromAcc)

		ctx.Commit(sn)
		return nil, nil
	})
}

// RevokeFormulation is formulation.RevokeFormulation
type RevokeFormulation struct {
	transaction.Base
	Seq_  uint64
	From_ common.Address
	To    common.Address
}

// IsUTXO retuns false
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

// Hash retuns the hash value of it
func (tx *RevokeFormulation) Hash() hash.Hash256 {
	var buffer bytes.Buffer
	if _, err := tx.WriteTo(&buffer); err != nil {
		panic(err)
	}
	return hash.DoubleHash(buffer.Bytes())
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
	if n, err := tx.To.WriteTo(w); err != nil {
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
	if n, err := tx.To.ReadFrom(r); err != nil {
		return read, err
	} else {
		read += n
	}
	return read, nil
}