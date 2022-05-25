package vm

import (
	"math/big"
)

type NFToken interface {
	GetName() CTypeString
	GetSymbol() CTypeString
	GetDecimals() CTypeUint8
	BalanceOf(address CTypeAddress) CTypeUint256
}

type nftoken struct {
	BuiltinContract
	Creator         CTypeAddress                                `contract:"storage"`
	Counter         CTypeUint256                                `contract:"storage"`
	Name            CTypeString                                 `contract:"storage"`
	Symbol          CTypeString                                 `contract:"storage"`
	Owners          map[CTypeUint256]CTypeAddress               `contract:"storage"`
	Balances        map[CTypeAddress]CTypeUint256               `contract:"storage"`
	TokenAllowances map[CTypeUint256]CTypeAddress               `contract:"storage"`
	Allowances      map[CTypeAddress]map[CTypeAddress]CTypeBool `contract:"storage"`
}

func (t *nftoken) Create(
	ctx *ContractContext,
	name CTypeString,
	symbol CTypeString,
) error {
	t.Creator = NewAddress(ctx.caller)
	t.Name = name
	t.Symbol = symbol
	t.Balances = make(map[CTypeAddress]CTypeUint256)
	t.Owners = make(map[CTypeUint256]CTypeAddress)
	t.TokenAllowances = make(map[CTypeUint256]CTypeAddress)
	t.Allowances = make(map[CTypeAddress]map[CTypeAddress]CTypeBool)
	return nil
}

func (t *nftoken) BuiltinId() uint8 {
	return 0x02
}

func (t *nftoken) GetName() CTypeString {
	return t.Name
}

func (t *nftoken) GetSymbol() CTypeString {
	return t.Symbol
}
func (t *nftoken) exists(tokenId CTypeUint256) bool {
	return requireAddress(t.Owners[tokenId])
}
func (t *nftoken) Mint(ctx *ContractContext, address CTypeAddress) CTypeUint256 {
	if !assertAddress(NewAddress(ctx.caller), t.Creator) {
		return CTypeUint256{}
	}
	if !requireAddress(address) {
		return CTypeUint256{}
	}
	tokenId := new(big.Int).Add(t.Counter.BigInt(), big.NewInt(1))
	oldBalance := t.Balances[address]
	newBalance := new(big.Int).Add(oldBalance.BigInt(), big.NewInt(1))
	t.Balances[address] = NewUint256(newBalance)
	t.Owners[NewUint256(tokenId)] = address
	t.Counter = NewUint256(tokenId)
	return NewUint256(tokenId)
}
func (t *nftoken) BalanceOf(addr CTypeAddress) CTypeUint256 {
	if !requireAddress(addr) {
		return CTypeUint256{}
	}
	return t.Balances[addr]
}

func (t *nftoken) OwnerOf(tokenId CTypeUint256) CTypeAddress {
	if !requireTokenId(tokenId) {
		return CTypeAddress{}
	}
	addr := t.Owners[tokenId]
	if !requireAddress(addr) {
		return CTypeAddress{}
	}
	return addr
}

func (t *nftoken) isApprovedOrOwner(spender CTypeAddress, tokenId CTypeUint256) bool {
	if !t.exists(tokenId) {
		return false
	}
	owner := t.OwnerOf(tokenId)
	if !assertAddress(spender, owner) {
		if t.IsApprovedForAll(owner, spender) == CBoolFalse {
			approved := t.GetApproved(tokenId)
			if !assertAddress(approved, spender) {
				return false
			}
		}
	}
	return true
}

func (t *nftoken) TransferFrom(ctx *ContractContext, from, to CTypeAddress, tokenId CTypeUint256) CTypeBool {
	if !requireAddress(from) {
		return CBoolFalse
	}
	if !requireAddress(to) {
		return CBoolFalse
	}
	if !requireTokenId(tokenId) {
		return CBoolFalse
	}
	caller := NewAddress(ctx.caller)
	if !t.isApprovedOrOwner(caller, tokenId) {
		return CBoolFalse
	}
	owner := t.OwnerOf(tokenId)
	if !assertAddress(owner, from) {
		return CBoolFalse
	}
	t.approve(CTypeAddress{}, tokenId)
	fromOldBalance := t.Balances[from]
	fromNewBalance := new(big.Int).Sub(fromOldBalance.BigInt(), big.NewInt(1))
	t.Balances[from] = NewUint256(fromNewBalance)
	toOldBalance := t.Balances[from]
	toNewBalance := new(big.Int).Sub(toOldBalance.BigInt(), big.NewInt(1))
	t.Balances[to] = NewUint256(toNewBalance)
	t.Owners[tokenId] = to
	return CBoolTrue
}

func (t *nftoken) approve(to CTypeAddress, tokenId CTypeUint256) {
	t.TokenAllowances[tokenId] = to
}
func (t *nftoken) Approve(ctx *ContractContext, to CTypeAddress, tokenId CTypeUint256) CTypeBool {
	if !requireAddress(to) {
		return CBoolFalse
	}
	if !requireTokenId(tokenId) {
		return CBoolFalse
	}
	owner := t.OwnerOf(tokenId)
	caller := NewAddress(ctx.caller)
	if !assertAddress(caller, owner) {
		if t.IsApprovedForAll(owner, caller) != CBoolTrue {
			return CBoolFalse
		}
	}
	t.approve(to, tokenId)
	return CBoolTrue
}
func (t *nftoken) GetApproved(tokenId CTypeUint256) CTypeAddress {
	if !t.exists(tokenId) {
		return CTypeAddress{}
	}
	return t.TokenAllowances[tokenId]
}

func (t *nftoken) SetApprovalForAll(ctx *ContractContext, operator CTypeAddress, value CTypeBool) CTypeBool {
	if !requireAddress(operator) {
		return CBoolFalse
	}
	owner := NewAddress(ctx.caller)
	if !assertAddress(owner, operator) {
		return CBoolFalse
	}
	if _, exists := t.Allowances[owner][operator]; exists {
		t.Allowances[owner][operator] = value
		return CBoolTrue
	}
	if _, exists := t.Allowances[owner]; exists {
		t.Allowances[owner][operator] = value
		return CBoolTrue
	}
	t.Allowances[owner] = make(map[CTypeAddress]CTypeBool)
	t.Allowances[owner][operator] = value
	return CBoolTrue
}

func (t *nftoken) IsApprovedForAll(owner, spender CTypeAddress) CTypeBool {
	if !requireAddress(owner) {
		return CBoolFalse
	}
	if !requireAddress(spender) {
		return CBoolFalse
	}
	if v, exists := t.Allowances[owner][spender]; exists {
		return v
	}
	return CBoolFalse
}
