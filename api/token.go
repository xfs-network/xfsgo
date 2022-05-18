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
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"xfsgo"
	"xfsgo/common"
	"xfsgo/crypto"
	"xfsgo/storage/badger"
	"xfsgo/vm"
)

type TokenCreateParams struct {
	From        string `json:"from"`
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	Decimals    string `json:"decimals"`
	TotalSupply string `json:"total_supply"`
}

var (
	tokenCreateParamsEncode = func(params TokenCreateParams) ([]byte, error) {
		buf := vm.NewBuffer(nil)
		if err := buf.WriteString(params.Name); err != nil {
			return nil, err
		}
		if err := buf.WriteString(params.Symbol); err != nil {
			return nil, err
		}
		decimals, ok := new(big.Int).SetString(params.Decimals, 10)
		if !ok {
			return nil, fmt.Errorf("failed to parse decimals")
		}
		decimalsData := decimals.Bytes()
		_, err := buf.Write(decimalsData[0:1])
		if err != nil {
			return nil, err
		}
		totalSupply, ok := new(big.Int).SetString(params.TotalSupply, 10)
		if !ok {
			return nil, fmt.Errorf("failed to parse decimals")
		}
		totalSupplyData := vm.NewUint256(totalSupply)
		_, err = buf.Write(totalSupplyData[:])
		if err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}
	tokenCode = []byte{
		0xd0, 0x23, 0x01,
	}
	tokenCreateDataEncode = func(params []byte) []byte {
		inputBuf := bytes.NewBuffer(nil)
		inputBuf.Write(tokenCode)
		inputBuf.Write(common.ZeroHash[:])
		inputBuf.Write(params)
		return inputBuf.Bytes()
	}
)

type TokenApiHandler struct {
	StateDb       badger.IStorage
	BlockChain    *xfsgo.BlockChain
	TxPendingPool *xfsgo.TxPool
	Wallet        *xfsgo.Wallet
}

type TokenCreateArgs struct {
	Value       string `json:"value"`
	GasLimit    string `json:"gas_limit"`
	GasPrice    string `json:"gas_price"`
	From        string `json:"from"`
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	Decimals    string `json:"decimals"`
	TotalSupply string `json:"total_supply"`
}
type TokenCreateResult struct {
	TransactionHash string `json:"transaction_hash"`
	ContractAddress string `json:"contract_address"`
}
type TokenGetArgs struct {
	Address   string `json:"address"`
	StateRoot string `json:"state_root"`
}

type TokenBalanceOfArgs struct {
	Contract  string `json:"contract"`
	StateRoot string `json:"state_root"`
	Address   string `json:"address"`
}

type TokenScheme struct {
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	Decimals    uint8  `json:"decimals"`
	TotalSupply string `json:"total_supply"`
}

func (handler *TokenApiHandler) Create(args TokenCreateArgs, resp *TokenCreateResult) error {
	var fromAddress common.Address
	if args.From == "" {
		fromAddress = handler.Wallet.GetDefault()
	} else {
		fromAddress = common.StrB58ToAddress(args.From)
	}
	key, err := handler.Wallet.GetKeyByAddress(fromAddress)
	if err != nil {
		return xfsgo.NewRPCErrorCause(-32603, err)
	}
	stdTx, err := ParseTransactionGasParams(&TransactionGasParams{
		GasPrice: args.GasPrice,
		GasLimit: args.GasLimit,
		Value:    args.Value,
	})
	if err != nil {
		return xfsgo.NewRPCErrorCause(-32603, err)
	}
	if stdTx.GasPrice != nil && stdTx.GasPrice.Sign() < 0 {
		return xfsgo.NewRPCErrorCause(-32603, fmt.Errorf("gas price must be >= 0"))
	} else if stdTx.GasPrice == nil || stdTx.GasPrice.Sign() == 0 {
		stdTx.GasPrice = common.NanoCoin2Atto(common.TxGasPrice)
	}
	if stdTx.GasLimit != nil && stdTx.GasLimit.Sign() < 0 {
		return xfsgo.NewRPCErrorCause(-32603, fmt.Errorf("gas limit must be >= 0"))
	} else if stdTx.GasLimit == nil || stdTx.GasLimit.Sign() == 0 {
		stdTx.GasLimit = common.TxGas
	}
	params, err := tokenCreateParamsEncode(TokenCreateParams{
		Name:        args.Name,
		Symbol:      args.Symbol,
		Decimals:    args.Decimals,
		TotalSupply: args.TotalSupply,
	})
	if err != nil {
		return xfsgo.NewRPCErrorCause(-32603, err)
	}
	stdTx.Data = tokenCreateDataEncode(params)
	stdTx.Nonce = handler.TxPendingPool.State().GetNonce(fromAddress)
	txn := xfsgo.NewTransactionByStdAndSign(stdTx, key)
	if err = handler.TxPendingPool.Add(txn); err != nil {
		return xfsgo.NewRPCErrorCause(-32603, err)
	}
	hash := txn.Hash()
	txhash := hash.Hex()

	caddr := crypto.CreateAddress(fromAddress.Hash(), stdTx.Nonce)
	*resp = TokenCreateResult{
		TransactionHash: txhash,
		ContractAddress: caddr.B58String(),
	}
	return nil
}

func (handler *TokenApiHandler) GetScheme(args TokenGetArgs, resp *TokenScheme) error {

	var stateRoot common.Hash
	var address common.Address
	if args.Address == "" {
		return xfsgo.NewRPCError(-32603, fmt.Sprintf("Contract address not be empty"))
	} else {
		address = common.StrB58ToAddress(args.Address)
	}
	if args.StateRoot == "" {
		lastBlock := handler.BlockChain.CurrentBHeader()
		stateRoot = lastBlock.StateRoot
	} else {
		stateRoot = common.Hex2Hash(args.StateRoot)
	}

	stateTree := xfsgo.NewStateTree(handler.StateDb, stateRoot[:])
	xvm := vm.NewXVM(stateTree)
	contract, err := xvm.GetBuiltinContract(address)
	if err != nil {
		return xfsgo.NewRPCErrorCause(-32603, err)
	}
	token, ok := contract.(vm.Token)
	if !ok || token == nil {
		return nil
	}
	name := token.GetName()
	var nameString string
	if !bytes.Equal(name[:], vm.CTypeString{}) {
		nameString = name.String()
	}
	symbol := token.GetSymbol()
	var symbolString string
	if !bytes.Equal(symbol[:], vm.CTypeString{}) {
		symbolString = symbol.String()
	}
	totalSupply := token.GetTotalSupply()
	var emptyUint256 = vm.CTypeUint256{}
	var totalSupplyHex string
	if !bytes.Equal(totalSupply[:], emptyUint256[:]) {
		totalSupplyHex = hex.EncodeToString(totalSupply[:])
	}
	decimals := token.GetDecimals()
	*resp = TokenScheme{
		Name:        nameString,
		Symbol:      symbolString,
		Decimals:    decimals.Uint8(),
		TotalSupply: totalSupplyHex,
	}
	return nil
}
func (handler *TokenApiHandler) BalanceOf(args TokenBalanceOfArgs, resp *string) error {
	var stateRoot common.Hash
	var contractAddress common.Address
	if args.Address == "" {
		return xfsgo.NewRPCError(-32603, fmt.Sprintf("Contract address not be empty"))
	} else {
		contractAddress = common.StrB58ToAddress(args.Contract)
	}
	if args.StateRoot == "" {
		lastBlock := handler.BlockChain.CurrentBHeader()
		stateRoot = lastBlock.StateRoot
	} else {
		stateRoot = common.Hex2Hash(args.StateRoot)
	}
	stateTree := xfsgo.NewStateTree(handler.StateDb, stateRoot[:])
	xvm := vm.NewXVM(stateTree)
	contract, err := xvm.GetBuiltinContract(contractAddress)
	if err != nil {
		return xfsgo.NewRPCErrorCause(-32603, err)
	}
	token, ok := contract.(vm.Token)
	if !ok || token == nil {
		return nil
	}
	callAddress := common.StrB58ToAddress(args.Address)
	balance := token.BalanceOf(vm.NewAddress(callAddress))
	balanceString := balance.BigInt().Text(10)
	*resp = balanceString
	return nil
}
