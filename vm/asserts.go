package vm

import (
	"bytes"
	"math/big"
	"testing"
)

func assertCTypeUint256(t *testing.T, got, want CTypeUint256) {
	gotValue := new(big.Int).SetBytes(got[:])
	wantValue := new(big.Int).SetBytes(want[:])
	if wantValue.Cmp(gotValue) != 0 {
		t.Fatalf("want value: '%s', but got value: '%s'", wantValue, gotValue)
	}
}

func assertCTypeUint256NotEq(t *testing.T, got, want CTypeUint256) {
	gotValue := new(big.Int).SetBytes(got[:])
	wantValue := new(big.Int).SetBytes(want[:])
	if wantValue.Cmp(gotValue) == 0 {
		t.Fatalf("want value: '%s', but got value: '%s'", wantValue, gotValue)
	}
}
func assertCTypeAddress(t *testing.T, got, want CTypeAddress) {
	if !bytes.Equal(got[:], want[:]) {
		t.Fatalf("want value: '0x%x', but got value: '0x%x'", want, got)
	}
}

func assertCTypeString(t *testing.T, got, want CTypeString) {
	if !bytes.Equal(got[:], want[:]) {
		t.Fatalf("want value: '0x%x', but got value: '0x%x'", want, got)
	}
}
func assertCTypeBool(t *testing.T, got, want CTypeBool) {
	if got != want {
		t.Fatalf("want value: '%d', but got value: '%d", want, got)
	}
}
