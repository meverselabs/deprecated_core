package chain

import (
	"bytes"

	"git.fleta.io/fleta/core/amount"
	"git.fleta.io/fleta/core/chain/account"
	"git.fleta.io/fleta/core/transaction/advanced"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/store"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/block"
	"git.fleta.io/fleta/core/level"
	"git.fleta.io/fleta/core/transaction"
)

// Provider TODO
type Provider interface {
	Config() Config
	Coordinate() common.Coordinate
	Genesis() hash.Hash256
	Height() uint32
	Account(addr common.Address) (*account.Account, error)
	Fee(tx transaction.Transaction) *amount.Amount
	BlockReward(height uint32) *amount.Amount
	HashCurrentBlock() (hash.Hash256, error)
	Block(height uint32) (*block.Block, error)
	BlockByHash(h hash.Hash256) (*block.Block, error)
	BlockHash(height uint32) (hash.Hash256, error)
	ObserverSigned(height uint32) (*block.ObserverSigned, error)
	ObserverSignedByHash(h hash.Hash256) (*block.ObserverSigned, error)
	Transactions(height uint32) ([]transaction.Transaction, error)
	Transaction(height uint32, index uint16) (transaction.Transaction, error)
}

// Chain TODO
type Chain interface {
	Provider
	Close()
	UpdateAccount(acc *account.Account) error
	ConnectBlock(b *block.Block, s *block.ObserverSigned, ExpectedPublicKey common.PublicKey) ([]hash.Hash256, error)
}

// Base TODO
type Base struct {
	genesisHash   hash.Hash256
	coordinate    common.Coordinate
	blockStore    store.Store
	accountStore  store.Store
	height        uint32
	hashPrevBlock hash.Hash256
	config        *Config
}

// NewBase TODO
func NewBase(config *Config, GenesisHash hash.Hash256, Coordinate common.Coordinate, blockStore store.Store, accountStore store.Store) (*Base, error) {
	var height uint32
	if bh, err := blockStore.Get([]byte("height")); err != nil {
	} else {
		height = util.BytesToUint32(bh)
	}

	cn := &Base{
		blockStore:   blockStore,
		accountStore: accountStore,
		height:       height,
		coordinate:   Coordinate,
		genesisHash:  GenesisHash,
		config:       config,
	}
	return cn, nil
}

// Coordinate TODO
func (cn *Base) Coordinate() common.Coordinate {
	var coord common.Coordinate
	copy(coord[:], cn.coordinate[:])
	return coord
}

// Genesis TODO
func (cn *Base) Genesis() hash.Hash256 {
	var h hash.Hash256
	copy(h[:], cn.genesisHash[:])
	return h
}

// Config TODO
func (cn *Base) Config() Config {
	return (*cn.config)
}

// HashCurrentBlock TODO
func (cn *Base) HashCurrentBlock() (hash.Hash256, error) {
	var prevHash hash.Hash256
	if cn.Height() == 0 {
		prevHash = cn.Genesis()
	} else {
		h, err := cn.BlockHash(cn.Height())
		if err != nil {
			return hash.Hash256{}, err
		}
		prevHash = h
	}
	return prevHash, nil
}

// Account TODO
func (cn *Base) Account(addr common.Address) (*account.Account, error) {
	if v, err := cn.accountStore.Get(addr[:]); err != nil {
		return nil, err
	} else {
		acc := new(account.Account)
		if _, err := acc.ReadFrom(bytes.NewReader(v)); err != nil {
			return nil, err
		}
		return acc, nil
	}
}

// UpdateAccount TODO
func (cn *Base) UpdateAccount(acc *account.Account) error {
	var buffer bytes.Buffer
	if _, err := acc.WriteTo(&buffer); err != nil {
		return err
	} else if err := cn.accountStore.Set(acc.Address[:], buffer.Bytes()); err != nil {
		return err
	}
	return nil
}

// BlockReward TODO TEMP
func (cn *Base) BlockReward(height uint32) *amount.Amount {
	return amount.COIN.MulC(10)
}

// Fee TODO TEMP
func (cn *Base) Fee(t transaction.Transaction) *amount.Amount {
	var baseFee = amount.COIN.DivC(10)
	switch tx := t.(type) {
	case *advanced.Trade:
		return baseFee.MulC(int64(len(tx.Vout)))
	case *advanced.Formulation:
		return baseFee.Add(cn.config.FormulationCost)
	case *advanced.RevokeFormulation:
		return baseFee
	case *advanced.MultiSigAccount:
		return baseFee.Add(cn.config.MultiSigAccountCost)
	default:
		panic("Unknown transaction type fee : " + block.TypeNameOfTransaction(tx))
	}
}

// Close TODO
func (cn *Base) Close() {
	cn.blockStore.Close()
	cn.accountStore.Close()
}

// ConnectBlock TODO
func (cn *Base) ConnectBlock(b *block.Block, s *block.ObserverSigned, ExpectedPublicKey common.PublicKey) ([]hash.Hash256, error) {
	prevHash, err := cn.HashCurrentBlock()
	if err != nil {
		return nil, err
	}
	if !b.Header.HashPrevBlock.Equal(prevHash) {
		return nil, ErrMismatchHashPrevBlock
	}

	if err := ValidateBlockGeneratorSignature(b, s.GeneratorSignature, ExpectedPublicKey); err != nil {
		return nil, err
	}
	height := cn.Height() + 1

	ctx := NewValidationContext()
	TxHashes := make([]hash.Hash256, 0, len(b.Transactions))
	for idx, tx := range b.Transactions {
		txHash, err := tx.Hash()
		if err != nil {
			return nil, err
		}
		TxHashes = append(TxHashes, txHash)

		sigs := b.TransactionSignatures[idx]
		addrs := make([]common.Address, 0, len(sigs))
		for _, sig := range sigs {
			if pubkey, err := common.RecoverPubkey(txHash, sig); err != nil {
				return nil, err
			} else {
				addrs = append(addrs, common.AddressFromPubkey(cn.Coordinate(), KeyAccountType, pubkey))
			}
		}
		if err := validateTransactionWithResult(ctx, cn, tx, addrs, uint16(idx)); err != nil {
			return nil, err
		}
	}
	root, err := level.BuildLevelRoot(TxHashes)
	if err != nil {
		return nil, err
	}
	if !b.Header.HashLevelRoot.Equal(hash.TwoHash(prevHash, root)) {
		return nil, ErrMismatchHashLevelRoot
	}

	formulationAcc, err := ctx.LoadAccount(cn, b.Header.FormulationAddress)
	if err != nil {
		return nil, err
	}
	formulationAcc.Balance = formulationAcc.Balance.Add(cn.BlockReward(height))

	for _, acc := range ctx.AccountHash {
		if err := cn.UpdateAccount(acc); err != nil {
			return nil, err
		}
	}

	if err := cn.writeBlock(height, b, s); err != nil {
		return nil, err
	}

	if err := cn.blockStore.Set([]byte("height"), util.Uint32ToBytes(height)); err != nil {
		return nil, err
	}
	cn.height = height

	return TxHashes, nil
}

// Height TODO
func (cn *Base) Height() uint32 {
	return cn.height
}

// Block TODO
func (cn *Base) Block(height uint32) (*block.Block, error) {
	if height > cn.Height() {
		return nil, ErrExceedChainHeight
	}
	if v, err := cn.blockStore.Get(toHeightBlockKey(height)); err != nil {
		return nil, err
	} else {
		b := new(block.Block)
		if _, err := b.ReadFrom(bytes.NewReader(v)); err != nil {
			return nil, err
		}
		return b, nil
	}
}

// BlockByHash TODO
func (cn *Base) BlockByHash(h hash.Hash256) (*block.Block, error) {
	if height, err := cn.blockStore.Get(toHashBlockHeightKey(h)); err != nil {
		return nil, err
	} else {
		return cn.Block(util.BytesToUint32(height))
	}
}

// BlockHash TODO
func (cn *Base) BlockHash(height uint32) (hash.Hash256, error) {
	if height > cn.Height() {
		return hash.Hash256{}, ErrExceedChainHeight
	}
	if v, err := cn.blockStore.Get(toHeightBlockHashKey(height)); err != nil {
		return hash.Hash256{}, err
	} else {
		var h hash.Hash256
		if _, err := h.ReadFrom(bytes.NewReader(v)); err != nil {
			return hash.Hash256{}, err
		}
		return h, nil
	}
}

// ObserverSigned TODO
func (cn *Base) ObserverSigned(height uint32) (*block.ObserverSigned, error) {
	if height > cn.Height() {
		return nil, ErrExceedChainHeight
	}
	if v, err := cn.blockStore.Get(toHeightObserverSignedKey(height)); err != nil {
		return nil, err
	} else {
		s := new(block.ObserverSigned)
		if _, err := s.ReadFrom(bytes.NewReader(v)); err != nil {
			return nil, err
		}
		return s, nil
	}
}

// ObserverSignedByHash TODO
func (cn *Base) ObserverSignedByHash(h hash.Hash256) (*block.ObserverSigned, error) {
	if height, err := cn.blockStore.Get(toHashBlockHeightKey(h)); err != nil {
		return nil, err
	} else {
		return cn.ObserverSigned(util.BytesToUint32(height))
	}
}

// Transactions TODO
func (cn *Base) Transactions(height uint32) ([]transaction.Transaction, error) {
	if height > cn.Height() {
		return nil, ErrExceedChainHeight
	}
	if b, err := cn.Block(height); err != nil {
		return nil, err
	} else {
		return b.Transactions, nil
	}
}

// Transaction TODO
func (cn *Base) Transaction(height uint32, index uint16) (transaction.Transaction, error) {
	if height > cn.Height() {
		return nil, ErrExceedChainHeight
	}
	if txs, err := cn.Transactions(height); err != nil {
		return nil, err
	} else {
		if index >= uint16(len(txs)) {
			return nil, ErrExceedTransactionIndex
		}
		return txs[index], nil
	}
}

func (cn *Base) writeBlock(height uint32, b *block.Block, s *block.ObserverSigned) error {
	{
		var buffer bytes.Buffer
		if _, err := b.WriteTo(&buffer); err != nil {
			return err
		} else if err := cn.blockStore.Set(toHeightBlockKey(height), buffer.Bytes()); err != nil {
			return err
		}
	}

	{
		var buffer bytes.Buffer
		if _, err := s.WriteTo(&buffer); err != nil {
			return err
		} else if err := cn.blockStore.Set(toHeightObserverSignedKey(height), buffer.Bytes()); err != nil {
			return err
		}
	}

	if h, err := s.Hash(); err != nil {
		return err
	} else if err := cn.blockStore.Set(toHeightBlockHashKey(height), h[:]); err != nil {
		return err
	} else if err := cn.blockStore.Set(toHashBlockHeightKey(h), util.Uint32ToBytes(height)); err != nil {
		return err
	}
	return nil
}
