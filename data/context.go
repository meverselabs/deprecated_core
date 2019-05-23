package data

import (
	"bytes"
	"sort"
	"strconv"

	"github.com/fletaio/common"
	"github.com/fletaio/common/hash"
	"github.com/fletaio/common/util"
	"github.com/fletaio/core/account"
	"github.com/fletaio/core/event"
	"github.com/fletaio/core/transaction"
)

// Context is an intermediate in-memory state using the context data stack between blocks
type Context struct {
	loader          Loader
	genTargetHeight uint32
	genLastHash     hash.Hash256
	cache           *cache
	stack           []*ContextData
	isLatestHash    bool
	dataHash        hash.Hash256
}

// NewContext returns a Context
func NewContext(loader Loader) *Context {
	ctx := &Context{
		loader:          loader,
		genTargetHeight: loader.TargetHeight(),
		genLastHash:     loader.LastHash(),
		stack:           []*ContextData{NewContextData(loader, nil)},
	}
	ctx.cache = newCache(ctx)
	return ctx
}

// NextContext returns the next Context of the Context
func (ctx *Context) NextContext(NextHash hash.Hash256) *Context {
	nctx := NewContext(ctx)
	nctx.genTargetHeight = ctx.genTargetHeight + 1
	nctx.genLastHash = NextHash
	return nctx
}

// Hash returns the hash value of it
func (ctx *Context) Hash() hash.Hash256 {
	if !ctx.isLatestHash {
		ctx.dataHash = ctx.Top().Hash()
		ctx.isLatestHash = true
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

// Eventer returns the eventer of the target chain
func (ctx *Context) Eventer() *Eventer {
	return ctx.loader.Eventer()
}

// TargetHeight returns the recorded target height when context generation
func (ctx *Context) TargetHeight() uint32 {
	return ctx.genTargetHeight
}

// LastHash returns the recorded prev hash when context generation
func (ctx *Context) LastHash() hash.Hash256 {
	return ctx.genLastHash
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
	ctx.isLatestHash = false
	ctx.Top().AddSeq(addr)
}

// Account returns the account instance of the address
func (ctx *Context) Account(addr common.Address) (account.Account, error) {
	ctx.isLatestHash = false
	return ctx.Top().Account(addr)
}

// AddressByName returns the account address of the name
func (ctx *Context) AddressByName(Name string) (common.Address, error) {
	return ctx.Top().AddressByName(Name)
}

// IsExistAccount checks that the account of the address is exist or not
func (ctx *Context) IsExistAccount(addr common.Address) (bool, error) {
	return ctx.Top().IsExistAccount(addr)
}

// IsExistAccountName checks that the account of the name is exist or not
func (ctx *Context) IsExistAccountName(Name string) (bool, error) {
	return ctx.Top().IsExistAccountName(Name)
}

// CreateAccount inserts the account to the top snapshot
func (ctx *Context) CreateAccount(acc account.Account) error {
	ctx.isLatestHash = false
	return ctx.Top().CreateAccount(acc)
}

// DeleteAccount deletes the account from the top snapshot
func (ctx *Context) DeleteAccount(acc account.Account) error {
	ctx.isLatestHash = false
	return ctx.Top().DeleteAccount(acc)
}

// AccountData returns the account data from the top snapshot
func (ctx *Context) AccountData(addr common.Address, name []byte) []byte {
	return ctx.Top().AccountData(addr, name)
}

// SetAccountData inserts the account data to the top snapshot
func (ctx *Context) SetAccountData(addr common.Address, name []byte, value []byte) {
	ctx.isLatestHash = false
	ctx.Top().SetAccountData(addr, name, value)
}

// IsExistUTXO checks that the utxo of the id is exist or not
func (ctx *Context) IsExistUTXO(id uint64) (bool, error) {
	return ctx.Top().IsExistUTXO(id)
}

// UTXO returns the UTXO from the top snapshot
func (ctx *Context) UTXO(id uint64) (*transaction.UTXO, error) {
	return ctx.Top().UTXO(id)
}

// CreateUTXO inserts the UTXO to the top snapshot
func (ctx *Context) CreateUTXO(id uint64, vout *transaction.TxOut) error {
	ctx.isLatestHash = false
	return ctx.Top().CreateUTXO(id, vout)
}

// DeleteUTXO deletes the UTXO from the top snapshot
func (ctx *Context) DeleteUTXO(id uint64) error {
	ctx.isLatestHash = false
	return ctx.Top().DeleteUTXO(id)
}

// EmitEvent creates the event to the top snapshot
func (ctx *Context) EmitEvent(e event.Event) error {
	ctx.isLatestHash = false
	return ctx.Top().EmitEvent(e)
}

// Dump prints the top context data of the context
func (ctx *Context) Dump() string {
	return ctx.Top().Dump()
}

// Snapshot push a snapshot and returns the snapshot number of it
func (ctx *Context) Snapshot() int {
	ctx.isLatestHash = false
	ctd := NewContextData(ctx.cache, ctx.Top())
	ctx.stack[len(ctx.stack)-1].isTop = false
	ctx.stack = append(ctx.stack, ctd)
	return len(ctx.stack)
}

// Revert removes snapshots after the snapshot number
func (ctx *Context) Revert(sn int) {
	ctx.isLatestHash = false
	if len(ctx.stack) >= sn {
		ctx.stack = ctx.stack[:sn-1]
	}
	ctx.stack[len(ctx.stack)-1].isTop = true
}

// Commit apply snapshots to the top after the snapshot number
func (ctx *Context) Commit(sn int) {
	ctx.isLatestHash = false
	for len(ctx.stack) >= sn {
		ctd := ctx.Top()
		ctx.stack = ctx.stack[:len(ctx.stack)-1]
		top := ctx.Top()
		for k, v := range ctd.SeqMap {
			top.SeqMap[k] = v
		}
		for k, v := range ctd.AccountMap {
			top.AccountMap[k] = v
		}
		for k, v := range ctd.CreatedAccountMap {
			top.CreatedAccountMap[k] = v
		}
		for k, v := range ctd.DeletedAccountMap {
			delete(top.AccountMap, k)
			delete(top.CreatedAccountMap, k)
			top.DeletedAccountMap[k] = v
		}
		for k, v := range ctd.AccountDataMap {
			top.AccountDataMap[k] = v
		}
		for k, v := range ctd.DeletedAccountDataMap {
			delete(top.AccountDataMap, k)
			top.DeletedAccountDataMap[k] = v
		}
		for k, v := range ctd.UTXOMap {
			top.UTXOMap[k] = v
		}
		for k, v := range ctd.CreatedUTXOMap {
			top.CreatedUTXOMap[k] = v
		}
		for k, v := range ctd.DeletedUTXOMap {
			delete(top.UTXOMap, k)
			delete(top.CreatedUTXOMap, k)
			top.DeletedUTXOMap[k] = v
		}
		for _, v := range ctd.Events {
			top.Events = append(top.Events, v)
		}
		top.EventIndex = ctd.EventIndex
	}
}

// StackSize returns the size of the context data stack
func (ctx *Context) StackSize() int {
	return len(ctx.stack)
}

// ContextData is a state data of the context
type ContextData struct {
	loader                Loader
	Parent                *ContextData
	SeqMap                map[common.Address]uint64
	AccountMap            map[common.Address]account.Account
	CreatedAccountMap     map[common.Address]account.Account
	DeletedAccountMap     map[common.Address]account.Account
	AccountNameMap        map[string]common.Address
	CreatedAccountNameMap map[string]common.Address
	DeletedAccountNameMap map[string]common.Address
	AccountDataMap        map[string][]byte
	DeletedAccountDataMap map[string]bool
	UTXOMap               map[uint64]*transaction.UTXO
	CreatedUTXOMap        map[uint64]*transaction.TxOut
	DeletedUTXOMap        map[uint64]bool
	Events                []event.Event
	EventIndex            uint16
	isTop                 bool
}

// NewContextData returns a ContextData
func NewContextData(loader Loader, Parent *ContextData) *ContextData {
	var eventIndex uint16
	if Parent != nil {
		eventIndex = Parent.EventIndex
	}
	ctd := &ContextData{
		loader:                loader,
		Parent:                Parent,
		SeqMap:                map[common.Address]uint64{},
		AccountMap:            map[common.Address]account.Account{},
		CreatedAccountMap:     map[common.Address]account.Account{},
		DeletedAccountMap:     map[common.Address]account.Account{},
		AccountNameMap:        map[string]common.Address{},
		CreatedAccountNameMap: map[string]common.Address{},
		DeletedAccountNameMap: map[string]common.Address{},
		AccountDataMap:        map[string][]byte{},
		DeletedAccountDataMap: map[string]bool{},
		UTXOMap:               map[uint64]*transaction.UTXO{},
		CreatedUTXOMap:        map[uint64]*transaction.TxOut{},
		DeletedUTXOMap:        map[uint64]bool{},
		Events:                []event.Event{},
		EventIndex:            eventIndex,
		isTop:                 true,
	}
	return ctd
}

// Hash returns the hash value of it
func (ctd *ContextData) Hash() hash.Hash256 {
	var buffer bytes.Buffer

	buffer.WriteString("ChainCoord")
	if _, err := ctd.loader.ChainCoord().WriteTo(&buffer); err != nil {
		panic(err)
	}
	buffer.WriteString("SeqMap")
	if len(ctd.SeqMap) > 0 {
		keys := []common.Address{}
		for k := range ctd.SeqMap {
			keys = append(keys, k)
		}
		sort.Sort(addressSlice(keys))
		for _, k := range keys {
			v := ctd.SeqMap[k]
			if _, err := k.WriteTo(&buffer); err != nil {
				panic(err)
			}
			if _, err := util.WriteUint64(&buffer, v); err != nil {
				panic(err)
			}
		}
	}
	buffer.WriteString("AccountMap")
	if len(ctd.AccountMap) > 0 {
		keys := []common.Address{}
		for k := range ctd.AccountMap {
			keys = append(keys, k)
		}
		sort.Sort(addressSlice(keys))
		for _, k := range keys {
			v := ctd.AccountMap[k]
			if _, err := k.WriteTo(&buffer); err != nil {
				panic(err)
			}
			if _, err := v.WriteTo(&buffer); err != nil {
				panic(err)
			}
		}
	}
	buffer.WriteString("CreatedAccountMap")
	if len(ctd.CreatedAccountMap) > 0 {
		keys := []common.Address{}
		for k := range ctd.CreatedAccountMap {
			keys = append(keys, k)
		}
		sort.Sort(addressSlice(keys))
		for _, k := range keys {
			v := ctd.CreatedAccountMap[k]
			if _, err := k.WriteTo(&buffer); err != nil {
				panic(err)
			}
			if _, err := v.WriteTo(&buffer); err != nil {
				panic(err)
			}
		}
	}
	buffer.WriteString("DeletedAccountMap")
	if len(ctd.DeletedAccountMap) > 0 {
		keys := []common.Address{}
		for k := range ctd.DeletedAccountMap {
			keys = append(keys, k)
		}
		sort.Sort(addressSlice(keys))
		for _, k := range keys {
			if _, err := k.WriteTo(&buffer); err != nil {
				panic(err)
			}
		}
	}
	buffer.WriteString("AccountNameMap")
	if len(ctd.AccountNameMap) > 0 {
		keys := []string{}
		for k := range ctd.AccountNameMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := ctd.AccountNameMap[k]
			if _, err := buffer.WriteString(k); err != nil {
				panic(err)
			}
			if _, err := v.WriteTo(&buffer); err != nil {
				panic(err)
			}
		}
	}
	buffer.WriteString("CreatedAccountNameMap")
	if len(ctd.CreatedAccountNameMap) > 0 {
		keys := []string{}
		for k := range ctd.CreatedAccountNameMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := ctd.CreatedAccountNameMap[k]
			if _, err := buffer.WriteString(k); err != nil {
				panic(err)
			}
			if _, err := v.WriteTo(&buffer); err != nil {
				panic(err)
			}
		}
	}
	buffer.WriteString("DeletedAccountNameMap")
	if len(ctd.DeletedAccountNameMap) > 0 {
		keys := []string{}
		for k := range ctd.DeletedAccountNameMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if _, err := buffer.WriteString(k); err != nil {
				panic(err)
			}
		}
	}
	buffer.WriteString("AccountDataMap")
	if len(ctd.AccountDataMap) > 0 {
		keys := []string{}
		for k := range ctd.AccountDataMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := ctd.AccountDataMap[k]
			buffer.WriteString(k)
			buffer.Write(v)
		}
	}
	buffer.WriteString("DeletedAccountDataMap")
	if len(ctd.DeletedAccountDataMap) > 0 {
		keys := []string{}
		for k := range ctd.DeletedAccountDataMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			buffer.WriteString(k)
		}
	}
	buffer.WriteString("UTXOMap")
	if len(ctd.UTXOMap) > 0 {
		keys := []uint64{}
		for k := range ctd.UTXOMap {
			keys = append(keys, k)
		}
		sort.Sort(uint64Slice(keys))
		for _, k := range keys {
			v := ctd.UTXOMap[k]
			if _, err := util.WriteUint64(&buffer, k); err != nil {
				panic(err)
			}
			if _, err := v.WriteTo(&buffer); err != nil {
				panic(err)
			}
		}
	}
	buffer.WriteString("CreatedUTXOMap")
	if len(ctd.CreatedUTXOMap) > 0 {
		keys := []uint64{}
		for k := range ctd.CreatedUTXOMap {
			keys = append(keys, k)
		}
		sort.Sort(uint64Slice(keys))
		for _, k := range keys {
			v := ctd.CreatedUTXOMap[k]
			if _, err := util.WriteUint64(&buffer, k); err != nil {
				panic(err)
			}
			if _, err := v.WriteTo(&buffer); err != nil {
				panic(err)
			}
		}
	}
	buffer.WriteString("DeletedUTXOMap")
	if len(ctd.DeletedUTXOMap) > 0 {
		keys := []uint64{}
		for k := range ctd.DeletedUTXOMap {
			keys = append(keys, k)
		}
		sort.Sort(uint64Slice(keys))
		for _, k := range keys {
			if _, err := util.WriteUint64(&buffer, k); err != nil {
				panic(err)
			}
		}
	}
	buffer.WriteString("Events")
	if len(ctd.Events) > 0 {
		for _, e := range ctd.Events {
			if _, err := e.WriteTo(&buffer); err != nil {
				panic(err)
			}
		}
	}
	return hash.DoubleHash(buffer.Bytes())
}

// Seq returns the sequence of the account
func (ctd *ContextData) Seq(addr common.Address) uint64 {
	if _, has := ctd.DeletedAccountMap[addr]; has {
		return 0
	}
	if seq, has := ctd.SeqMap[addr]; has {
		return seq
	} else if ctd.Parent != nil {
		seq := ctd.Parent.Seq(addr)
		if seq > 0 && ctd.isTop {
			ctd.SeqMap[addr] = seq
		}
		return seq
	} else {
		seq := ctd.loader.Seq(addr)
		if seq > 0 && ctd.isTop {
			ctd.SeqMap[addr] = seq
		}
		return seq
	}
}

// AddSeq update the sequence of the target account
func (ctd *ContextData) AddSeq(addr common.Address) {
	if _, has := ctd.DeletedAccountMap[addr]; has {
		return
	}
	ctd.SeqMap[addr] = ctd.Seq(addr) + 1
}

// Account returns the account instance of the address
func (ctd *ContextData) Account(addr common.Address) (account.Account, error) {
	if _, has := ctd.DeletedAccountMap[addr]; has {
		return nil, ErrNotExistAccount
	}
	if acc, has := ctd.AccountMap[addr]; has {
		return acc, nil
	} else if acc, has := ctd.CreatedAccountMap[addr]; has {
		return acc, nil
	} else if ctd.Parent != nil {
		if acc, err := ctd.Parent.Account(addr); err != nil {
			return nil, err
		} else {
			if ctd.isTop {
				nacc := acc.Clone()
				ctd.AccountMap[addr] = nacc
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
				ctd.AccountMap[addr] = nacc
				return nacc, nil
			} else {
				return acc, nil
			}
		}
	}
}

// AddressByName returns the account address of the name
func (ctd *ContextData) AddressByName(Name string) (common.Address, error) {
	if _, has := ctd.DeletedAccountNameMap[Name]; has {
		return common.Address{}, ErrNotExistAccount
	}
	if addr, has := ctd.AccountNameMap[Name]; has {
		return addr, nil
	} else if addr, has := ctd.CreatedAccountNameMap[Name]; has {
		return addr, nil
	} else if ctd.Parent != nil {
		if addr, err := ctd.Parent.AddressByName(Name); err != nil {
			return common.Address{}, err
		} else {
			if ctd.isTop {
				naddr := addr.Clone()
				ctd.AccountNameMap[Name] = naddr
				return naddr, nil
			} else {
				return addr, nil
			}
		}
	} else {
		if addr, err := ctd.loader.AddressByName(Name); err != nil {
			return common.Address{}, err
		} else {
			if ctd.isTop {
				naddr := addr.Clone()
				ctd.AccountNameMap[Name] = naddr
				return naddr, nil
			} else {
				return addr, nil
			}
		}
	}
}

// IsExistAccount checks that the account of the address is exist or not
func (ctd *ContextData) IsExistAccount(addr common.Address) (bool, error) {
	if _, has := ctd.DeletedAccountMap[addr]; has {
		return false, nil
	}
	if _, has := ctd.AccountMap[addr]; has {
		return true, nil
	} else if _, has := ctd.CreatedAccountMap[addr]; has {
		return true, nil
	} else if ctd.Parent != nil {
		return ctd.Parent.IsExistAccount(addr)
	} else {
		return ctd.loader.IsExistAccount(addr)
	}
}

// IsExistAccountName checks that the account of the address is exist or not
func (ctd *ContextData) IsExistAccountName(Name string) (bool, error) {
	if _, has := ctd.DeletedAccountNameMap[Name]; has {
		return false, nil
	}
	if _, has := ctd.AccountNameMap[Name]; has {
		return true, nil
	} else if _, has := ctd.CreatedAccountNameMap[Name]; has {
		return true, nil
	} else if ctd.Parent != nil {
		return ctd.Parent.IsExistAccountName(Name)
	} else {
		return ctd.loader.IsExistAccountName(Name)
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
	if _, err := ctd.AddressByName(acc.Name()); err != nil {
		if err != ErrNotExistAccount {
			return err
		}
	} else {
		return ErrExistAccount
	}
	ctd.CreatedAccountMap[acc.Address()] = acc
	ctd.CreatedAccountNameMap[acc.Name()] = acc.Address()
	return nil
}

// DeleteAccount deletes the account
func (ctd *ContextData) DeleteAccount(acc account.Account) error {
	if _, err := ctd.Account(acc.Address()); err != nil {
		return err
	}
	ctd.DeletedAccountMap[acc.Address()] = acc
	ctd.DeletedAccountNameMap[acc.Name()] = acc.Address()
	delete(ctd.AccountMap, acc.Address())
	delete(ctd.AccountNameMap, acc.Name())
	return nil
}

// AccountData returns the account data
func (ctd *ContextData) AccountData(addr common.Address, name []byte) []byte {
	key := string(addr[:]) + string(name)
	if ctd.DeletedAccountDataMap[key] {
		return nil
	}
	if value, has := ctd.AccountDataMap[key]; has {
		return value
	} else if ctd.Parent != nil {
		value := ctd.Parent.AccountData(addr, name)
		if len(value) > 0 {
			if ctd.isTop {
				nvalue := make([]byte, len(value))
				copy(nvalue, value)
				ctd.AccountDataMap[key] = nvalue
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
				ctd.AccountDataMap[key] = nvalue
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
		delete(ctd.AccountDataMap, key)
		ctd.DeletedAccountDataMap[key] = true
	} else {
		delete(ctd.DeletedAccountDataMap, key)
		ctd.AccountDataMap[key] = value
	}
}

// IsExistUTXO checks that the utxo of the id is exist or not
func (ctd *ContextData) IsExistUTXO(id uint64) (bool, error) {
	if _, has := ctd.DeletedUTXOMap[id]; has {
		return false, nil
	}
	if _, has := ctd.UTXOMap[id]; has {
		return true, nil
	} else if _, has := ctd.CreatedUTXOMap[id]; has {
		return true, nil
	} else if ctd.Parent != nil {
		return ctd.Parent.IsExistUTXO(id)
	} else {
		return ctd.loader.IsExistUTXO(id)
	}
}

// UTXO returns the UTXO
func (ctd *ContextData) UTXO(id uint64) (*transaction.UTXO, error) {
	if ctd.DeletedUTXOMap[id] {
		return nil, ErrDoubleSpent
	}
	if utxo, has := ctd.UTXOMap[id]; has {
		return utxo, nil
	} else if ctd.Parent != nil {
		if utxo, err := ctd.Parent.UTXO(id); err != nil {
			return nil, err
		} else {
			if ctd.isTop {
				nutxo := utxo.Clone()
				ctd.UTXOMap[id] = nutxo
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
				ctd.UTXOMap[id] = nutxo
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
	ctd.CreatedUTXOMap[id] = vout
	return nil
}

// DeleteUTXO deletes the UTXO
func (ctd *ContextData) DeleteUTXO(id uint64) error {
	if _, err := ctd.UTXO(id); err != nil {
		return err
	}
	ctd.DeletedUTXOMap[id] = true
	return nil
}

// EmitEvent creates the event to the top snapshot
func (ctd *ContextData) EmitEvent(e event.Event) error {
	e.SetIndex(ctd.EventIndex)
	ctd.EventIndex++
	ctd.Events = append(ctd.Events, e)
	return nil
}

// Dump prints the context data
func (ctd *ContextData) Dump() string {
	var buffer bytes.Buffer
	buffer.WriteString("SeqMap\n")
	{
		keys := []common.Address{}
		for k := range ctd.SeqMap {
			keys = append(keys, k)
		}
		sort.Sort(addressSlice(keys))
		for _, k := range keys {
			v := ctd.SeqMap[k]
			buffer.WriteString(k.String())
			buffer.WriteString(": ")
			buffer.WriteString(strconv.FormatInt(int64(v), 10))
			buffer.WriteString("\n")
		}
	}
	buffer.WriteString("\n")
	buffer.WriteString("AccountMap\n")
	{
		keys := []common.Address{}
		for k := range ctd.AccountMap {
			keys = append(keys, k)
		}
		sort.Sort(addressSlice(keys))
		for _, k := range keys {
			v := ctd.AccountMap[k]
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
	buffer.WriteString("CreatedAccountMap\n")
	{
		keys := []common.Address{}
		for k := range ctd.CreatedAccountMap {
			keys = append(keys, k)
		}
		sort.Sort(addressSlice(keys))
		for _, k := range keys {
			v := ctd.CreatedAccountMap[k]
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
	buffer.WriteString("DeletedAccountMap\n")
	{
		keys := []common.Address{}
		for k := range ctd.DeletedAccountMap {
			keys = append(keys, k)
		}
		sort.Sort(addressSlice(keys))
		for _, k := range keys {
			buffer.WriteString(k.String())
			buffer.WriteString("\n")
		}
	}
	buffer.WriteString("\n")
	buffer.WriteString("AccountNameMap\n")
	{
		keys := []string{}
		for k := range ctd.AccountNameMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := ctd.AccountNameMap[k]
			buffer.WriteString(k)
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
	buffer.WriteString("CreatedAccountNameMap\n")
	{
		keys := []string{}
		for k := range ctd.CreatedAccountNameMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := ctd.CreatedAccountNameMap[k]
			buffer.WriteString(k)
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
	buffer.WriteString("DeletedAccountNameMap\n")
	{
		keys := []string{}
		for k := range ctd.DeletedAccountNameMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			buffer.WriteString(k)
			buffer.WriteString("\n")
		}
	}
	buffer.WriteString("\n")
	buffer.WriteString("AccountDataMap\n")
	{
		keys := []string{}
		for k := range ctd.AccountDataMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := ctd.AccountDataMap[k]
			buffer.WriteString(hash.Hash([]byte(k)).String())
			buffer.WriteString(": ")
			buffer.WriteString(hash.Hash([]byte(v)).String())
			buffer.WriteString("\n")
		}
	}
	buffer.WriteString("\n")
	buffer.WriteString("DeletedAccountDataMap\n")
	{
		keys := []string{}
		for k := range ctd.DeletedAccountDataMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			buffer.WriteString(string(k))
			buffer.WriteString("\n")
		}
	}
	buffer.WriteString("\n")
	buffer.WriteString("UTXOMap\n")
	{
		keys := []uint64{}
		for k := range ctd.UTXOMap {
			keys = append(keys, k)
		}
		sort.Sort(uint64Slice(keys))
		for _, k := range keys {
			v := ctd.UTXOMap[k]
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
	buffer.WriteString("CreatedUTXOMap\n")
	{
		keys := []uint64{}
		for k := range ctd.CreatedUTXOMap {
			keys = append(keys, k)
		}
		sort.Sort(uint64Slice(keys))
		for _, k := range keys {
			v := ctd.CreatedUTXOMap[k]
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
	buffer.WriteString("DeletedUTXOMap\n")
	{
		keys := []uint64{}
		for k := range ctd.DeletedUTXOMap {
			keys = append(keys, k)
		}
		sort.Sort(uint64Slice(keys))
		for _, k := range keys {
			buffer.WriteString(strconv.FormatInt(int64(k), 10))
			buffer.WriteString("\n")
		}
	}
	buffer.WriteString("\n")
	buffer.WriteString("Events\n")
	{
		for _, v := range ctd.Events {
			var hb bytes.Buffer
			if _, err := v.WriteTo(&hb); err != nil {
				panic(err)
			}
			buffer.WriteString(hash.Hash(hb.Bytes()).String())
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
