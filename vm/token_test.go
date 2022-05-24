package vm

import (
	"bytes"
	"math/big"
	"testing"
	"xfsgo/common"
)

var (
	testCtx = &ContractContext{
		caller: common.Address{0xff},
	}
	testWantToken = &token{
		Name:        CTypeString("Tether USD"),
		Symbol:      CTypeString("USDT"),
		Decimals:    CTypeUint8{18},
		TotalSupply: CTypeUint256{0xff},
		Owner:       CTypeAddress{0xff},
	}
)

func assertCTypeUint256(t *testing.T, got, want CTypeUint256) {
	gotValue := new(big.Int).SetBytes(got[:])
	wantValue := new(big.Int).SetBytes(want[:])
	if wantValue.Cmp(gotValue) != 0 {
		t.Fatalf("want value: '%s', but got value: '%s'", wantValue, gotValue)
	}
}
func assertCTypeBool(t *testing.T, want, got CTypeBool) {
	if got != want {
		t.Fatalf("want value: '%d', but got value: '%d", want, got)
	}
}

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
	if !bytes.Equal(stdToken.Name, testWantToken.Name) {
		t.Fatalf("got Name: 0x%x, want Name: 0x%x", stdToken.Name, testWantToken.Name)
	}
	if !bytes.Equal(stdToken.Symbol, testWantToken.Symbol) {
		t.Fatalf("got Symbol: 0x%x, want Symbol: 0x%x", stdToken.Symbol, testWantToken.Symbol)
	}
	if stdToken.Decimals != testWantToken.Decimals {
		t.Fatalf("got Decimals: %d, want Decimals: %d", stdToken.Decimals, testWantToken.Decimals)
	}
	if !bytes.Equal(stdToken.TotalSupply[:], testWantToken.TotalSupply[:]) {
		t.Fatalf("got TotalSupply: 0x%x, want TotalSupply: 0x%x", stdToken.TotalSupply, testWantToken.TotalSupply)
	}
	if !bytes.Equal(stdToken.Owner[:], testWantToken.Owner[:]) {
		t.Fatalf("got Owner: 0x%x, want Owner: 0x%x", stdToken.Owner, testWantToken.Owner)
	}
	if balance, exists := stdToken.Balances[stdToken.Owner]; exists {
		assertCTypeUint256(t, balance, stdToken.TotalSupply)
	} else {
		t.Fatalf("unable got Owner balance, want Owner balance: 0x%x", stdToken.Balances)
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
	targetAddress := CTypeAddress{0xf1}
	transferAmount := CTypeUint256{0x10}
	result := stdToken.Transfer(testCtx, targetAddress, transferAmount)
	assertCTypeBool(t, result, CBoolTrue)
	if balance, exists := stdToken.Balances[targetAddress]; exists {
		assertCTypeUint256(t, balance, transferAmount)
	} else {
		t.Fatalf("unable got target balance, want target balance: 0x%x", transferAmount)
	}
	wrongContext := &ContractContext{
		caller: common.Address{0x10},
	}
	result = stdToken.Transfer(wrongContext, targetAddress, transferAmount)
	assertCTypeBool(t, result, CBoolFalse)
	zeroContext := &ContractContext{
		caller: common.Address{},
	}
	result = stdToken.Transfer(zeroContext, targetAddress, transferAmount)
	assertCTypeBool(t, result, CBoolFalse)
	result = stdToken.Transfer(zeroContext, zeroAddress, transferAmount)
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
	testSpender := CTypeAddress{0xff}
	stdToken.Approve(testCtx, testSpender, NewUint256(big.NewInt(10)))
}
