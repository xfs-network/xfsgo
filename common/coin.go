// Copyright 2018 The xfsgo Authors
// This file is part of the xfsgo library.
//
// The xfsgo library is free software: you can redistribute it and/or modify
// it under the terms of the MIT Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The xfsgo library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// MIT Lesser General Public License for more details.
//
// You should have received a copy of the MIT Lesser General Public License
// along with the xfsgo library. If not, see <https://mit-license.org/>.

package common

import (
	"errors"
	"math"
	"math/big"
)

const Coin = 10 ^ 6

var AttoCoin = uint64(math.Pow(10, 18))
var NanoCoin = uint64(math.Pow(10, 9))

// gas
func BaseCoin2Atto(coin string) (*big.Int, error) {
	bigCoin, ok := new(big.Float).SetString(coin)
	if !ok {
		return nil, errors.New("string to big.Int Error")
	}
	attocoin := bigCoin.Mul(bigCoin, big.NewFloat(float64(AttoCoin)))
	i, _ := attocoin.Int(nil)
	return i, nil
}

// min coin
func Atto2BaseCoin(atto *big.Int) *big.Int {
	i := big.NewInt(0)
	i.Add(i, atto)
	i.Div(i, big.NewInt(int64(AttoCoin)))
	return i
}

// atto to nano
func AttoCoin2Nano(atto *big.Int) *big.Int {
	i := big.NewInt(0)
	i.Add(i, atto)
	i.Div(i, big.NewInt(int64(NanoCoin)))
	return i
}

// // min coin
// func Atto2BaseCoin(atto *big.Int) *big.Float {
// 	f := new(big.Float).SetInt(atto)
// 	fSub := new(big.Float).SetUint64(AttoCoin)

// 	f.Quo(f, fSub)
// 	return f
// }

func BaseCoin2Nano(coin string) (*big.Int, error) {

	bigCoin, ok := new(big.Float).SetString(coin)
	if !ok {
		return nil, errors.New("string to big.Int Error")
	}
	attocoin := bigCoin.Mul(bigCoin, big.NewFloat(float64(NanoCoin)))
	i, _ := attocoin.Int(nil)
	return i, nil
}

func NanoCoin2BaseCoin(nano *big.Int) *big.Int {
	i := big.NewInt(0)
	i.Add(i, nano)
	i.Div(i, big.NewInt(int64(NanoCoin)))
	return i
}

// nano to atto
func NanoCoin2Atto(nano *big.Int) *big.Int {
	a := new(big.Float).SetInt64(nano.Int64())
	a.Mul(a, big.NewFloat(float64(NanoCoin)))
	i, _ := a.Int(nil)
	return i
}

func Atto2BaseRatCoin(coin string) (*big.Float, error) {
	x, ok := new(big.Float).SetString(coin)
	if !ok {
		return nil, errors.New("string to big.Int Error")
	}
	y := new(big.Float).SetUint64(AttoCoin)
	sum := big.NewFloat(0)
	sum.Quo(x, y)
	return sum, nil
}

func Nano2BaseRatCoin(coin string) (*big.Float, error) {
	x, ok := new(big.Float).SetString(coin)
	if !ok {
		return nil, errors.New("string to big.Int Error")
	}
	y := new(big.Float).SetUint64(NanoCoin)
	sum := big.NewFloat(0)
	sum.Quo(x, y)
	return sum, nil
}
