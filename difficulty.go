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

package xfsgo

import (
	"math/big"
	"xfsgo/common"
)

var (
	big0xff = big.NewInt(0xff)
)

// BigByZip zips 256 bit difficulty to uint32
func BigByZip(target *big.Int) uint32 {
	if target.Sign() == 0 {
		return 0
	}
	c := uint(3)
	e := uint(len(target.Bytes()))
	var mantissa uint
	if e <= c {
		mantissa = uint(target.Bits()[0])
		shift := 8 * (c - e)
		mantissa <<= shift
	} else {
		shift := 8 * (e - c)
		mantissaNum := target.Rsh(target, shift)
		mantissa = uint(mantissaNum.Bits()[0])
	}
	mantissa <<= 8
	mantissa = mantissa & 0xffffffff
	return uint32(mantissa | e)
}

// BitsUnzip unzips 32bit Bits in BlockHeader to 256 bit Difficulty.
func BitsUnzip(bits uint32) *big.Int {
	mantissa := bits & 0xffffff00
	mantissa >>= 8
	e := uint(bits & 0xff)
	c := uint(3)
	var bn *big.Int
	if e <= c {
		shift := 8 * (c - e)
		mantissa >>= shift
		bn = big.NewInt(int64(mantissa))
	} else {
		bn = big.NewInt(int64(mantissa))
		shift := 8 * (e - c)
		bn.Lsh(bn, shift)
	}
	return bn
}

func CalcDifficultyByBits(bits uint32) float64 {
	return float64(GenesisBits) / float64(bits)
}

func CalcWorkloadByBits(bits uint32) *big.Int {
	target := BitsUnzip(bits)
	max := BitsUnzip(GenesisBits)
	difficulty := new(big.Int).Div(max, target)
	n1 := new(big.Int).Mul(difficulty, common.Big256Bits)
	return new(big.Int).Div(n1, max)
}

func CalcHashRateByBits(bits uint32) common.HashRate {
	df := CalcDifficultyByBits(bits)
	difficulty := new(big.Int).SetInt64(int64(df))
	n1 := new(big.Int).Mul(difficulty, common.Big256Bits)
	workload := new(big.Int).Div(n1, BitsUnzip(GenesisBits))
	workload64 := workload.Uint64()
	rateN := float64(workload64) / float64(targetTimePerBlock)
	return common.HashRate(rateN)
}
