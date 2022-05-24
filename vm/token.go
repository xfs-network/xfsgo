package vm

import (
	"bytes"
	"github.com/sirupsen/logrus"
	"math/big"
)

type Token interface {
	GetName() CTypeString
	GetSymbol() CTypeString
	GetDecimals() CTypeUint8
	GetTotalSupply() CTypeUint256
	BalanceOf(address CTypeAddress) CTypeUint256
}

type token struct {
	BuiltinContract
	Name        CTypeString                                    `contract:"storage"`
	Symbol      CTypeString                                    `contract:"storage"`
	Decimals    CTypeUint8                                     `contract:"storage"`
	TotalSupply CTypeUint256                                   `contract:"storage"`
	Owner       CTypeAddress                                   `contract:"storage"`
	Balances    map[CTypeAddress]CTypeUint256                  `contract:"storage"`
	Allowances  map[CTypeAddress]map[CTypeAddress]CTypeUint256 `contract:"storage"`
}

var (
	zeroAddress = CTypeAddress{}
)

func (t *token) Create(
	ctx *ContractContext,
	name CTypeString,
	symbol CTypeString,
	decimals CTypeUint8,
	totalSupply CTypeUint256) error {
	t.Owner = NewAddress(ctx.caller)
	t.Name = name
	t.Symbol = symbol
	t.Decimals = decimals
	t.TotalSupply = totalSupply
	t.Allowances = make(map[CTypeAddress]map[CTypeAddress]CTypeUint256)
	t.Balances = make(map[CTypeAddress]CTypeUint256)
	t.Balances[t.Owner] = totalSupply
	return nil
}
func requireAddress(a CTypeAddress) bool {
	return !assertAddress(a, zeroAddress)
}
func assertAddress(a CTypeAddress, b CTypeAddress) bool {
	return bytes.Equal(a[:], b[:])
}

func (t *token) BuiltinId() uint8 {
	return 0x01
}

func (t *token) GetName() CTypeString {
	return t.Name
}

func (t *token) GetSymbol() CTypeString {
	return t.Symbol
}

func (t *token) GetDecimals() CTypeUint8 {
	return t.Decimals
}

func (t *token) GetTotalSupply() CTypeUint256 {
	return t.TotalSupply
}
func (t *token) Mint(ctx *ContractContext, address CTypeAddress, amount CTypeUint256) CTypeBool {
	if !assertAddress(NewAddress(ctx.caller), t.Owner) {
		return CTypeBool{0}
	}
	if !requireAddress(address) {
		return CTypeBool{0}
	}
	oldTotalSupply := new(big.Int).SetBytes(t.TotalSupply[:])
	amountValue := new(big.Int).SetBytes(amount[:])
	newTotalSupply := new(big.Int).Add(oldTotalSupply, amountValue)
	t.TotalSupply = NewUint256(newTotalSupply)
	return t.Transfer(ctx, address, amount)
}
func (t *token) BalanceOf(addr CTypeAddress) CTypeUint256 {
	aa := addr.Address()
	logrus.Infof("aa: %s", aa.B58String())
	if v, ok := t.Balances[addr]; ok {
		return v
	}
	return CTypeUint256{}
}
func (t *token) Transfer(ctx *ContractContext, address CTypeAddress, amount CTypeUint256) CTypeBool {
	if !requireAddress(address) {
		return CBoolFalse
	}
	caller := NewAddress(ctx.caller)
	if v, ok := t.Balances[caller]; ok {
		balance := new(big.Int).SetBytes(v[:])
		amountValue := new(big.Int).SetBytes(amount[:])
		residual := new(big.Int).Sub(balance, amountValue)
		if residual.Sign() < 0 {
			return CBoolFalse
		}
		t.Balances[caller] = NewUint256(residual)
		var targetBalance *big.Int
		if tv, ex := t.Balances[address]; ex {
			targetBalance = new(big.Int).SetBytes(tv[:])
		} else {
			targetBalance = big.NewInt(0)
		}
		newBalance := new(big.Int).Add(targetBalance, amountValue)
		t.Balances[address] = NewUint256(newBalance)
		return CBoolTrue
	}
	return CBoolFalse
}
func (t *token) TransferFrom(from, to CTypeAddress, amount CTypeUint256) CTypeBool {
	if !requireAddress(from) {
		return CBoolFalse
	}
	if !requireAddress(to) {
		return CBoolFalse
	}
	if v, ok := t.Balances[from]; ok {
		balance := new(big.Int).SetBytes(v[:])
		amountValue := new(big.Int).SetBytes(amount[:])
		residual := new(big.Int).Sub(balance, amountValue)
		if residual.Sign() < 0 {
			return CBoolFalse
		}
		t.Balances[from] = NewUint256(residual)
		var targetBalance *big.Int
		if tv, ex := t.Balances[from]; ex {
			targetBalance = new(big.Int).SetBytes(tv[:])
		} else {
			targetBalance = big.NewInt(0)
		}
		newBalance := new(big.Int).Add(targetBalance, amountValue)
		t.Balances[from] = NewUint256(newBalance)
		return CBoolTrue
	}
	return CBoolFalse
}
func (t *token) Approve(ctx *ContractContext, spender CTypeAddress, amount CTypeUint256) CTypeBool {
	owner := NewAddress(ctx.caller)
	if _, exists := t.Allowances[owner][spender]; exists {
		t.Allowances[owner][spender] = amount
	} else {
		t.Allowances[owner] = make(map[CTypeAddress]CTypeUint256)
		t.Allowances[owner][spender] = amount
	}
	if _, ok := t.Allowances[owner][spender]; ok {
	} else {
		t.Allowances[owner] = make(map[CTypeAddress]CTypeUint256)
		t.Allowances[owner][spender] = amount
	}
	return CBoolFalse
}
func (t *token) Allowance(owner, spender CTypeAddress) CTypeUint256 {
	if !requireAddress(owner) {
		return CTypeUint256{}
	}
	if !requireAddress(spender) {
		return CTypeUint256{}
	}
	if v, exists := t.Allowances[owner]; exists {
		if v == nil {
			return CTypeUint256{}
		}
		if vv, vvexists := v[spender]; vvexists {
			return vv
		}
	}
	return CTypeUint256{}
}
