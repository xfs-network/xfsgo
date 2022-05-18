package crypto

import (
    "testing"
    "xfsgo/common"
    "xfsgo/common/ahash"
)

func TestCreateAddress(t *testing.T) {
	fromAddressText := "aJTobAyvdXeEGW7DHA1Yqc6PaVa2apHdX"
    wantAddress := "nTbqBjP3sYjwAFXf6e76nyuGpfbXd1P4i"
    wantNonce := uint64(1)
	fromAddress := common.StrB58ToAddress(fromAddressText)
	fromAddressHashBytes := ahash.SHA256(fromAddress.Bytes())
	fromAddressHash := common.Bytes2Hash(fromAddressHashBytes)
	gotAddress := CreateAddress(fromAddressHash, wantNonce)
    gotAddressStr := gotAddress.B58String()
    if gotAddressStr != wantAddress {
        t.Errorf("got: %s, want: %s", gotAddressStr, wantAddress)
    }
}
