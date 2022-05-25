package vm

import "bytes"

var (
	zeroAddress = CTypeAddress{}
	zeroUInt256 = CTypeUint256{}
)

func requireAddress(a CTypeAddress) bool {
	return !assertAddress(a, zeroAddress)
}
func assertAddress(a CTypeAddress, b CTypeAddress) bool {
	return bytes.Equal(a[:], b[:])
}
func assertUInt256(a CTypeUint256, b CTypeUint256) bool {
	return bytes.Equal(a[:], b[:])
}
func requireTokenId(tokenId CTypeUint256) bool {
	return !assertUInt256(tokenId, zeroUInt256)
}
