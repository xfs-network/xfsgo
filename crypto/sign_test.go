package crypto

import (
	"testing"
	"xfsgo/common/ahash"
)

func TestECDSASign(t *testing.T) {
	data := "hello"
	key := MustGenPrvKey()
	datahash := ahash.SHA256([]byte(data))
	signed, err := ECDSASign(datahash, key)
	if err != nil {
		t.Fatal(err)
	}
	if verified := ECDSAVerifySignature(key.PublicKey, datahash, signed); !verified {
		t.Fatal("check sign failed")
	}
}

