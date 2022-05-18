package common
import (
    "testing"
    "xfsgo/common/ahash"
)

func TestMakeSateKey(t *testing.T) {
    addr := StrB58ToAddress("o8GK8KLkMr6y2sxmdx9CCsv6AV2r9XpFC")
    keyHash := ahash.SHA256Array([]byte("Name"))
    namespace := MakeStateKey(addr, keyHash[:])
    namespacestr := BytesToHexString(namespace)
    wantKey := "0xb3bbb6a93301e07c0f508a4e9d8e65a91ea52da4d668ed6441e81d8b5f7cbe6f"
    if namespacestr != wantKey {
        t.Errorf("got: 0x%x, want: 0x%x", namespacestr, wantKey)
    }
}
