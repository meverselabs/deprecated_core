package data

import (
	"bytes"
	"sort"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/account"
	"git.fleta.io/fleta/core/db"
	"git.fleta.io/fleta/core/transaction"
)

// Context TODO
type Context struct {
	loader Loader
	stack  []*ContextData
}

// NewContext TODO
func NewContext(loader Loader) *Context {
	ctx := &Context{
		loader: loader,
		stack:  []*ContextData{NewContextData(loader, nil)},
	}
	return ctx
}

// ChainCoord TODO
func (ctx *Context) ChainCoord() *common.Coordinate {
	return ctx.loader.ChainCoord()
}

// IsMainChain TODO
func (ctx *Context) IsMainChain() bool {
	coord := ctx.ChainCoord()
	return coord.Height == 0 && coord.Index == 0
}

// Top TODO
func (ctx *Context) Top() *ContextData {
	return ctx.stack[len(ctx.stack)-1]
}

// Seq TODO
func (ctx *Context) Seq(addr common.Address) uint64 {
	return ctx.Top().Seq(addr)
}

// AddSeq TODO
func (ctx *Context) AddSeq(addr common.Address) {
	ctx.Top().AddSeq(addr)
}

// Account TODO
func (ctx *Context) Account(addr common.Address) (account.Account, error) {
	return ctx.Top().Account(addr)
}

// IsExistAccount TODO
func (ctx *Context) IsExistAccount(addr common.Address) (bool, error) {
	if _, err := ctx.Account(addr); err != nil {
		if err != db.ErrNotExistKey {
			return true, err
		}
		return false, nil
	} else {
		return true, nil
	}
}

// CreateAccount TODO
func (ctx *Context) CreateAccount(acc account.Account) error {
	return ctx.Top().CreateAccount(acc)
}

// DeleteAccount TODO
func (ctx *Context) DeleteAccount(acc account.Account) error {
	return ctx.Top().DeleteAccount(acc)
}

// AccountData TODO
func (ctx *Context) AccountData(addr common.Address, name []byte) []byte {
	return ctx.Top().AccountData(addr, name)
}

// SetAccountData TODO
func (ctx *Context) SetAccountData(addr common.Address, name []byte, value []byte) {
	ctx.Top().SetAccountData(addr, name, value)
}

// UTXO TODO
func (ctx *Context) UTXO(id uint64) (*transaction.UTXO, error) {
	return ctx.Top().UTXO(id)
}

// CreateUTXO TODO
func (ctx *Context) CreateUTXO(id uint64, vout *transaction.TxOut) error {
	return ctx.Top().CreateUTXO(id, vout)
}

// DeleteUTXO TODO
func (ctx *Context) DeleteUTXO(id uint64) error {
	return ctx.Top().DeleteUTXO(id)
}

// Snapshot TODO
func (ctx *Context) Snapshot() int {
	ctd := NewContextData(ctx.loader, ctx.Top())
	ctx.stack = append(ctx.stack, ctd)
	return len(ctx.stack)
}

// Revert TODO
func (ctx *Context) Revert(sn int) {
	if len(ctx.stack) >= sn {
		ctx.stack = ctx.stack[:sn-1]
	}
}

// Commit TODO
func (ctx *Context) Commit(sn int) {
	for len(ctx.stack) >= sn {
		ctd := ctx.Top()
		ctx.stack = ctx.stack[:len(ctx.stack)-1]
		top := ctx.Top()
		for k, v := range ctd.SeqHash {
			top.SeqHash[k] = v
		}
		for k, v := range ctd.AccountHash {
			top.AccountHash[k] = v
		}
		for k, v := range ctd.CreatedAccountHash {
			top.CreatedAccountHash[k] = v
		}
		for k, v := range ctd.DeletedAccountHash {
			delete(top.AccountHash, k)
			delete(top.CreatedAccountHash, k)
			top.DeletedAccountHash[k] = v
		}
		for k, v := range ctd.AccountDataHash {
			top.AccountDataHash[k] = v
		}
		for k, v := range ctd.DeletedAccountDataHash {
			delete(top.AccountDataHash, k)
			top.DeletedAccountDataHash[k] = v
		}
		for k, v := range ctd.UTXOHash {
			top.UTXOHash[k] = v
		}
		for k, v := range ctd.CreatedUTXOHash {
			top.CreatedUTXOHash[k] = v
		}
		for k, v := range ctd.DeletedUTXOHash {
			delete(top.UTXOHash, k)
			delete(top.CreatedUTXOHash, k)
			top.DeletedUTXOHash[k] = v
		}
	}
}

// StackSize TODO
func (ctx *Context) StackSize() int {
	return len(ctx.stack)
}

// ContextData TODO
type ContextData struct {
	loader                 Loader
	Parent                 *ContextData
	SeqHash                map[common.Address]uint64
	AccountHash            map[common.Address]account.Account
	CreatedAccountHash     map[common.Address]account.Account
	DeletedAccountHash     map[common.Address]account.Account
	AccountDataHash        map[string][]byte
	DeletedAccountDataHash map[string]bool
	UTXOHash               map[uint64]*transaction.UTXO
	CreatedUTXOHash        map[uint64]*transaction.TxOut
	DeletedUTXOHash        map[uint64]bool
}

// NewContextData TODO
func NewContextData(loader Loader, Parent *ContextData) *ContextData {
	ctd := &ContextData{
		loader:                 loader,
		Parent:                 Parent,
		SeqHash:                map[common.Address]uint64{},
		AccountHash:            map[common.Address]account.Account{},
		CreatedAccountHash:     map[common.Address]account.Account{},
		DeletedAccountHash:     map[common.Address]account.Account{},
		AccountDataHash:        map[string][]byte{},
		DeletedAccountDataHash: map[string]bool{},
		UTXOHash:               map[uint64]*transaction.UTXO{},
		CreatedUTXOHash:        map[uint64]*transaction.TxOut{},
		DeletedUTXOHash:        map[uint64]bool{},
	}
	return ctd
}

// Hash TODO
func (ctd *ContextData) Hash() hash.Hash256 {
	var buffer bytes.Buffer
	buffer.WriteString("SeqHash")
	{
		keys := []common.Address{}
		for k := range ctd.SeqHash {
			keys = append(keys, k)
		}
		sort.Sort(addressSlice(keys))
		for _, k := range keys {
			v := ctd.SeqHash[k]
			if _, err := k.WriteTo(&buffer); err != nil {
				panic(err)
			}
			if _, err := util.WriteUint64(&buffer, v); err != nil {
				panic(err)
			}
		}
	}
	buffer.WriteString("AccountHash")
	{
		keys := []common.Address{}
		for k := range ctd.AccountHash {
			keys = append(keys, k)
		}
		sort.Sort(addressSlice(keys))
		for _, k := range keys {
			v := ctd.AccountHash[k]
			if _, err := k.WriteTo(&buffer); err != nil {
				panic(err)
			}
			if _, err := v.WriteTo(&buffer); err != nil {
				panic(err)
			}
		}
	}
	buffer.WriteString("CreatedAccountHash")
	{
		keys := []common.Address{}
		for k := range ctd.CreatedAccountHash {
			keys = append(keys, k)
		}
		sort.Sort(addressSlice(keys))
		for _, k := range keys {
			v := ctd.CreatedAccountHash[k]
			if _, err := k.WriteTo(&buffer); err != nil {
				panic(err)
			}
			if _, err := v.WriteTo(&buffer); err != nil {
				panic(err)
			}
		}
	}
	buffer.WriteString("DeletedAccountHash")
	{
		keys := []common.Address{}
		for k := range ctd.DeletedAccountHash {
			keys = append(keys, k)
		}
		sort.Sort(addressSlice(keys))
		for _, k := range keys {
			if _, err := k.WriteTo(&buffer); err != nil {
				panic(err)
			}
		}
	}
	buffer.WriteString("AccountDataHash")
	{
		keys := []string{}
		for k := range ctd.AccountDataHash {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := ctd.AccountDataHash[k]
			buffer.WriteString(k)
			buffer.Write(v)
		}
	}
	buffer.WriteString("DeletedAccountDataHash")
	{
		keys := []string{}
		for k := range ctd.DeletedAccountDataHash {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			buffer.WriteString(k)
		}
	}
	buffer.WriteString("UTXOHash")
	{
		keys := []uint64{}
		for k := range ctd.UTXOHash {
			keys = append(keys, k)
		}
		sort.Sort(uint64Slice(keys))
		for _, k := range keys {
			v := ctd.UTXOHash[k]
			if _, err := util.WriteUint64(&buffer, k); err != nil {
				panic(err)
			}
			if _, err := v.WriteTo(&buffer); err != nil {
				panic(err)
			}
		}
	}
	buffer.WriteString("CreatedUTXOHash")
	{
		keys := []uint64{}
		for k := range ctd.CreatedUTXOHash {
			keys = append(keys, k)
		}
		sort.Sort(uint64Slice(keys))
		for _, k := range keys {
			v := ctd.CreatedUTXOHash[k]
			if _, err := util.WriteUint64(&buffer, k); err != nil {
				panic(err)
			}
			if _, err := v.WriteTo(&buffer); err != nil {
				panic(err)
			}
		}
	}
	buffer.WriteString("DeletedUTXOHash")
	{
		keys := []uint64{}
		for k := range ctd.DeletedUTXOHash {
			keys = append(keys, k)
		}
		sort.Sort(uint64Slice(keys))
		for _, k := range keys {
			if _, err := util.WriteUint64(&buffer, k); err != nil {
				panic(err)
			}
		}
	}
	return hash.Hash(buffer.Bytes())
}

// Seq TODO
func (ctd *ContextData) Seq(addr common.Address) uint64 {
	if _, has := ctd.DeletedAccountHash[addr]; has {
		return 0
	}
	if seq, has := ctd.SeqHash[addr]; has {
		return seq
	} else if ctd.Parent != nil {
		seq := ctd.Parent.Seq(addr)
		if seq > 0 {
			ctd.SeqHash[addr] = seq
		}
		return seq
	} else {
		seq := ctd.loader.Seq(addr)
		if seq > 0 {
			ctd.SeqHash[addr] = seq
		}
		return seq
	}
}

// AddSeq TODO
func (ctd *ContextData) AddSeq(addr common.Address) {
	if _, has := ctd.DeletedAccountHash[addr]; has {
		return
	}
	ctd.SeqHash[addr] = ctd.Seq(addr) + 1
}

// Account TODO
func (ctd *ContextData) Account(addr common.Address) (account.Account, error) {
	if _, has := ctd.DeletedAccountHash[addr]; has {
		return nil, db.ErrNotExistKey
	}
	if acc, has := ctd.AccountHash[addr]; has {
		return acc, nil
	} else if acc, has := ctd.CreatedAccountHash[addr]; has {
		return acc, nil
	} else if ctd.Parent != nil {
		if acc, err := ctd.Parent.Account(addr); err != nil {
			return nil, err
		} else {
			nacc := acc.Clone()
			ctd.AccountHash[addr] = nacc
			return nacc, nil
		}
	} else {
		if acc, err := ctd.loader.Account(addr); err != nil {
			return nil, err
		} else {
			ctd.AccountHash[addr] = acc
			return acc, nil
		}
	}

}

// CreateAccount TODO
func (ctd *ContextData) CreateAccount(acc account.Account) error {
	if _, err := ctd.Account(acc.Address()); err != nil {
		if err != db.ErrNotExistKey {
			return err
		}
	} else {
		return ErrExistAccount
	}
	ctd.CreatedAccountHash[acc.Address()] = acc
	return nil
}

// DeleteAccount TODO
func (ctd *ContextData) DeleteAccount(acc account.Account) error {
	if _, err := ctd.Account(acc.Address()); err != nil {
		return err
	}
	ctd.DeletedAccountHash[acc.Address()] = acc
	delete(ctd.AccountHash, acc.Address())
	return nil
}

// AccountData TODO
func (ctd *ContextData) AccountData(addr common.Address, name []byte) []byte {
	key := string(addr[:]) + string(name)
	if ctd.DeletedAccountDataHash[key] {
		return nil
	}
	if value, has := ctd.AccountDataHash[key]; has {
		return value
	} else if ctd.Parent != nil {
		value := ctd.Parent.AccountData(addr, name)
		if len(value) > 0 {
			nvalue := make([]byte, len(value))
			copy(nvalue, value)
			ctd.AccountDataHash[key] = nvalue
			return nvalue
		} else {
			return nil
		}
	} else {
		value := ctd.loader.AccountData(addr, name)
		if len(value) > 0 {
			ctd.AccountDataHash[key] = value
			return value
		} else {
			return nil
		}
	}
}

// SetAccountData TODO
func (ctd *ContextData) SetAccountData(addr common.Address, name []byte, value []byte) {
	key := string(addr[:]) + string(name)
	if len(value) == 0 {
		delete(ctd.AccountDataHash, key)
		ctd.DeletedAccountDataHash[key] = true
	} else {
		delete(ctd.DeletedAccountDataHash, key)
		ctd.AccountDataHash[key] = value
	}
}

// UTXO TODO
func (ctd *ContextData) UTXO(id uint64) (*transaction.UTXO, error) {
	if ctd.DeletedUTXOHash[id] {
		return nil, ErrDoubleSpent
	}
	if utxo, has := ctd.UTXOHash[id]; has {
		return utxo, nil
	} else if ctd.Parent != nil {
		if utxo, err := ctd.Parent.UTXO(id); err != nil {
			return nil, err
		} else {
			nutxo := utxo.Clone()
			ctd.UTXOHash[id] = nutxo
			return nutxo, nil
		}
	} else {
		if utxo, err := ctd.loader.UTXO(id); err != nil {
			return nil, err
		} else {
			ctd.UTXOHash[id] = utxo
			return utxo, nil
		}
	}
}

// CreateUTXO TODO
func (ctd *ContextData) CreateUTXO(id uint64, vout *transaction.TxOut) error {
	if _, err := ctd.UTXO(id); err != nil {
		if err != db.ErrNotExistKey {
			return err
		}
	} else {
		return ErrExistAccount
	}
	ctd.CreatedUTXOHash[id] = vout
	return nil
}

// DeleteUTXO TODO
func (ctd *ContextData) DeleteUTXO(id uint64) error {
	if _, err := ctd.UTXO(id); err != nil {
		return err
	}
	ctd.DeletedUTXOHash[id] = true
	return nil
}

type addressSlice []common.Address

func (p addressSlice) Len() int           { return len(p) }
func (p addressSlice) Less(i, j int) bool { return bytes.Compare(p[i][:], p[j][:]) < 0 }
func (p addressSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type uint64Slice []uint64

func (p uint64Slice) Len() int           { return len(p) }
func (p uint64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p uint64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
