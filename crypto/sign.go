package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/hex"
	"fmt"
	"xfsgo/common"
	"xfsgo/crypto/secp256k1"
)

func ECDSASign2Hex(hash []byte, prv *ecdsa.PrivateKey) (string, error) {
	sig, err := ECDSASign(hash, prv)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(sig), nil
}

func ECDSASign(digestHash []byte, prv *ecdsa.PrivateKey) ([]byte, error) {
	if len(digestHash) != DigestLength {
		return nil, fmt.Errorf("hash is required to be exactly %d bytes (%d)", DigestLength, len(digestHash))
	}
	seckey := common.PaddedBigBytes(prv.D, prv.Params().BitSize/8)
	defer zeroBytes(seckey)
	return secp256k1.Sign(digestHash, seckey)
}

func ECDSASignNoRecover(digestHash []byte, prv *ecdsa.PrivateKey) ([]byte, error) {
	signed, err := ECDSASign(digestHash, prv)
	if err != nil {
		return nil, err
	}
	return signed[:len(signed)-1], nil
}

func Ecrecover(hash, sig []byte) ([]byte, error) {
	return secp256k1.RecoverPubkey(hash, sig)
}

func SigToPub(hash, sig []byte) (*ecdsa.PublicKey, error) {
	s, err := Ecrecover(hash, sig)
	if err != nil {
		return nil, err
	}
	x, y := elliptic.Unmarshal(secp256k1.S256(), s)
	return &ecdsa.PublicKey{Curve: secp256k1.S256(), X: x, Y: y}, nil
}
func CompressPubkey(pubkey *ecdsa.PublicKey) []byte {
	return secp256k1.CompressPubkey(pubkey.X, pubkey.Y)
}
func ECDSAVerifySignature(pubkey ecdsa.PublicKey, digestHash, signature []byte) bool {

	if len(signature) > 64 {
		signature = signature[:64]
	}
	return secp256k1.VerifySignature(CompressPubkey(&pubkey), digestHash, signature)
}

func zeroBytes(bytes []byte) {
	for i := range bytes {
		bytes[i] = 0
	}
}
