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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"xfsgo"
	"xfsgo/common"
	"xfsgo/crypto"

	"github.com/sirupsen/logrus"
)

type TxPoolHandler struct {
	TxPool *xfsgo.TxPool
}

type GetTranByHashArgs struct {
	Hash string `json:"hash"`
}

type GetAddrNonceByHashArgs struct {
	Address string `json:"address"`
}

// type ModTranGasArgs struct {
// 	GasLimit string `json:"gas_limit"`
// 	GasPrice string `json:"gas_price"`
// 	Hash     string `json:"hash"`
// }

type RawTransactionArgs struct {
	Data string `json:"data"`
}

type RemoveTxHashArgs struct {
	Hash string `json:"hash"`
}

func (tx *TxPoolHandler) GetPending(_ EmptyArgs, resp **TransactionsResp) error {
	data := tx.TxPool.GetPendingTxs()
	return coverTxs2Resp(data, resp)
}

func (tx *TxPoolHandler) GetQueue(_ EmptyArgs, resp **TransactionsResp) error {
	data := tx.TxPool.GetQueues()
	return coverTxs2Resp(data, resp)
}

func (tx *TxPoolHandler) GetPendingSize(_ EmptyArgs, resp *int) error {
	data := tx.TxPool.GetPendingTxsSize()
	*resp = data
	return nil
}

func (tx *TxPoolHandler) GetQueueSize(_ EmptyArgs, resp *int) error {
	data := tx.TxPool.GetQueueSize()
	*resp = data
	return nil
}

func (tx *TxPoolHandler) GetTxPoolSize(_ EmptyArgs, resp *int) error {
	data := tx.TxPool.GetTxPoolSize()
	*resp = int(data)
	return nil
}

func (tx *TxPoolHandler) RemoveTx(args RemoveTxHashArgs, resp *string) error {
	if args.Hash == "" {
		return xfsgo.NewRPCError(-1006, "Parameter cannot be empty")
	}
	if err := common.HashCalibrator(args.Hash); err != nil {
		return xfsgo.NewRPCErrorCause(-32001, err)
	}
	txobj := tx.TxPool.GetTransaction(common.Hex2Hash(args.Hash))
	tx.TxPool.RemoveTx(txobj)
	return nil
}

func (tx *TxPoolHandler) Clear(_ EmptyArgs, resp *string) error {
	tx.TxPool.RemoveTransactions(tx.TxPool.GetPendingTxs())
	return nil
}

func (tx *TxPoolHandler) RemoveQueues(_ EmptyArgs, resp *string) error {
	tx.TxPool.RemoveTransactions(tx.TxPool.GetQueues())
	return nil
}

func (tx *TxPoolHandler) GetTranByHash(args GetTranByHashArgs, resp **TransactionResp) error {
	if args.Hash == "" {
		return xfsgo.NewRPCError(-1006, "Parameter cannot be empty")
	}
	if err := common.HashCalibrator(args.Hash); err != nil {
		return xfsgo.NewRPCErrorCause(-32001, err)
	}

	tranObj := tx.TxPool.GetTransaction(common.Hex2Hash(args.Hash))
	return coverTx2Resp(tranObj, resp)
}

func (tx *TxPoolHandler) GetAddrTxNonce(args GetAddrNonceByHashArgs, resp *int64) error {
	if args.Address == "" {
		return xfsgo.NewRPCError(-1006, "Parameter data cannot be empty")
	}
	state := tx.TxPool.State()
	addr := common.B58ToAddress([]byte(args.Address))
	if !crypto.VerifyAddress(addr) {
		return xfsgo.NewRPCErrorCause(-32001, fmt.Errorf("failed to verify 'from' address: %s", addr))
	}
	nonce := state.GetNonce(addr)

	*resp = int64(nonce)
	return nil
}

func (tx *TxPoolHandler) SendRawTransaction(args RawTransactionArgs, resp *string) error {
	if args.Data == "" {
		return xfsgo.NewRPCError(-1006, "Parameter data cannot be empty")
	}
	// logrus.Infof("Handle RPC request by SendRawTransaction: args.data=%s", args.Data)
	//databytes, err := urlsafeb64.Decode(args.Data)
	databytes, err := base64.StdEncoding.DecodeString(args.Data)
	if err != nil {
		return xfsgo.NewRPCErrorCause(-32001, fmt.Errorf("failed to parse data: %s", err))
	}
	rawtx := &StringRawTransaction{}
	if err := json.Unmarshal(databytes, rawtx); err != nil {
		return xfsgo.NewRPCErrorCause(-32001, fmt.Errorf("failed to parse data: %s", err))
	}
	logrus.Debugf("Successfully parse transaction json data: %s", string(databytes))
	txdata, err := CoverTransaction(rawtx)
	if err != nil {
		return xfsgo.NewRPCErrorCause(-32001, err)
	}
	if err := tx.TxPool.Add(txdata); err != nil {
		return xfsgo.NewRPCErrorCause(-32001, err)
	}
	txhash := txdata.Hash()
	*resp = txhash.Hex()
	return nil
}
