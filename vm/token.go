package vm

import (
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
		return CBoolFalse
	}
	if !requireAddress(address) {
		return CBoolFalse
	}
	newTotalSupply := new(big.Int).Add(t.TotalSupply.BigInt(), amount.BigInt())
	t.TotalSupply = NewUint256(newTotalSupply)
	oldBalance := t.Balances[address]
	newBalance := new(big.Int).Add(oldBalance.BigInt(), amount.BigInt())
	t.Balances[address] = NewUint256(newBalance)
	return CBoolTrue
}
func (t *token) BalanceOf(addr CTypeAddress) CTypeUint256 {
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
		residual := new(big.Int).Sub(v.BigInt(), amount.BigInt())
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
		newBalance := new(big.Int).Add(targetBalance, amount.BigInt())
		t.Balances[address] = NewUint256(newBalance)
		return CBoolTrue
	}
	return CBoolFalse
}
func (t *token) TransferFrom(ctx *ContractContext, from, to CTypeAddress, amount CTypeUint256) CTypeBool {
	if !requireAddress(from) {
		return CBoolFalse
	}
	if !requireAddress(to) {
		return CBoolFalse
	}
	spender := NewAddress(ctx.caller)
	allowance := t.Allowance(from, spender)
	if allowance.BigInt().Cmp(amount.BigInt()) < 0 {
		return CBoolFalse
	}
	if v, ok := t.Balances[from]; ok {
		residual := new(big.Int).Sub(v.BigInt(), amount.BigInt())
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
		newBalance := new(big.Int).Add(targetBalance, amount.BigInt())
		t.Balances[from] = NewUint256(newBalance)
		oldAllowance := t.Allowances[from][spender]
		newAllowance := new(big.Int).Sub(oldAllowance.BigInt(), amount.BigInt())
		t.Allowances[from][spender] = NewUint256(newAllowance)
		return CBoolTrue
	}
	return CBoolFalse
}
func (t *token) Approve(ctx *ContractContext, spender CTypeAddress, amount CTypeUint256) CTypeBool {
	if !requireAddress(spender) {
		return CBoolFalse
	}
	owner := NewAddress(ctx.caller)
	if _, exists := t.Allowances[owner][spender]; exists {
		t.Allowances[owner][spender] = amount
		return CBoolTrue
	}
	if _, exists := t.Allowances[owner]; exists {
		t.Allowances[owner][spender] = amount
		return CBoolTrue
	}
	t.Allowances[owner] = make(map[CTypeAddress]CTypeUint256)
	t.Allowances[owner][spender] = amount
	return CBoolTrue
}
func (t *token) Allowance(owner, spender CTypeAddress) CTypeUint256 {
	if !requireAddress(owner) {
		return CTypeUint256{}
	}
	if !requireAddress(spender) {
		return CTypeUint256{}
	}
	if v, exists := t.Allowances[owner][spender]; exists {
		return v
	}
	return CTypeUint256{}
}

func (t *token) Burn(ctx *ContractContext, address CTypeAddress, amount CTypeUint256) CTypeBool {
	if !assertAddress(NewAddress(ctx.caller), t.Owner) {
		return CBoolFalse
	}
	if oldBalance, exists := t.Balances[address]; exists {
		newBalance := new(big.Int).Sub(oldBalance.BigInt(), amount.BigInt())
		if newBalance.Sign() < 0 {
			return CBoolFalse
		}
		t.Balances[address] = NewUint256(newBalance)
		oldTotalSupply := t.TotalSupply
		newTotalSupply := new(big.Int).Sub(oldTotalSupply.BigInt(), amount.BigInt())
		t.TotalSupply = NewUint256(newTotalSupply)
		return CBoolTrue
	}
	return CBoolFalse
}
