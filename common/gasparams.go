package common

import "math/big"

var TxGas = big.NewInt(25000)
var TxGasPrice = big.NewInt(10)

// var GasLimitBoundDivisor = big.NewInt(1024)
var GenesisGasLimit = new(big.Int).Mul(TxGas, Big100)
var MinGasLimit = TxGas

var TxPoolGasLimit = new(big.Int).Mul(TxGas, Big1K)

func CalcTxInitialCost(data []byte) *big.Int {
	igas := new(big.Int).Set(TxGas)
	return igas
}

func DefaultGasPrice() *big.Int {
	return NanoCoin2Atto(TxGasPrice)
}
