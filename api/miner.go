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

package api

import (
	"fmt"
	"math/big"
	"strconv"
	"time"
	"xfsgo"
	"xfsgo/common"
	"xfsgo/miner"
)

type MinerAPIHandler struct {
	Miner *miner.Miner
}

// type MinerSetGasLimitArgs struct {
// 	Value string `json:"value"`
// }

type MinerSetGasPriceArgs struct {
	Value string `json:"value"`
}

type MinerSetCoinbaseArgs struct {
	Coinbase string `json:"coinbase"`
}

type MinerSetWorkerArgs struct {
	Num string `json:"num"`
}

func (handler *MinerAPIHandler) Start(args MinerStartArgs, resp *string) error {
	num, err := strconv.ParseUint(args.Num, 10, 32)
	if err != nil {
		return errorcase(err)
	}
	handler.Miner.Start(uint32(num))
	return nil

}

func (handler *MinerAPIHandler) Stop(_ EmptyArgs, resp *string) error {
	handler.Miner.Stop()
	*resp = ""
	return nil
}

func (handler *MinerAPIHandler) SetWorkers(args MinerSetWorkerArgs, resp *string) error {
	var err error
	var num int64
	if num, err = strconv.ParseInt(args.Num, 10, 64); err != nil {
		return errorcase(err)
	}
	return errorcase(handler.Miner.SetWorkers(uint32(num)))
}

func (handler *MinerAPIHandler) SetGasPrice(args MinerSetGasPriceArgs, resp *string) error {
	gaspriceBig, ok := new(big.Int).SetString(args.Value, 10)
	if !ok {
		return xfsgo.NewRPCError(-1006, "string to big.Int error")
	}
	GasPrice := common.NanoCoin2Atto(gaspriceBig)
	return errorcase(handler.Miner.SetGasPrice(GasPrice))
}

// func (handler *MinerAPIHandler) SetGasLimit(args MinerSetGasLimitArgs, resp *string) error {
// 	value, ok := new(big.Int).SetString(args.Value, 10)
// 	if !ok {
// 		return xfsgo.NewRPCError(-1006, "Number format err")
// 	}
// 	return errorcase(handler.Miner.SetGasLimit(value))
// }

func (handler *MinerAPIHandler) Status(_ EmptyArgs, resp *MinerStatusResp) error {
	mMiner := handler.Miner
	gasLimit := handler.Miner.GetGasLimit()
	gasPrice := handler.Miner.GetGasPrice()
	MinStartTime := handler.Miner.LastStartTime
	MinCoinbase := handler.Miner.Coinbase
	hashRate := handler.Miner.RunningHashRate()

	MinWorkers := handler.Miner.GetWorkerNum()
	MinStatus := handler.Miner.GetMinStatus()
	result := &MinerStatusResp{
		Status:           MinStatus,
		TargetHeight:     new(big.Int).SetUint64(mMiner.TargetHeight()).Text(10),
		TargetDifficulty: fmt.Sprintf("%.4f", mMiner.NextDifficulty()),
		TargetHashRate:   fmt.Sprintf("%.2f", mMiner.TargetHashRate()),
		GasLimit:         gasLimit.Text(10),
		GasPrice:         gasPrice.Text(10),
		LastStartTime:    MinStartTime.Format(time.RFC3339),
		Coinbase:         MinCoinbase.B58String(),
		HashRate:         fmt.Sprintf("%.2f", float64(hashRate)),
		Workers:          strconv.Itoa(int(MinWorkers)),
	}
	*resp = *result
	return nil
}
