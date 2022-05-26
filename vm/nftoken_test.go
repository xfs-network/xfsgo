package vm

import (
	"math/big"
	"testing"
)

var (
	mockNFToken = &nftoken{
		Name:    CTypeString("ACollection"),
		Symbol:  CTypeString("AC"),
		Creator: CTypeAddress{0xff},
	}
)

func TestNftoken_Create(t *testing.T) {
	stdToken := new(nftoken)
	err := stdToken.Create(&ContractContext{
		caller: mockNFToken.Creator.Address(),
		logger: NewLogger(),
	},
		mockNFToken.Name,
		mockNFToken.Symbol,
	)
	if err != nil {
		t.Fatal(err)
	}
	assertCTypeString(t, stdToken.Name, mockNFToken.Name)
	assertCTypeString(t, stdToken.Symbol, mockNFToken.Symbol)
	assertCTypeAddress(t, stdToken.Creator, mockNFToken.Creator)
	if stdToken.Balances == nil {
		t.Fatal("Balance map not be initial")
	}
	if stdToken.Owners == nil {
		t.Fatal("Owners map not be initial")
	}
	if stdToken.TokenAllowances == nil {
		t.Fatal("TokenAllowance map not be initial")
	}
	if stdToken.Allowances == nil {
		t.Fatal("Allowance map not be initial")
	}
	assertCTypeUint256(t, stdToken.Counter, CTypeUint256{})
}

func TestNftoken_Mint(t *testing.T) {

	stdToken := new(nftoken)
	err := stdToken.Create(&ContractContext{
		caller: mockNFToken.Creator.Address(),
		logger: NewLogger(),
	},
		mockNFToken.Name,
		mockNFToken.Symbol,
	)
	if err != nil {
		t.Fatal(err)
	}
	aAddress := CTypeAddress{0x1}
	bAddress := CTypeAddress{0x2}
	// 使用a地址给b地址铸造nft
	result := stdToken.Mint(&ContractContext{
		caller: aAddress.Address(),
		logger: NewLogger(),
	}, bAddress)
	// 预期结果：失败，没有铸造权限，返回零id
	assertCTypeUint256(t, result, CTypeUint256{})
	// 使用mock地址给a地址铸造
	result = stdToken.Mint(&ContractContext{
		caller: mockNFToken.Creator.Address(),
		logger: NewLogger(),
	}, aAddress)
	// 预期结果：成功, 返回非零id
	assertCTypeUint256NotEq(t, result, CTypeUint256{})
	// 查询藏品所有人
	// 预期结果为a地址
	owner := stdToken.OwnerOf(result)
	assertCTypeAddress(t, owner, aAddress)
	// 查询地址余额
	// 预期结果为1
	aBalance := stdToken.BalanceOf(aAddress)
	assertCTypeUint256(t, aBalance, NewUint256(big.NewInt(1)))
}

func TestNftoken_TransferFrom(t *testing.T) {

	stdToken := new(nftoken)
	err := stdToken.Create(&ContractContext{
		caller: mockNFToken.Creator.Address(),
		logger: NewLogger(),
	},
		mockNFToken.Name,
		mockNFToken.Symbol,
	)
	if err != nil {
		t.Fatal(err)
	}
	aAddress := CTypeAddress{0x1}
	bAddress := CTypeAddress{0x2}
	cAddress := CTypeAddress{0x3}
	// 使用a地址从b地址向c地址转移ID 1
	result := stdToken.TransferFrom(&ContractContext{
		caller: aAddress.Address(),
		logger: NewLogger(),
	}, bAddress, cAddress, NewUint256(big.NewInt(1)))
	// 预期结果：失败
	assertCTypeBool(t, result, CBoolFalse)
	// 使用合约创建者给b地址铸造nft
	tokenId := stdToken.Mint(&ContractContext{
		caller: mockNFToken.Creator.Address(),
		logger: NewLogger(),
	}, bAddress)
	// 预期结果，非零id
	assertCTypeUint256NotEq(t, tokenId, CTypeUint256{})
	// 使用a地址从b地址向c地址转移ID 1
	result = stdToken.TransferFrom(&ContractContext{
		caller: aAddress.Address(),
		logger: NewLogger(),
	}, bAddress, cAddress, NewUint256(big.NewInt(1)))
	// 预期结果：失败
	assertCTypeBool(t, result, CBoolFalse)
	// 使用b地址给a地址授予ID 1的转移权限
	approved := stdToken.Approve(&ContractContext{
		caller: bAddress.Address(),
		logger: NewLogger(),
	}, aAddress, NewUint256(big.NewInt(1)))
	assertCTypeBool(t, approved, CBoolTrue)
	// 再次转移
	result = stdToken.TransferFrom(&ContractContext{
		caller: aAddress.Address(),
		logger: NewLogger(),
	}, bAddress, cAddress, NewUint256(big.NewInt(1)))
	// 预期结果：成功
	assertCTypeBool(t, result, CBoolTrue)
	// 检查id 1的所属人是否为c地址
	owner := stdToken.OwnerOf(NewUint256(big.NewInt(1)))
	assertCTypeAddress(t, owner, cAddress)
	// 检查 b 地址余额是否为0
	bBalance := stdToken.BalanceOf(bAddress)
	assertCTypeUint256(t, bBalance, CTypeUint256{})
	// 检查授权是否被清理
	gotApproved := stdToken.GetApproved(NewUint256(big.NewInt(1)))
	assertCTypeAddress(t, gotApproved, CTypeAddress{})
}
