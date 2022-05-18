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

package sub

import (
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"xfsgo"
	"xfsgo/common"

	"github.com/spf13/cobra"
)

var (
	workers      string
	minerCommand = &cobra.Command{
		Use:                   "miner <command> [options]",
		DisableFlagsInUseLine: true,
		Short:                 "miner serve info",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	minerStartCommand = &cobra.Command{
		Use:                   "start [options]",
		DisableFlagsInUseLine: true,
		Short:                 "Start mining service",
		RunE:                  runMinerStart,
	}
	minerStopCommand = &cobra.Command{
		Use:                   "stop [options]",
		DisableFlagsInUseLine: true,
		Short:                 "Stop mining services",
		RunE:                  runMinerStop,
	}
	minerSetWorkersCommand = &cobra.Command{
		Use:                   "setworkers [options] [count]",
		DisableFlagsInUseLine: true,
		Short:                 "Start mining service",
		RunE:                  setWorkers,
	}
	minerSetGasPriceCommand = &cobra.Command{
		Use:                   "setgasprice [options] <price>",
		DisableFlagsInUseLine: true,
		Short:                 "Miner set gas price",
		RunE:                  setGasPrice,
	}
	minerGetStatusCommand = &cobra.Command{
		Use:                   "status [options]",
		DisableFlagsInUseLine: true,
		Short:                 "Get current miner status",
		RunE:                  getStatus,
	}
)

func runMinerStart(_ *cobra.Command, args []string) error {
	config, err := parseClientConfig(cfgFile)
	if err != nil {
		return err
	}
	var res *string = nil
	req := &minerStartArgs{
		Num: workers,
	}
	cli := xfsgo.NewClient(config.rpcClientApiHost, config.rpcClientApiTimeOut)
	if err = cli.CallMethod(1, "Miner.Start", &req, &res); err != nil {
		return err
	}
	fmt.Printf("Miner started successfully\n")
	return nil
}

func runMinerStop(_ *cobra.Command, _ []string) error {
	config, err := parseClientConfig(cfgFile)
	if err != nil {
		return err
	}
	var res *string = nil
	cli := xfsgo.NewClient(config.rpcClientApiHost, config.rpcClientApiTimeOut)
	err = cli.CallMethod(1, "Miner.Stop", nil, &res)
	if err != nil {
		return err
	}
	fmt.Println("Miner stopped...")
	return nil
}
func setWorkers(cmd *cobra.Command, args []string) error {
	config, err := parseClientConfig(cfgFile)
	if err != nil {
		return err
	}
	var res *string = nil
	cli := xfsgo.NewClient(config.rpcClientApiHost, config.rpcClientApiTimeOut)
	req := &MinerWorkerArgs{
		Num: args[0],
	}
	err = cli.CallMethod(1, "Miner.SetWorkers", &req, &res)
	if err != nil {
		return err
	}
	return nil
}

func setGasPrice(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("value err")
	}
	config, err := parseClientConfig(cfgFile)
	if err != nil {
		return err
	}
	req := &MinerSetGasPriceArgs{
		Value: args[0],
	}
	var res *string = nil
	cli := xfsgo.NewClient(config.rpcClientApiHost, config.rpcClientApiTimeOut)
	err = cli.CallMethod(1, "Miner.SetGasPrice", &req, &res)
	if err != nil {
		return err
	}
	return nil
}

func getStatus(_ *cobra.Command, _ []string) error {
	config, err := parseClientConfig(cfgFile)
	if err != nil {
		return err
	}
	res := make(map[string]interface{})
	cli := xfsgo.NewClient(config.rpcClientApiHost, config.rpcClientApiTimeOut)
	err = cli.CallMethod(1, "Miner.Status", nil, &res)
	if err != nil {
		return nil
	}

	var statusStr string
	if res["status"].(bool) {
		statusStr = "Running"
	} else {
		statusStr = "Stop"
	}
	minerHashRateStr := res["hash_rate"].(string)
	minerHashRate, _ := strconv.ParseFloat(minerHashRateStr, 64)
	hashrate := common.HashRate(minerHashRate)
	targetDifficulty := res["target_difficulty"].(string)
	targetHashRateStr := res["target_hash_rate"].(string)
	targetHashRate, _ := strconv.ParseFloat(targetHashRateStr, 64)
	targethashrate := common.HashRate(targetHashRate)
	fmt.Printf("Status: %v\n", statusStr)
	fmt.Printf("LastStarTime: %s\n", res["last_start_time"])
	fmt.Printf("Workers: %s\n", res["workers"])
	fmt.Printf("Coinbase: %s\n", res["coinbase"])

	// common.NanoCoin
	gasPriceBig, ok := new(big.Int).SetString(res["gas_price"].(string), 10)
	if !ok {
		return errors.New("string to big.Int error")
	}

	gasPrice := common.NanoCoin2BaseCoin(gasPriceBig)
	fmt.Printf("GasPrice: %s (Nano)\n", gasPrice.String())
	fmt.Printf("GasLimit: %s\n", res["gas_limit"].(string))
	fmt.Printf("TargetHeight: %s\n", res["target_height"].(string))
	fmt.Printf("TargetDifficulty: %s\n", targetDifficulty)
	fmt.Printf("TargetHashRate: %s\n", targethashrate)
	fmt.Printf("HashRate: %s\n", hashrate)
	return nil
}
func init() {
	minerCommand.AddCommand(minerStartCommand)
	mFlags := minerStartCommand.PersistentFlags()
	mFlags.StringVarP(&workers, "workers", "", "0", "Set number of workers")
	minerCommand.AddCommand(minerStopCommand)
	minerCommand.AddCommand(minerSetGasPriceCommand)
	// minerCommand.AddCommand(minerSetGasLimitCommand)
	minerCommand.AddCommand(minerGetStatusCommand)
	minerCommand.AddCommand(minerSetWorkersCommand)
	rootCmd.AddCommand(minerCommand)
}
