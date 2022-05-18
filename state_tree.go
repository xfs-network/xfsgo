// Copyright 2018 The xfsgo Authors
// This file is part of the xfsgo library.
//
// The xfsgo library is free software: you can redistribute it and/or modify
// it under the terms of the MIT Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The xfsgo library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// MIT Lesser General Public License for more details.
//
// You should have received a copy of the MIT Lesser General Public License
// along with the xfsgo library. If not, see <https://mit-license.org/>.

package xfsgo

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"xfsgo/avlmerkle"
	"xfsgo/common"
	"xfsgo/common/ahash"
	"xfsgo/common/rawencode"
	"xfsgo/storage/badger"
)

//StateObj is an importment type which represents an xfs account that is being modified.
// The flow of usage is as follows:
// First, you need to obtain a StateObj object.
// Second, access and modify the balance of account through the object.
// Finally, call Commit method to write the modified merkleTree into a database.
type StateObj struct {
	merkleTree   *avlmerkle.Tree
	address      common.Address //hash of address of the account
	balance      *big.Int
	nonce        uint64
	extra        []byte
	code         []byte
	stateRoot    common.Hash
	cacheStorage map[[32]byte][]byte
	db           badger.IStorage
    mTree *avlmerkle.Tree
}

func loadBytesByMapKey(m map[string]string, key string) (data []byte, rt bool) {
	var str string
	var err error
	if str, rt = m[key]; rt {
		if data, err = hex.DecodeString(str); err != nil {
			rt = false
		}
	}
	return
}
func (so *StateObj) Decode(data []byte) error {
	r := common.StringDecodeMap(string(data))
	if r == nil {
		return nil
	}
	if address, ok := r["address"]; ok {
		so.address = common.StrB58ToAddress(address)
	}
	if balance, ok := r["balance"]; ok {
		if num, ok := new(big.Int).
			SetString(balance, 10); ok {
			so.balance = num
		}
	}
	if nonce, ok := r["nonce"]; ok {
		if num, ok := new(big.Int).
			SetString(nonce, 10); ok {
			so.nonce = num.Uint64()
		}
	}
	if bs, ok := loadBytesByMapKey(r, "extra"); ok {
		so.extra = bs
	}
	if bs, ok := loadBytesByMapKey(r, "code"); ok {
		so.code = bs
	}
	if bs, ok := loadBytesByMapKey(r, "state_root"); ok {
		so.stateRoot = common.Bytes2Hash(bs)
	}
	return nil
}

func (so *StateObj) Encode() ([]byte, error) {
	objmap := map[string]string{
		"address": so.address.String(),
		"balance": so.balance.Text(10),
		"nonce":   new(big.Int).SetUint64(so.nonce).Text(10),
		"extra":   hex.EncodeToString(so.extra),
	}
	if so.code != nil {
		objmap["code"] = hex.EncodeToString(so.code)
	}
	if !bytes.Equal(so.stateRoot[:], common.HashZ[:]) {
		objmap["state_root"] = hex.EncodeToString(so.stateRoot[:])
	}
	enc := common.SortAndEncodeMap(objmap)
	return []byte(enc), nil
}

func NewStateObj(address common.Address, tree *avlmerkle.Tree, db badger.IStorage) *StateObj {
	obj := &StateObj{
		address:      address,
		merkleTree:   tree,
		db:           db,
		cacheStorage: make(map[[32]byte][]byte),
	}
	return obj
}

// AddBalance adds amount to StateObj's balance.
// It is used to add funds to the destination account of a transfer.
func (so *StateObj) AddBalance(val *big.Int) {
	if val == nil || val.Sign() <= 0 {
		return
	}
	oldBalance := so.balance
	if oldBalance == nil {
		oldBalance = zeroBigN
	}
	newBalance := new(big.Int).Add(oldBalance, val)
	so.SetBalance(newBalance)
}

// SubBalance removes amount from StateObj's balance.
// It is used to remove funds from the origin account of a transfer.
func (so *StateObj) SubBalance(val *big.Int) {
	if val == nil || val.Sign() <= 0 {
		return
	}
	oldBalance := so.balance
	if oldBalance == nil {
		oldBalance = zeroBigN
	}
	newBalance := oldBalance.Sub(oldBalance, val)
	so.SetBalance(newBalance)
}

func (so *StateObj) SetBalance(val *big.Int) {
	if val == nil || val.Sign() < 0 {
		return
	}
	so.balance = val
}

func (so *StateObj) GetBalance() *big.Int {
	return so.balance
}

func (so *StateObj) GetAddress() common.Address {
	return so.address
}

func (so *StateObj) SetNonce(nonce uint64) {
	so.nonce = nonce
}
func (so *StateObj) AddNonce(nonce uint64) {
	so.nonce += nonce
}
func (so *StateObj) SubNonce(nonce uint64) {
	so.nonce -= nonce
}
func (so *StateObj) GetNonce() uint64 {
	return so.nonce
}
func (so *StateObj) GetExtra() []byte {
	return so.extra
}

func (so *StateObj) SetCode(code []byte) {
	so.code = code
}
func (so *StateObj) SetState(key [32]byte, value []byte) {
	so.cacheStorage[key] = value
}
func (so *StateObj) GetCode() []byte {
	return so.code
}
func (so *StateObj) getStateTree() *avlmerkle.Tree {
    if so.mTree == nil {
        so.mTree = avlmerkle.NewTree(so.db, so.stateRoot[:])
    }
    return so.mTree
}

func (so *StateObj) GetStateValue(key [32]byte) []byte {
	if val, exists := so.cacheStorage[key]; exists {
		return val
	}
    relkey := common.MakeStateKey(so.address, key[:])
    stateTree := so.getStateTree()
    if stateTree == nil {
        return nil
    }
    if val, ok := stateTree.Get(relkey); ok {
        return val
    }
	return nil
}

func (so *StateObj) GetStateRoot() common.Hash {
	return so.stateRoot
}

func (so *StateObj) Update() {
    st := so.getStateTree()
	for k, v := range so.cacheStorage {
        relkey := common.MakeStateKey(so.address, k[:])
		st.Put(relkey, v) 
	}
	stateRoot := st.Checksum()
	so.stateRoot = common.Bytes2Hash(stateRoot)
	objRaw, _ := rawencode.Encode(so)
	hash := ahash.SHA256(so.address[:])
	so.merkleTree.Put(hash, objRaw)

}

type StateTree struct {
	root       []byte
	treeDB     badger.IStorage
	merkleTree *avlmerkle.Tree
	objs       map[common.Address]*StateObj
}

func NewStateTree(db badger.IStorage, root []byte) *StateTree {
	st := &StateTree{
		root:   root,
		treeDB: db,
		objs:   make(map[common.Address]*StateObj),
	}
	st.merkleTree = avlmerkle.NewTree(st.treeDB, root)
	return st
}

func NewStateTreeN(db badger.IStorage, root []byte) (*StateTree, error) {
	var err error
	st := &StateTree{
		root:   root,
		treeDB: db,
		objs:   make(map[common.Address]*StateObj),
	}
	st.merkleTree, err = avlmerkle.NewTreeN(st.treeDB, root)
	return st, err
}

func (st *StateTree) HashAccount(addr common.Address) bool {
	return st.GetStateObj(addr) != nil
}

func (st *StateTree) GetBalance(addr common.Address) *big.Int {
	obj := st.GetStateObj(addr)
	if obj != nil {
		if obj.balance == nil {
			return zeroBigN
		}
		return obj.balance
	}
	return zeroBigN
}

func (st *StateTree) Copy() *StateTree {
	cpy := new(StateTree)
	copy(cpy.root, st.root)
	cpy.treeDB = st.treeDB
	cpy.merkleTree = st.merkleTree.Copy()
	cpy.objs = make(map[common.Address]*StateObj)
	for k, v := range st.objs {
		cpy.objs[k] = v
	}
	return cpy
}
func (st *StateTree) Set(snap *StateTree) *StateTree {
	st.root = snap.root
	st.treeDB = snap.treeDB
	st.merkleTree = snap.merkleTree
	st.objs = snap.objs
	return st
}

func (st *StateTree) AddBalance(addr common.Address, val *big.Int) {
	obj := st.GetOrNewStateObj(addr)
	if obj != nil {
		obj.AddBalance(val)
	}
}
func (st *StateTree) GetNonce(addr common.Address) uint64 {
	obj := st.GetStateObj(addr)
	if obj != nil {
		return obj.nonce
	}
	return 0
}

func (st *StateTree) AddNonce(addr common.Address, val uint64) {
	obj := st.GetOrNewStateObj(addr)
	if obj != nil {
		obj.AddNonce(val)
	}
}

func (st *StateTree) GetStateObj(addr common.Address) *StateObj {
	if st.objs[addr] != nil {
		return st.objs[addr]
	}
	hash := ahash.SHA256(addr.Bytes())
	if val, has := st.merkleTree.Get(hash); has {
		obj := &StateObj{}
        obj.db = st.treeDB
		if err := rawencode.Decode(val, obj); err != nil {
			return nil
		}
		obj.merkleTree = st.merkleTree
		st.objs[addr] = obj
        
		return obj
	}
	return nil
}

func (st *StateTree) newStateObj(address common.Address) *StateObj {
	obj := NewStateObj(address, st.merkleTree, st.treeDB)
	st.objs[obj.address] = obj
	return obj
}
func (st *StateTree) CreateAccount(addr common.Address) *StateObj {
	old := st.GetStateObj(addr)
	add := st.newStateObj(addr)
	if old != nil {
		add.balance = old.balance
	}
	return add
}

func (st *StateTree) GetOrNewStateObj(addr common.Address) *StateObj {
	stateObj := st.GetStateObj(addr)
	if stateObj == nil {
		stateObj = st.CreateAccount(addr)
	}
	return stateObj
}
func (st *StateTree) SetState(addr common.Address, key [32]byte, value []byte) {
	obj := st.GetOrNewStateObj(addr)
	if obj != nil {
		obj.SetState(key, value)
	}
}
func (st *StateTree) SetCode(addr common.Address, code []byte) {
	obj := st.GetOrNewStateObj(addr)
	if obj != nil {
		obj.SetCode(code)
	}
}
func (st *StateTree) GetCode(addr common.Address) []byte {
	obj := st.GetOrNewStateObj(addr)
	if obj != nil {
		return obj.GetCode()
	}
	return nil
}

func (st *StateTree) GetStateValue(addr common.Address, key [32]byte) []byte {
	obj := st.GetOrNewStateObj(addr)
	if obj != nil {
		return obj.GetStateValue(key)
	}
	return nil
}
func (st *StateTree) Root() []byte {
	return st.merkleTree.Checksum()
}

func (st *StateTree) RootHex() string {
	return st.merkleTree.ChecksumHex()
}

func (st *StateTree) UpdateAll() {
	for _, v := range st.objs {
		v.Update()
	}
}

func (st *StateTree) Commit() error {
    for _, v := range st.objs {
        if v.mTree != nil {
            err := v.mTree.Commit()
            if err != nil {
                return err
            }
        }
    }
	return st.merkleTree.Commit()
}
