package vm

import (
	"math/big"
	"testing"
	"xfsgo/common"
)

var (
	testCtx = &ContractContext{
		caller: common.Address{0xff},
		logger: NewLogger(),
	}
	testWantToken = &token{
		Name:        CTypeString("Tether USD"),
		Symbol:      CTypeString("USDT"),
		Decimals:    CTypeUint8{18},
		TotalSupply: NewUint256(big.NewInt(10)),
		Owner:       CTypeAddress{0xff},
	}
)

func TestToken_Create(t *testing.T) {
	stdToken := new(token)
	err := stdToken.Create(testCtx,
		testWantToken.Name,
		testWantToken.Symbol,
		testWantToken.Decimals,
		testWantToken.TotalSupply,
	)
	if err != nil {
		t.Fatal(err)
	}
	assertCTypeString(t, stdToken.Name, testWantToken.Name)
	assertCTypeString(t, stdToken.Symbol, testWantToken.Symbol)
	if stdToken.Decimals != testWantToken.Decimals {
		t.Fatalf("got value: %d, but want value: %d", stdToken.Decimals, testWantToken.Decimals)
	}
	assertCTypeUint256(t, stdToken.TotalSupply, testWantToken.TotalSupply)
	assertCTypeAddress(t, stdToken.Owner, testWantToken.Owner)
	if balance, exists := stdToken.Balances[stdToken.Owner]; exists {
		assertCTypeUint256(t, balance, stdToken.TotalSupply)
	} else {
		t.Fatalf("unable got value, but want value: 0x%x", stdToken.Balances)
	}
}

func TestToken_Transfer(t *testing.T) {
	stdToken := new(token)
	err := stdToken.Create(testCtx,
		testWantToken.Name,
		testWantToken.Symbol,
		testWantToken.Decimals,
		testWantToken.TotalSupply,
	)
	if err != nil {
		t.Fatal(err)
	}
	sender := NewAddress(testCtx.caller)
	targetAddress := CTypeAddress{0xf1}
	transferAmount := NewUint256(big.NewInt(1))
	oldbalance := stdToken.Balances[sender]
	wantSenderBalance := new(big.Int).Sub(oldbalance.BigInt(), transferAmount.BigInt())
	// 第一次转移测试，使用有余额的发送地址向没有余额的目标地址转移且余额足够
	// 期望结果：成功
	result := stdToken.Transfer(testCtx, targetAddress, transferAmount)
	assertCTypeBool(t, result, CBoolTrue)
	// 期望结果：目标地址的值等于新值
	gotTargetBalance := stdToken.BalanceOf(targetAddress)
	assertCTypeUint256(t, gotTargetBalance, transferAmount)
	// 期望结果：发送地址的值为扣减正确的值
	gotSenderBalance := stdToken.BalanceOf(sender)
	assertCTypeUint256(t, gotSenderBalance, NewUint256(wantSenderBalance))
	wrongContext := &ContractContext{
		caller: common.Address{0x10},
		logger: NewLogger(),
	}
	// 第二次转移测试, 使用没有余额的发送地址向有余额的目标地址转移
	// 期望结果：失败
	result = stdToken.Transfer(wrongContext, targetAddress, transferAmount)
	assertCTypeBool(t, result, CBoolFalse)
	zeroContext := &ContractContext{
		caller: common.Address{},
	}
	// 第三次转移测试，使用零地址作为发送地址向目标地址转移
	// 期望结果：失败
	result = stdToken.Transfer(zeroContext, targetAddress, transferAmount)
	assertCTypeBool(t, result, CBoolFalse)
	// 第四次测试，使用正常发送地址向零地址转移
	// 期望结果：失败
	result = stdToken.Transfer(testCtx, zeroAddress, transferAmount)
	assertCTypeBool(t, result, CBoolFalse)
}
func TestToken_Approve(t *testing.T) {
	stdToken := new(token)
	err := stdToken.Create(testCtx,
		testWantToken.Name,
		testWantToken.Symbol,
		testWantToken.Decimals,
		testWantToken.TotalSupply,
	)
	if err != nil {
		t.Fatal(err)
	}
	testSpender := CTypeAddress{0xf1}
	wantAmount := NewUint256(big.NewInt(10))
	// 第一次测试向一个不存在的owner,spender的key赋值
	// 期望结果：成功，符合预期值
	result := stdToken.Approve(testCtx, testSpender, wantAmount)
	assertCTypeBool(t, result, CBoolTrue)
	ownerAddress := NewAddress(testCtx.caller)
	gotSpenderAllowance := stdToken.Allowance(ownerAddress, testSpender)
	assertCTypeUint256(t, gotSpenderAllowance, wantAmount)
	wantAmount = NewUint256(big.NewInt(20))
	// 第二次测试使用已存在的owner,spender的key修改其值
	// 期望结果：成功，并且符合预期值
	result = stdToken.Approve(testCtx, testSpender, wantAmount)
	assertCTypeBool(t, result, CBoolTrue)
	gotSpenderAllowance = stdToken.Allowance(ownerAddress, testSpender)
	assertCTypeUint256(t, gotSpenderAllowance, wantAmount)
	testSpender = CTypeAddress{0x99}
	// 第三次测试向已存在owner，不存在的spender的key赋值
	// 期望结果：成功，并且符合预期值
	result = stdToken.Approve(testCtx, testSpender, wantAmount)
	assertCTypeBool(t, result, CBoolTrue)
	gotSpenderAllowance = stdToken.Allowance(ownerAddress, testSpender)
	assertCTypeUint256(t, gotSpenderAllowance, wantAmount)
}

func TestToken_Mint(t *testing.T) {
	stdToken := new(token)
	err := stdToken.Create(testCtx,
		testWantToken.Name,
		testWantToken.Symbol,
		testWantToken.Decimals,
		testWantToken.TotalSupply,
	)
	if err != nil {
		t.Fatal(err)
	}
	oldTotalSupply := stdToken.TotalSupply
	wantTotalSupply := new(big.Int).Add(oldTotalSupply.BigInt(), big.NewInt(10))
	oldOwnerBalance := stdToken.Balances[stdToken.Owner]
	wantOwnerBalance := new(big.Int).Add(oldOwnerBalance.BigInt(), big.NewInt(10))

	// 使用owner地址，向owner地址铸造
	// 期望结果：成功，并且符合预期值
	result := stdToken.Mint(testCtx, stdToken.Owner, NewUint256(big.NewInt(10)))
	assertCTypeBool(t, result, CBoolTrue)
	assertCTypeUint256(t, stdToken.TotalSupply, NewUint256(wantTotalSupply))
	assertCTypeUint256(t, stdToken.Balances[stdToken.Owner], NewUint256(wantOwnerBalance))

	testAddress := CTypeAddress{0x7}
	oldTestBalance := stdToken.Balances[testAddress]
	wantTestBalance := new(big.Int).Add(oldTestBalance.BigInt(), big.NewInt(10))
	oldTotalSupply = stdToken.TotalSupply
	wantTotalSupply = new(big.Int).Add(oldTotalSupply.BigInt(), big.NewInt(10))

	// 使用owner地址，向其他地址铸造
	// 期望结果：成功，并且符合预期值
	result = stdToken.Mint(testCtx, testAddress, NewUint256(big.NewInt(10)))
	assertCTypeBool(t, result, CBoolTrue)
	assertCTypeUint256(t, stdToken.TotalSupply, NewUint256(wantTotalSupply))
	assertCTypeUint256(t, stdToken.Balances[testAddress], NewUint256(wantTestBalance))
	testOtherContext := &ContractContext{
		caller: common.Address{0x06},
	}
	// 使用其他地址铸造
	// 期望结果：失败
	result = stdToken.Mint(testOtherContext, testAddress, NewUint256(big.NewInt(10)))
	assertCTypeBool(t, result, CBoolFalse)
}

func TestToken_TransferFrom(t *testing.T) {
	stdToken := new(token)
	err := stdToken.Create(testCtx,
		testWantToken.Name,
		testWantToken.Symbol,
		testWantToken.Decimals,
		testWantToken.TotalSupply,
	)
	if err != nil {
		t.Fatal(err)
	}
	aAddress := CTypeAddress{0x1}
	bAddress := CTypeAddress{0x2}
	cAddress := CTypeAddress{0x3}
	// 测试1: 使用 a 地址从 b 地址向 c 地址转移 10
	// 预期结果：失败，消息发送者没有 b 地址的授额
	result := stdToken.TransferFrom(&ContractContext{
		caller: aAddress.Address(),
		logger: NewLogger(),
	}, bAddress, cAddress, NewUint256(big.NewInt(10)))
	assertCTypeBool(t, result, CBoolFalse)
	// 使用 b 地址作为发送者，授予 a 地址的额度 5
	result = stdToken.Approve(&ContractContext{
		caller: bAddress.Address(),
		logger: NewLogger(),
	}, aAddress, NewUint256(big.NewInt(5)))
	assertCTypeBool(t, result, CBoolTrue)
	// 继续测试1
	// 预期结果：失败，b 地址没有余额
	result = stdToken.TransferFrom(&ContractContext{
		caller: aAddress.Address(),
		logger: NewLogger(),
	}, bAddress, aAddress, NewUint256(big.NewInt(10)))
	assertCTypeBool(t, result, CBoolFalse)
	// 给 b 地址 100 余额，并再次尝试测试1
	// 预期结果：失败, a 地址拥有 b 地址的转移额度不够
	result = stdToken.Mint(testCtx, bAddress, NewUint256(big.NewInt(100)))
	assertCTypeBool(t, result, CBoolTrue)
	result = stdToken.TransferFrom(&ContractContext{
		caller: aAddress.Address(),
		logger: NewLogger(),
	}, bAddress, cAddress, NewUint256(big.NewInt(10)))
	assertCTypeBool(t, result, CBoolFalse)
	// 使用 b 地址向 a 地址重新授额 20
	result = stdToken.Approve(&ContractContext{
		caller: bAddress.Address(),
		logger: NewLogger(),
	}, aAddress, NewUint256(big.NewInt(20)))
	assertCTypeBool(t, result, CBoolTrue)
	// 再次尝试测试1
	// 预期结果：成功
	result = stdToken.TransferFrom(&ContractContext{
		caller: aAddress.Address(),
		logger: NewLogger(),
	}, bAddress, cAddress, NewUint256(big.NewInt(10)))
	assertCTypeBool(t, result, CBoolTrue)
	// 检查a地址拥有b地址的转移额度是否扣减
	// 预期结果应该为20-10=10
	allowance := stdToken.Allowance(bAddress, aAddress)
	assertCTypeUint256(t, allowance, NewUint256(big.NewInt(10)))
}

func TestToken_Burn(t *testing.T) {
	stdToken := new(token)
	err := stdToken.Create(testCtx,
		testWantToken.Name,
		testWantToken.Symbol,
		testWantToken.Decimals,
		testWantToken.TotalSupply,
	)
	if err != nil {
		t.Fatal(err)
	}
	aAddress := CTypeAddress{0x1}
	// 给 a 地址预设余额 100
	mintAmount := NewUint256(big.NewInt(100))
	result := stdToken.Mint(testCtx, aAddress, mintAmount)
	assertCTypeBool(t, result, CBoolTrue)
	oldTotalSupply := stdToken.GetTotalSupply()
	// 燃烧 a 地址余额 20
	burnAmount := NewUint256(big.NewInt(20))
	result = stdToken.Burn(testCtx, aAddress, burnAmount)
	assertCTypeBool(t, result, CBoolTrue)
	// 检查a的余额是否为100-20=80
	gotABalance := stdToken.BalanceOf(aAddress)
	assertCTypeUint256(t, gotABalance, NewUint256(big.NewInt(80)))
	// 检查总量是否已扣减
	wantTotalSupply := new(big.Int).Sub(oldTotalSupply.BigInt(), burnAmount.BigInt())
	assertCTypeUint256(t, stdToken.TotalSupply, NewUint256(wantTotalSupply))
	// 使用 a 地址燃烧 b地址余额
	// 预期结果：失败，权限不足
	bAddress := CTypeAddress{0x2}
	result = stdToken.Burn(&ContractContext{
		caller: aAddress.Address(),
		logger: NewLogger(),
	}, bAddress, NewUint256(big.NewInt(10)))
	assertCTypeBool(t, result, CBoolFalse)
}
