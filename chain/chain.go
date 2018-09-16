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
	AccountProvider
	BlockProvider
	Config() *Config
	GenesisHash() hash.Hash256
	Coordinate() *common.Coordinate
	ObserverPubkeys() []common.PublicKey
	FormulationHash() map[string]common.PublicKey
	Fee(tx transaction.Transaction) *amount.Amount
	BlockReward(height uint32) *amount.Amount
}

// AccountProvider TODO
type AccountProvider interface {
	Account(addr common.Address) (*account.Account, error)
	AccountData(addr common.Address, name string) ([]byte, error)
	Addresses(pubhash common.PublicHash) ([]common.Address, error)
}

// BlockProvider TODO
type BlockProvider interface {
	Height() uint32
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
	UpdateAccountData(addr common.Address, name string, data []byte) error
	ConnectBlock(b *block.Block, s *block.ObserverSigned, ExpectedPublicKey common.PublicKey) ([]hash.Hash256, error)
}

// Base TODO
type Base struct {
	blockStore      store.Store
	accountStore    store.Store
	dataStore       store.Store
	hashStore       store.Store
	height          uint32
	hashPrevBlock   hash.Hash256
	genesis         *Genesis
	genesisHash     hash.Hash256
	observerPubkeys []common.PublicKey
	config          *Config
	formulationHash map[string]common.PublicKey
}

// NewBase TODO
func NewBase(config *Config, genesis *Genesis, blockStore store.Store, accountStore store.Store, dataStore store.Store, hashStore store.Store) (*Base, error) {
	cn := &Base{
		blockStore:      blockStore,
		accountStore:    accountStore,
		dataStore:       dataStore,
		hashStore:       hashStore,
		genesis:         genesis,
		config:          config,
		formulationHash: map[string]common.PublicKey{},
	}

	cn.observerPubkeys = cn.genesis.ObserverPubkeys
	if h, err := genesis.Hash(); err != nil {
		return nil, err
	} else {
		cn.genesisHash = h
	}

	if bh, err := blockStore.Get([]byte("height")); err != nil {
		if err := cn.initGenesisAccount(); err != nil {
			return nil, err
		}
	} else {
		cn.height = util.BytesToUint32(bh)
	}
	return cn, nil
}

func (cn *Base) initGenesisAccount() error {
	if cn.Height() > 0 {
		return ErrInvalidHeight
	}

	ctx := NewValidationContext()
	for i, v := range cn.genesis.Accounts {
		addr := common.NewAddress(v.Type, 0, uint16(i))
		switch v.Type {
		case SingleAccountType:
			if v.UnlockHeight > 0 {
				return ErrInvalidUnlockHeight
			}
			if len(v.KeyHashes) > 1 {
				return ErrExceedAddressCount
			}
			acc := CreateAccount(cn, addr, v.KeyHashes)
			acc.Balance = acc.Balance.Add(v.Amount)
			ctx.AccountHash[string(addr[:])] = acc
		case LockedAccountType:
			if v.UnlockHeight == 0 {
				return ErrInvalidUnlockHeight
			}
			if len(v.KeyHashes) > 1 {
				return ErrExceedAddressCount
			}
			acc := CreateAccount(cn, addr, v.KeyHashes)
			acc.Balance = acc.Balance.Add(v.Amount)
			ctx.AccountHash[string(addr[:])] = acc
			ctx.AccountDataHash[string(toAccountDataKey(addr, "UnlockHeight"))] = util.Uint32ToBytes(v.UnlockHeight)
		case MultiSigAccountType:
			if v.UnlockHeight > 0 {
				return ErrInvalidUnlockHeight
			}
			if v.Required == 0 || int(v.Required) > len(v.KeyHashes) {
				return ErrInvalidMultiSigRequired
			}
			acc := CreateAccount(cn, addr, v.KeyHashes)
			acc.Balance = acc.Balance.Add(v.Amount)
			ctx.AccountHash[string(addr[:])] = acc
			ctx.AccountDataHash[string(toAccountDataKey(addr, "Required"))] = []byte{byte(v.Required)}
		case FormulationAccountType:
			if v.UnlockHeight > 0 {
				return ErrInvalidUnlockHeight
			}
			if !v.Amount.IsZero() {
				return ErrInvalidAmount
			}
			acc := CreateAccount(cn, addr, v.KeyHashes)
			ctx.AccountHash[string(addr[:])] = acc
			ctx.AccountDataHash[string(toAccountDataKey(addr, "PublicKey"))] = v.PublicKey[:]
		default:
			return ErrInvalidGenesisAccountType
		}
	}

	for key, data := range ctx.AccountDataHash {
		if err := cn.updateAccountDataByKey([]byte(key), data); err != nil {
			return err
		}
	}
	for _, acc := range ctx.AccountHash {
		if err := cn.UpdateAccount(acc); err != nil {
			return err
		}
	}

	if err := cn.blockStore.Set([]byte("height"), util.Uint32ToBytes(0)); err != nil {
		return err
	}
	return nil
}

// Coordinate TODO
func (cn *Base) Coordinate() *common.Coordinate {
	return cn.genesis.Coordinate.Clone()
}

// ObserverPubkeys TODO
func (cn *Base) ObserverPubkeys() []common.PublicKey {
	pubkeys := make([]common.PublicKey, 0, len(cn.observerPubkeys))
	for _, key := range cn.observerPubkeys {
		var pubkey common.PublicKey
		copy(pubkey[:], key[:])
		pubkeys = append(pubkeys, pubkey)
	}
	return pubkeys
}

// GenesisHash TODO
func (cn *Base) GenesisHash() hash.Hash256 {
	var genesisHash hash.Hash256
	copy(genesisHash[:], cn.genesisHash[:])
	return genesisHash
}

// Config TODO
func (cn *Base) Config() *Config {
	return cn.config.Clone()
}

// HashCurrentBlock TODO
func (cn *Base) HashCurrentBlock() (hash.Hash256, error) {
	var curHash hash.Hash256
	if cn.Height() == 0 {
		curHash = cn.GenesisHash()
	} else {
		h, err := cn.BlockHash(cn.Height())
		if err != nil {
			return hash.Hash256{}, err
		}
		curHash = h
	}
	return curHash, nil
}

// FormulationHash TODO
func (cn *Base) FormulationHash() map[string]common.PublicKey {
	hash := map[string]common.PublicKey{}
	for k, v := range cn.formulationHash {
		var pubkey common.PublicKey
		copy(pubkey[:], v[:])
		hash[k] = pubkey
	}
	return hash
}

// Addresses TODO
func (cn *Base) Addresses(pubhash common.PublicHash) ([]common.Address, error) {
	if _, values, err := cn.hashStore.Scan(pubhash[:]); err != nil {
		return nil, err
	} else {
		list := make([]common.Address, 0, len(values))
		for _, v := range values {
			var addr common.Address
			copy(addr[:], v)
			list = append(list, addr)
		}
		return list, nil
	}
}

// Account TODO
func (cn *Base) Account(addr common.Address) (*account.Account, error) {
	if v, err := cn.accountStore.Get(addr[:]); err != nil {
		return nil, err
	} else {
		acc := &account.Account{
			Balance:  amount.NewCoinAmount(0, 0),
			IsExist_: true,
		}
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
	} else {
		if !acc.IsExist_ {
			switch acc.Address.Type() {
			case FormulationAccountType:
				bs, err := cn.AccountData(acc.Address, "PublicKey")
				if err != nil {
					return err
				}
				var pubkey common.PublicKey
				copy(pubkey[:], bs)
				cn.formulationHash[string(acc.Address[:])] = pubkey
			}
			for _, ph := range acc.KeyHashes {
				bs := make([]byte, common.PublicHashSize+common.AddressSize)
				copy(bs[:], ph[:])
				copy(bs[common.PublicHashSize:], acc.Address[:])
				if err := cn.hashStore.Set(bs, acc.Address[:]); err != nil {
					return err
				}
			}
		}
		if err := cn.accountStore.Set(acc.Address[:], buffer.Bytes()); err != nil {
			return err
		}
	}
	return nil
}

// AccountData TODO
func (cn *Base) AccountData(addr common.Address, name string) ([]byte, error) {
	if bs, err := cn.dataStore.Get(toAccountDataKey(addr, name)); err != nil {
		return nil, err
	} else {
		return bs, nil
	}
}

// UpdateAccountData TODO
func (cn *Base) UpdateAccountData(addr common.Address, name string, data []byte) error {
	return cn.updateAccountDataByKey(toAccountDataKey(addr, name), data)
}

func (cn *Base) updateAccountDataByKey(key []byte, data []byte) error {
	if err := cn.dataStore.Set(key, data); err != nil {
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
	case *advanced.TaggedTrade:
		return baseFee.MulC(2)
	case *advanced.Formulation:
		return baseFee.Add(cn.config.FormulationCost)
	case *advanced.RevokeFormulation:
		return baseFee
	case *advanced.SingleAccount:
		return baseFee.Add(cn.config.SingleAccountCost)
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

	if len(b.Transactions) != len(b.TransactionSignatures) {
		return nil, ErrMismatchSignaturesCount
	}

	ctx := NewValidationContext()
	TxHashes := make([]hash.Hash256, 0, len(b.Transactions))
	for idx, tx := range b.Transactions {
		txHash, err := tx.Hash()
		if err != nil {
			return nil, err
		}
		TxHashes = append(TxHashes, txHash)

		sigs := b.TransactionSignatures[idx]
		signers := make([]common.PublicHash, 0, len(sigs))
		for _, sig := range sigs {
			if pubkey, err := common.RecoverPubkey(txHash, sig); err != nil {
				return nil, err
			} else {
				signers = append(signers, common.NewPublicHash(pubkey))
			}
		}
		ctx.CurrentTxHash = txHash
		if err := validateTransactionWithResult(ctx, cn, tx, signers, uint16(idx)); err != nil {
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

	formulationAcc, err := ctx.LoadAccount(cn, b.Header.FormulationAddress, false)
	if err != nil {
		return nil, err
	}
	formulationAcc.Balance = formulationAcc.Balance.Add(cn.BlockReward(height))

	for key, data := range ctx.AccountDataHash {
		if err := cn.updateAccountDataByKey([]byte(key), data); err != nil {
			return nil, err
		}
	}
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
