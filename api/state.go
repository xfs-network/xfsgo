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
	"xfsgo"
	"xfsgo/avlmerkle"
	"xfsgo/common"
	"xfsgo/storage/badger"
)

type StateAPIHandler struct {
	StateDb    *badger.Storage
	BlockChain *xfsgo.BlockChain
}

type GetAccountArgs struct {
	RootHash string `json:"root_hash"`
	Address  string `json:"address"`
}

type GetBalanceArgs struct {
	RootHash string `json:"root_hash"`
	Address  string `json:"address"`
}

type GetStorageAtArgs struct {
	RootHash string `json:"root_hash"`
	Key      string `json:"key"`
}

func (state *StateAPIHandler) GetBalance(args GetBalanceArgs, resp **string) error {
	var rootHash common.Hash
	if args.RootHash == "" {
		rootHash = state.BlockChain.CurrentBHeader().StateRoot
	} else {
		if err := common.HashCalibrator(args.RootHash); err != nil {
			return xfsgo.NewRPCErrorCause(-32001, err)
		}
		rootHash = common.Hex2Hash(args.RootHash)
	}

	if args.Address == "" {
		return xfsgo.NewRPCError(-32601, "Address not found")
	}

	if err := common.AddrCalibrator(args.Address); err != nil {
		return xfsgo.NewRPCErrorCause(-32001, err)
	}

	rootHashByte := rootHash.Bytes()

	stateTree := xfsgo.NewStateTree(state.StateDb, rootHashByte)

	address := common.B58ToAddress([]byte(args.Address))

	data := stateTree.GetStateObj(address)

	if data == (&xfsgo.StateObj{}) || data == nil || data.GetBalance() == nil {
		**resp = "0"
		return nil
	}
	respstring := data.GetBalance().String()
	*resp = &respstring
	return nil

}

func (state *StateAPIHandler) GetAccount(args GetAccountArgs, resp **StateObjResp) error {
	var statehash []byte
	if args.RootHash == "" {
		rootHash := state.BlockChain.CurrentBHeader().StateRoot
		statehash = rootHash[:]
	} else {
		if err := common.HashCalibrator(args.RootHash); err != nil {
			return xfsgo.NewRPCErrorCause(-32001, err)
		}
		statehash = common.Hex2bytes(args.RootHash)
	}
	if args.Address == "" {
		return xfsgo.NewRPCError(-32601, "Address not found")
	}

	if err := common.AddrCalibrator(args.Address); err != nil {
		return xfsgo.NewRPCErrorCause(-32001, err)
	}

	stateTree := xfsgo.NewStateTree(state.StateDb, statehash)

	address := common.B58ToAddress([]byte(args.Address))

	data := stateTree.GetStateObj(address)
	return coverState2Resp(data, resp)
}

func (state *StateAPIHandler) GetStorageAt(args GetStorageAtArgs, resp **string) error {
	if args.RootHash == "" {
		return xfsgo.NewRPCError(-32601, "Require storage root hash")
	}
	rootHashBytes, err := common.HexToBytes(args.RootHash)
	if err != nil {
		return xfsgo.NewRPCErrorCause(-32001, err)
	}
	stateTree, err := avlmerkle.NewTreeN(state.StateDb, rootHashBytes)
	if err != nil {
		return xfsgo.NewRPCError(-32001,
			fmt.Sprintf("Notfound root hash: 0x%x", rootHashBytes))
	}
	keyBytes, err := common.HexToBytes(args.Key)
	if err != nil {
		return xfsgo.NewRPCErrorCause(-32001, err)
	}
	val, exists := stateTree.Get(keyBytes)
	if !exists {
		return xfsgo.NewRPCError(-32601,
			fmt.Sprintf("Notfound value by key: 0x%x", keyBytes[:]))
	}
	outhex := common.BytesToHexString(val)
	if outhex == "" {
		return xfsgo.NewRPCError(-32601,
			fmt.Sprintf("Notfound value by key: 0x%x", keyBytes[:]))
	}
	*resp = &outhex
	return nil
}
