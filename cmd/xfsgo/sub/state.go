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
	"fmt"
	"xfsgo"
	"xfsgo/common"

	"github.com/spf13/cobra"
)

var (
	roothash        string
	getStateCommand = &cobra.Command{
		Use:                   "state <command> [options]",
		DisableFlagsInUseLine: true,
		Short:                 "get state info",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	getAccountCommand = &cobra.Command{
		Use:                   "getaccount [options] <address>",
		DisableFlagsInUseLine: true,
		Short:                 "Specifies the hash value of the world state tree root",
		RunE:                  GetAccount,
	}
	getBalanceCommand = &cobra.Command{
		Use:                   "getbalance [options] <address>",
		DisableFlagsInUseLine: true,
		Short:                 "Specifies the hash value of the world state tree root",
		RunE:                  GetBalance,
	}
)

func GetAccount(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return cmd.Help()
	}
	address := args[0]
	rootHash := roothash

	config, err := parseClientConfig(cfgFile)
	if err != nil {
		return err
	}

	cli := xfsgo.NewClient(config.rpcClientApiHost, config.rpcClientApiTimeOut)
	result := make(map[string]interface{}, 1)
	req := &getAccountArgs{
		RootHash: rootHash,
		Address:  address,
	}
	err = cli.CallMethod(1, "State.GetAccount", &req, &result)
	if err != nil {
		fmt.Println(err)
		return err
	}
	bs, err := common.MarshalIndent(result)
	if err != nil {
		return err
	}
	fmt.Println(string(bs))
	return nil

}
func GetBalance(cmd *cobra.Command, args []string) error {
	if len(args) < 1 {
		return cmd.Help()
	}
	address := args[0]
	rootHash := roothash

	config, err := parseClientConfig(cfgFile)
	if err != nil {
		return err
	}

	cli := xfsgo.NewClient(config.rpcClientApiHost, config.rpcClientApiTimeOut)
	var result string
	req := &getAccountArgs{
		RootHash: rootHash,
		Address:  address,
	}
	err = cli.CallMethod(1, "State.GetBalance", &req, &result)
	if err != nil {
		return err
	}
	bal, err := common.Atto2BaseRatCoin(result)
	if err != nil {
		return err
	}
	balFloat, _ := bal.Float64()
	fmt.Printf("%-3.9f\n", balFloat)
	return nil
}

func init() {
	rootCmd.AddCommand(getStateCommand)
	getAccountCommandFlags := getAccountCommand.PersistentFlags()
	getAccountCommandFlags.StringVarP(&roothash, "root", "r", "", "Set state tree root hash")
	getStateCommand.AddCommand(getAccountCommand)
	getBalanceCommandFlags := getBalanceCommand.PersistentFlags()
	getBalanceCommandFlags.StringVarP(&roothash, "root", "r", "", "Set state tree root hash")
	getStateCommand.AddCommand(getBalanceCommand)
}
