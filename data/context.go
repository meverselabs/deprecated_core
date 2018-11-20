package data

import (
	"bytes"
	"sort"
	"strconv"

	"git.fleta.io/fleta/common"
	"git.fleta.io/fleta/common/hash"
	"git.fleta.io/fleta/common/util"
	"git.fleta.io/fleta/core/account"
	"git.fleta.io/fleta/core/transaction"
)

// Context is an intermediate in-memory state using the context data stack between blocks
type Context struct {
	loader          Loader
	genTargetHeight uint32
	getPrevHash     hash.Hash256
	cache           *cache
	stack           []*ContextData
	isOldHash       bool
	dataHash        hash.Hash256
}

// NewContext returns a Context
func NewContext(loader Loader) *Context {
	ctx := &Context{
		loader:          loader,
		genTargetHeight: loader.TargetHeight(),
		getPrevHash:     loader.LastBlockHash(),
		stack:           []*ContextData{NewContextData(loader, nil)},
	}
	ctx.cache = newCache(ctx)
	return ctx
}

// Hash returns the hash value of it
func (ctx *Context) Hash() hash.Hash256 {
	if ctx.isOldHash {
		ctx.dataHash = ctx.Top().Hash()
		ctx.isOldHash = false
	}
	return ctx.dataHash
}

// ChainCoord returns the coordinate of the target chain
func (ctx *Context) ChainCoord() *common.Coordinate {
	return ctx.loader.ChainCoord()
}

// Accounter returns the accounter of the target chain
func (ctx *Context) Accounter() *Accounter {
	return ctx.loader.Accounter()
}

// Transactor returns the transactor of the target chain
func (ctx *Context) Transactor() *Transactor {
	return ctx.loader.Transactor()
}

// TargetHeight returns the recorded target height when context generation
func (ctx *Context) TargetHeight() uint32 {
	return ctx.genTargetHeight
}

// LastBlockHash returns the last block hash of the target chain
func (ctx *Context) LastBlockHash() hash.Hash256 {
	return ctx.getPrevHash
}

// Top returns the top snapshot
func (ctx *Context) Top() *ContextData {
	return ctx.stack[len(ctx.stack)-1]
}

// Seq returns the sequence of the target account
func (ctx *Context) Seq(addr common.Address) uint64 {
	return ctx.Top().Seq(addr)
}

// AddSeq update the sequence of the target account
func (ctx *Context) AddSeq(addr common.Address) {
	ctx.isOldHash = true
	ctx.Top().AddSeq(addr)
}

// Account returns the account instance of the address
func (ctx *Context) Account(addr common.Address) (account.Account, error) {
	ctx.isOldHash = true
	return ctx.Top().Account(addr)
}

// IsExistAccount checks that the account of the address is exist or not
func (ctx *Context) IsExistAccount(addr common.Address) (bool, error) {
	return ctx.Top().IsExistAccount(addr)
}

// CreateAccount inserts the account to the top snapshot
func (ctx *Context) CreateAccount(acc account.Account) error {
	ctx.isOldHash = true
	return ctx.Top().CreateAccount(acc)
}

// DeleteAccount deletes the account from the top snapshot
func (ctx *Context) DeleteAccount(acc account.Account) error {
	ctx.isOldHash = true
	return ctx.Top().DeleteAccount(acc)
}

// AccountBalance returns the account balance
func (ctx *Context) AccountBalance(addr common.Address) (*account.Balance, error) {
	ctx.isOldHash = true
	return ctx.Top().AccountBalance(addr)
}

// AccountData returns the account data from the top snapshot
func (ctx *Context) AccountData(addr common.Address, name []byte) []byte {
	return ctx.Top().AccountData(addr, name)
}

// SetAccountData inserts the account data to the top snapshot
func (ctx *Context) SetAccountData(addr common.Address, name []byte, value []byte) {
	ctx.isOldHash = true
	ctx.Top().SetAccountData(addr, name, value)
}

// UTXO returns the UTXO from the top snapshot
func (ctx *Context) UTXO(id uint64) (*transaction.UTXO, error) {
	return ctx.Top().UTXO(id)
}

// CreateUTXO inserts the UTXO to the top snapshot
func (ctx *Context) CreateUTXO(id uint64, vout *transaction.TxOut) error {
	ctx.isOldHash = true
	return ctx.Top().CreateUTXO(id, vout)
}

// DeleteUTXO deletes the UTXO from the top snapshot
func (ctx *Context) DeleteUTXO(id uint64) error {
	ctx.isOldHash = true
	return ctx.Top().DeleteUTXO(id)
}

func (ctx *Context) Dump() string {
	return ctx.Top().Dump()
}

// Snapshot push a snapshot and returns the snapshot number of it
func (ctx *Context) Snapshot() int {
	ctx.isOldHash = true
	ctd := NewContextData(ctx.cache, ctx.Top())
	ctx.stack[len(ctx.stack)-1].isTop = false
	ctx.stack = append(ctx.stack, ctd)
	return len(ctx.stack)
}

// Revert removes snapshots after the snapshot number
func (ctx *Context) Revert(sn int) {
	ctx.isOldHash = true
	if len(ctx.stack) >= sn {
		ctx.stack = ctx.stack[:sn-1]
	}
	ctx.stack[len(ctx.stack)-1].isTop = true
}

// Commit apply snapshots to the top after the snapshot number
func (ctx *Context) Commit(sn int) {
	ctx.isOldHash = true
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
		for k, v := range ctd.AccountBalanceHash {
			top.AccountBalanceHash[k] = v
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

// StackSize returns the size of the context data stack
func (ctx *Context) StackSize() int {
	return len(ctx.stack)
}

// ContextData is a state data of the context
type ContextData struct {
	loader                 Loader
	Parent                 *ContextData
	SeqHash                map[common.Address]uint64
	AccountHash            map[common.Address]account.Account
	CreatedAccountHash     map[common.Address]account.Account
	DeletedAccountHash     map[common.Address]account.Account
	AccountBalanceHash     map[common.Address]*account.Balance
	AccountDataHash        map[string][]byte
	DeletedAccountDataHash map[string]bool
	UTXOHash               map[uint64]*transaction.UTXO
	CreatedUTXOHash        map[uint64]*transaction.TxOut
	DeletedUTXOHash        map[uint64]bool
	isTop                  bool
}

// NewContextData returns a ContextData
func NewContextData(loader Loader, Parent *ContextData) *ContextData {
	ctd := &ContextData{
		loader:                 loader,
		Parent:                 Parent,
		SeqHash:                map[common.Address]uint64{},
		AccountHash:            map[common.Address]account.Account{},
		CreatedAccountHash:     map[common.Address]account.Account{},
		DeletedAccountHash:     map[common.Address]account.Account{},
		AccountBalanceHash:     map[common.Address]*account.Balance{},
		AccountDataHash:        map[string][]byte{},
		DeletedAccountDataHash: map[string]bool{},
		UTXOHash:               map[uint64]*transaction.UTXO{},
		CreatedUTXOHash:        map[uint64]*transaction.TxOut{},
		DeletedUTXOHash:        map[uint64]bool{},
		isTop:                  true,
	}
	return ctd
}

// Hash returns the hash value of it
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
	buffer.WriteString("AccountBalanceHash")
	{
		keys := []common.Address{}
		for k := range ctd.AccountBalanceHash {
			keys = append(keys, k)
		}
		sort.Sort(addressSlice(keys))
		for _, k := range keys {
			v := ctd.AccountBalanceHash[k]
			if _, err := k.WriteTo(&buffer); err != nil {
				panic(err)
			}
			if _, err := v.WriteTo(&buffer); err != nil {
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

// Seq returns the sequence of the account
func (ctd *ContextData) Seq(addr common.Address) uint64 {
	if _, has := ctd.DeletedAccountHash[addr]; has {
		return 0
	}
	if seq, has := ctd.SeqHash[addr]; has {
		return seq
	} else if ctd.Parent != nil {
		seq := ctd.Parent.Seq(addr)
		if seq > 0 && ctd.isTop {
			ctd.SeqHash[addr] = seq
		}
		return seq
	} else {
		seq := ctd.loader.Seq(addr)
		if seq > 0 && ctd.isTop {
			ctd.SeqHash[addr] = seq
		}
		return seq
	}
}

// AddSeq update the sequence of the target account
func (ctd *ContextData) AddSeq(addr common.Address) {
	if _, has := ctd.DeletedAccountHash[addr]; has {
		return
	}
	ctd.SeqHash[addr] = ctd.Seq(addr) + 1
}

// Account returns the account instance of the address
func (ctd *ContextData) Account(addr common.Address) (account.Account, error) {
	if _, has := ctd.DeletedAccountHash[addr]; has {
		return nil, ErrNotExistAccount
	}
	if acc, has := ctd.AccountHash[addr]; has {
		return acc, nil
	} else if acc, has := ctd.CreatedAccountHash[addr]; has {
		return acc, nil
	} else if ctd.Parent != nil {
		if acc, err := ctd.Parent.Account(addr); err != nil {
			return nil, err
		} else {
			if ctd.isTop {
				nacc := acc.Clone()
				ctd.AccountHash[addr] = nacc
				return nacc, nil
			} else {
				return acc, nil
			}
		}
	} else {
		if acc, err := ctd.loader.Account(addr); err != nil {
			return nil, err
		} else {
			if ctd.isTop {
				nacc := acc.Clone()
				ctd.AccountHash[addr] = nacc
				return nacc, nil
			} else {
				return acc, nil
			}
		}
	}
}

// IsExistAccount checks that the account of the address is exist or not
func (ctd *ContextData) IsExistAccount(addr common.Address) (bool, error) {
	if _, has := ctd.DeletedAccountHash[addr]; has {
		return false, nil
	}
	if _, has := ctd.AccountHash[addr]; has {
		return true, nil
	} else if _, has := ctd.CreatedAccountHash[addr]; has {
		return true, nil
	} else if ctd.Parent != nil {
		return ctd.Parent.IsExistAccount(addr)
	} else {
		return ctd.loader.IsExistAccount(addr)
	}
}

// CreateAccount inserts the account
func (ctd *ContextData) CreateAccount(acc account.Account) error {
	if _, err := ctd.Account(acc.Address()); err != nil {
		if err != ErrNotExistAccount {
			return err
		}
	} else {
		return ErrExistAccount
	}
	ctd.CreatedAccountHash[acc.Address()] = acc
	return nil
}

// DeleteAccount deletes the account
func (ctd *ContextData) DeleteAccount(acc account.Account) error {
	if _, err := ctd.Account(acc.Address()); err != nil {
		return err
	}
	ctd.DeletedAccountHash[acc.Address()] = acc
	delete(ctd.AccountHash, acc.Address())
	return nil
}

// AccountBalance returns the account balance
func (ctd *ContextData) AccountBalance(addr common.Address) (*account.Balance, error) {
	if _, has := ctd.DeletedAccountHash[addr]; has {
		return nil, ErrNotExistAccount
	}
	if bc, has := ctd.AccountBalanceHash[addr]; has {
		return bc, nil
	} else if ctd.Parent != nil {
		if bc, err := ctd.Parent.AccountBalance(addr); err != nil {
			return nil, err
		} else {
			if ctd.isTop {
				nbc := bc.Clone()
				ctd.AccountBalanceHash[addr] = nbc
				return nbc, nil
			} else {
				return bc, nil
			}
		}
	} else {
		if bc, err := ctd.loader.AccountBalance(addr); err != nil {
			return nil, err
		} else {
			if ctd.isTop {
				nbc := bc.Clone()
				ctd.AccountBalanceHash[addr] = nbc
				return nbc, nil
			} else {
				return bc, nil
			}
		}
	}
}

// AccountData returns the account data
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
			if ctd.isTop {
				nvalue := make([]byte, len(value))
				copy(nvalue, value)
				ctd.AccountDataHash[key] = nvalue
				return nvalue
			} else {
				return value
			}
		} else {
			return nil
		}
	} else {
		value := ctd.loader.AccountData(addr, name)
		if len(value) > 0 {
			if ctd.isTop {
				nvalue := make([]byte, len(value))
				copy(nvalue, value)
				ctd.AccountDataHash[key] = nvalue
				return nvalue
			} else {
				return value
			}
		} else {
			return nil
		}
	}
}

// SetAccountData inserts the account data
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

// UTXO returns the UTXO
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
			if ctd.isTop {
				nutxo := utxo.Clone()
				ctd.UTXOHash[id] = nutxo
				return nutxo, nil
			} else {
				return utxo, nil
			}
		}
	} else {
		if utxo, err := ctd.loader.UTXO(id); err != nil {
			return nil, err
		} else {
			if ctd.isTop {
				nutxo := utxo.Clone()
				ctd.UTXOHash[id] = nutxo
				return nutxo, nil
			} else {
				return utxo, nil
			}
		}
	}
}

// CreateUTXO inserts the UTXO
func (ctd *ContextData) CreateUTXO(id uint64, vout *transaction.TxOut) error {
	if _, err := ctd.UTXO(id); err != nil {
		if err != ErrNotExistUTXO {
			return err
		}
	} else {
		return ErrExistUTXO
	}
	ctd.CreatedUTXOHash[id] = vout
	return nil
}

// DeleteUTXO deletes the UTXO
func (ctd *ContextData) DeleteUTXO(id uint64) error {
	if _, err := ctd.UTXO(id); err != nil {
		return err
	}
	ctd.DeletedUTXOHash[id] = true
	return nil
}

// Hash returns the hash value of it
func (ctd *ContextData) Dump() string {
	var buffer bytes.Buffer
	buffer.WriteString("SeqHash\n")
	{
		keys := []common.Address{}
		for k := range ctd.SeqHash {
			keys = append(keys, k)
		}
		sort.Sort(addressSlice(keys))
		for _, k := range keys {
			v := ctd.SeqHash[k]
			buffer.WriteString(k.String())
			buffer.WriteString(": ")
			buffer.WriteString(strconv.FormatInt(int64(v), 10))
			buffer.WriteString("\n")
		}
	}
	buffer.WriteString("\n")
	buffer.WriteString("AccountHash\n")
	{
		keys := []common.Address{}
		for k := range ctd.AccountHash {
			keys = append(keys, k)
		}
		sort.Sort(addressSlice(keys))
		for _, k := range keys {
			v := ctd.AccountHash[k]
			buffer.WriteString(k.String())
			buffer.WriteString(": ")
			var hb bytes.Buffer
			if _, err := v.WriteTo(&hb); err != nil {
				panic(err)
			}
			buffer.WriteString(hash.Hash(hb.Bytes()).String())
			buffer.WriteString("\n")
		}
	}
	buffer.WriteString("\n")
	buffer.WriteString("CreatedAccountHash\n")
	{
		keys := []common.Address{}
		for k := range ctd.CreatedAccountHash {
			keys = append(keys, k)
		}
		sort.Sort(addressSlice(keys))
		for _, k := range keys {
			v := ctd.CreatedAccountHash[k]
			buffer.WriteString(k.String())
			buffer.WriteString(": ")
			var hb bytes.Buffer
			if _, err := v.WriteTo(&hb); err != nil {
				panic(err)
			}
			buffer.WriteString(hash.Hash(hb.Bytes()).String())
			buffer.WriteString("\n")
		}
	}
	buffer.WriteString("\n")
	buffer.WriteString("DeletedAccountHash\n")
	{
		keys := []common.Address{}
		for k := range ctd.DeletedAccountHash {
			keys = append(keys, k)
		}
		sort.Sort(addressSlice(keys))
		for _, k := range keys {
			buffer.WriteString(k.String())
			buffer.WriteString("\n")
		}
	}
	buffer.WriteString("\n")
	buffer.WriteString("AccountBalanceHash\n")
	{
		keys := []common.Address{}
		for k := range ctd.AccountBalanceHash {
			keys = append(keys, k)
		}
		sort.Sort(addressSlice(keys))
		for _, k := range keys {
			v := ctd.AccountBalanceHash[k]
			buffer.WriteString(k.String())
			buffer.WriteString(": ")
			var hb bytes.Buffer
			if _, err := v.WriteTo(&hb); err != nil {
				panic(err)
			}
			buffer.WriteString(hash.Hash(hb.Bytes()).String())
			buffer.WriteString("\n")
		}
	}
	buffer.WriteString("\n")
	buffer.WriteString("AccountDataHash\n")
	{
		keys := []string{}
		for k := range ctd.AccountDataHash {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := ctd.AccountDataHash[k]
			buffer.WriteString(string(k))
			buffer.WriteString(": ")
			buffer.WriteString(string(v))
			buffer.WriteString("\n")
		}
	}
	buffer.WriteString("\n")
	buffer.WriteString("DeletedAccountDataHash\n")
	{
		keys := []string{}
		for k := range ctd.DeletedAccountDataHash {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			buffer.WriteString(string(k))
			buffer.WriteString("\n")
		}
	}
	buffer.WriteString("\n")
	buffer.WriteString("UTXOHash\n")
	{
		keys := []uint64{}
		for k := range ctd.UTXOHash {
			keys = append(keys, k)
		}
		sort.Sort(uint64Slice(keys))
		for _, k := range keys {
			v := ctd.UTXOHash[k]
			buffer.WriteString(strconv.FormatInt(int64(k), 10))
			buffer.WriteString(": ")
			var hb bytes.Buffer
			if _, err := v.WriteTo(&hb); err != nil {
				panic(err)
			}
			buffer.WriteString(hash.Hash(hb.Bytes()).String())
			buffer.WriteString("\n")
		}
	}
	buffer.WriteString("\n")
	buffer.WriteString("CreatedUTXOHash\n")
	{
		keys := []uint64{}
		for k := range ctd.CreatedUTXOHash {
			keys = append(keys, k)
		}
		sort.Sort(uint64Slice(keys))
		for _, k := range keys {
			v := ctd.CreatedUTXOHash[k]
			buffer.WriteString(strconv.FormatInt(int64(k), 10))
			buffer.WriteString(": ")
			var hb bytes.Buffer
			if _, err := v.WriteTo(&hb); err != nil {
				panic(err)
			}
			buffer.WriteString(hash.Hash(hb.Bytes()).String())
			buffer.WriteString("\n")
		}
	}
	buffer.WriteString("\n")
	buffer.WriteString("DeletedUTXOHash\n")
	{
		keys := []uint64{}
		for k := range ctd.DeletedUTXOHash {
			keys = append(keys, k)
		}
		sort.Sort(uint64Slice(keys))
		for _, k := range keys {
			buffer.WriteString(strconv.FormatInt(int64(k), 10))
			buffer.WriteString("\n")
		}
	}
	return buffer.String()
}

type addressSlice []common.Address

func (p addressSlice) Len() int           { return len(p) }
func (p addressSlice) Less(i, j int) bool { return bytes.Compare(p[i][:], p[j][:]) < 0 }
func (p addressSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

type uint64Slice []uint64

func (p uint64Slice) Len() int           { return len(p) }
func (p uint64Slice) Less(i, j int) bool { return p[i] < p[j] }
func (p uint64Slice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
