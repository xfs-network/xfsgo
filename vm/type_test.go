package vm

import (
	"math/big"
	"testing"
)

func TestCTypeUint256_BigInt(t *testing.T) {
	value := big.NewInt(10)
	got := NewUint256(value)
	gotBig := got.BigInt()
	if gotBig.Cmp(value) != 0 {
		t.Fatalf("want value: %s, but got value: %s", value, gotBig)
	}
}
