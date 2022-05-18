package test

import (
	"math/big"
	"xfsgo/common"
)

var TestGenesisBits = uint32(534773790)

var TestTxGasPrice = common.NanoCoin2Atto(big.NewInt(10))
var TestTxGasLimit = big.NewInt(25000)

var TestTxPoolGasPrice = big.NewInt(24000)
var TestTxPoolGasLimit = new(big.Int).Mul(TestTxPoolGasPrice, common.Big100)

var TestMinerCoinbase = "WyRXjify4LGnBHpm9vfahQc4NBauxC6VR"
var TestMinerWorkers = uint32(10)

var Version = uint32(0)
