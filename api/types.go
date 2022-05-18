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
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"xfsgo"
	"xfsgo/common"
	"xfsgo/crypto"
)

type EmptyArgs = interface{}

type StateObjResp struct {
	Balance   *string      `json:"balance"`
	Nonce     uint64       `json:"nonce"`
	Extra     *string      `json:"extra"`
	Code      *string      `json:"code"`
	StateRoot *common.Hash `json:"state_root"`
}

type BlockHeaderResp struct {
	Height        uint64         `json:"height"`
	Version       uint32         `json:"version"`
	HashPrevBlock common.Hash    `json:"hash_prev_block"`
	Timestamp     uint64         `json:"timestamp"`
	Coinbase      common.Address `json:"coinbase"`
	// merkle tree root hash
	StateRoot        common.Hash `json:"state_root"`
	TransactionsRoot common.Hash `json:"transactions_root"`
	ReceiptsRoot     common.Hash `json:"receipts_root"`
	GasLimit         *big.Int    `json:"gas_limit"`
	GasUsed          *big.Int    `json:"gas_used"`
	// pow
	Bits       uint32      `json:"bits"`
	Nonce      uint32      `json:"nonce"`
	ExtraNonce uint64      `json:"extranonce"`
	Hash       common.Hash `json:"hash"`
}

type BlockResp struct {
	Height        uint64         `json:"height"`
	Version       uint32         `json:"version"`
	HashPrevBlock common.Hash    `json:"hash_prev_block"`
	Timestamp     uint64         `json:"timestamp"`
	Coinbase      common.Address `json:"coinbase"`
	// merkle tree root hash
	StateRoot        common.Hash `json:"state_root"`
	TransactionsRoot common.Hash `json:"transactions_root"`
	ReceiptsRoot     common.Hash `json:"receipts_root"`
	GasLimit         *big.Int    `json:"gas_limit"`
	GasUsed          *big.Int    `json:"gas_used"`
	// pow
	Bits         uint32           `json:"bits"`
	Nonce        uint32           `json:"nonce"`
	ExtraNonce   uint64           `json:"extranonce"`
	Hash         common.Hash      `json:"hash"`
	Transactions TransactionsResp `json:"transactions"`
}

type TransactionResp struct {
	Version   uint32         `json:"version"`
	To        common.Address `json:"to"`
	GasPrice  *big.Int       `json:"gas_price"`
	GasLimit  *big.Int       `json:"gas_limit"`
	Nonce     uint64         `json:"nonce"`
	Value     *big.Int       `json:"value"`
	From      string         `json:"from"`
	Hash      common.Hash    `json:"hash"`
	Data      string         `json:"data"`
	Signature string         `json:"signature"`
}

type MinerStartArgs struct {
	Num string `json:"num"`
}

type MinerStatusResp struct {
	Status           bool   `json:"status"`
	LastStartTime    string `json:"last_start_time"`
	Workers          string `json:"workers"`
	Coinbase         string `json:"coinbase"`
	GasPrice         string `json:"gas_price"`
	GasLimit         string `json:"gas_limit"`
	TargetHeight     string `json:"target_height"`
	TargetDifficulty string `json:"target_difficulty"`
	TargetHashRate   string `json:"target_hash_rate"`
	HashRate         string `json:"hash_rate"`
}

type ReceiptResp struct {
	Version     uint32      `json:"version"`
	Status      uint32      `json:"status"`
	TxHash      common.Hash `json:"tx_hash"`
	GasUsed     *big.Int    `json:"gas_used"`
	BlockHeight uint64      `json:"block_height"`
	BlockHash   common.Hash `json:"block_hash"`
	BlockIndex  uint64      `json:"block_index"`
	TxIndex     uint64      `json:"tx_index"`
}

type ChainStatusResp struct {
	Status        bool   `json:"status"`
	CurrentBlock  string `json:"current_block"`
	HighestBlock  string `json:"highest_block"`
	StartingBlock string `json:"starting_block"`
}

// type GetBlockChains []*xfsgo.Block
type BlocksResp []*BlockResp

// type transactions []*xfsgo.Transaction
type TransactionsResp []*TransactionResp

type Wallet struct {
	addr    common.Address
	newTime int64
}

type Wallets []*Wallet

func (a Wallets) Len() int {
	return len(a)
}

func (a Wallets) Less(i, j int) bool {
	return a[i].newTime > a[j].newTime
}

func (a Wallets) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func coverTxs2Resp(pending []*xfsgo.Transaction, dst **TransactionsResp) error {
	if len(pending) == 0 {
		return nil
	}
	txs := make([]*TransactionResp, 0)
	for _, item := range pending {
		var txres *TransactionResp
		if err := coverTx2Resp(item, &txres); err != nil {
			return err
		}
		txs = append(txs, txres)
	}
	*dst = (*TransactionsResp)(&txs)
	return nil
}

func coverBlock2Resp(block *xfsgo.Block, dst **BlockResp) error {
	if block == nil {
		return nil
	}
	result := new(BlockResp)
	if err := common.Objcopy(block.Header, result); err != nil {
		return err
	}
	if err := common.Objcopy(block, result); err != nil {
		return err
	}
	result.Hash = block.HeaderHash()
	txs := make([]*TransactionResp, 0)
	for _, item := range block.Transactions {
		var txres *TransactionResp
		if err := coverTx2Resp(item, &txres); err != nil {
			return err
		}
		txs = append(txs, txres)
	}
	if len(txs) > 0 {
		result.Transactions = txs
	}
	*dst = result
	return nil
}

func coverBlockHeader2Resp(block *xfsgo.Block, dst **BlockHeaderResp) error {
	if block == nil {
		return nil
	}
	if err := common.Objcopy(block.Header, &dst); err != nil {
		return err
	}
	result := *dst
	result.Hash = block.HeaderHash()
	return nil
}

func coverTx2Resp(tx *xfsgo.Transaction, dst **TransactionResp) error {
	if tx == nil {
		return nil
	}
	if err := common.Objcopy(tx, &dst); err != nil {
		return err
	}
	result := *dst
	txhash := tx.Hash()
	result.Hash = txhash
	from, err := tx.FromAddr()
	if err != nil {
		return err
	}
    datahex := hex.EncodeToString(tx.Data)
    signhex := hex.EncodeToString(tx.Signature)
    result.Data = "0x" + datahex
    result.Signature = "0x" + signhex
	result.From = from.B58String()
	return nil
}
func coverReceipt(src *ReceiptResp, dst **ReceiptResp) error {
	if src == nil {
		return nil
	}
	return common.Objcopy(src, &dst)
}

func coverState2Resp(state *xfsgo.StateObj, dst **StateObjResp) error {
	if state == nil {
		return nil
	}
	result := new(StateObjResp)
	extra := state.GetExtra()
	if extra != nil {
		statehex := hex.EncodeToString(state.GetExtra())
		if statehex != "" {
			statehex = "0x" + statehex
			result.Extra = &statehex
		}
	}
	code := state.GetCode()
	if code != nil {
		codehex := hex.EncodeToString(state.GetCode())
		if codehex != "" {
			codehex = "0x" + codehex
			result.Code = &codehex
		}
	}
	stateRoot := state.GetStateRoot()
	if !bytes.Equal(stateRoot[:], common.HashZ[:]) {
		result.StateRoot = &stateRoot
	}
	balance := state.GetBalance()
    if balance == nil {
        balance = big.NewInt(0)
    }
    balanceTextm := balance.Text(10)
	result.Balance = &balanceTextm
	result.Nonce = state.GetNonce()
	*dst = result
	return nil
}

type TransactionGasParams struct {
	Value    string `json:"value"`
	GasLimit string `json:"gas_limit"`
	GasPrice string `json:"gas_price"`
}

type StringRawTransaction struct {
	Version   string `json:"version"`
	To        string `json:"to"`
	Value     string `json:"value"`
	Data      string `json:"data"`
	GasLimit  string `json:"gas_limit"`
	GasPrice  string `json:"gas_price"`
	Signature string `json:"signature"`
	Nonce     string `json:"nonce"`
}

func (t *StringRawTransaction) String() string {
	jsondata, err := json.Marshal(t)
	if err != nil {
		panic(err)
	}
	return string(jsondata)
}

func CoverTransaction(r *StringRawTransaction) (*xfsgo.Transaction, error) {
	version, err := strconv.ParseInt(r.Version, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version: %s", err)
	}
	signature := common.Hex2bytes(r.Signature)
	if signature == nil || len(signature) < 1 {
		return nil, fmt.Errorf("failed to parse signature: %s", err)
	}
	toaddr := common.ZeroAddr
	if r.To != "" {
		toaddr = common.StrB58ToAddress(r.To)
		if !crypto.VerifyAddress(toaddr) {
			return nil, fmt.Errorf("failed to verify 'to' address: %s", r.To)
		}
	} else if r.Data == "" {
		return nil, fmt.Errorf("failed to parse 'to' address")
	}
	gasprice, ok := new(big.Int).SetString(r.GasPrice, 10)
	if !ok {
		return nil, fmt.Errorf("failed to parse gasprice")
	}
	gaslimit, ok := new(big.Int).SetString(r.GasLimit, 10)
	if !ok {
		return nil, fmt.Errorf("failed to parse gasprice")
	}
	data := common.Hex2bytes(r.Data)
	nonce, err := strconv.ParseInt(r.Nonce, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse nonce: %s", err)
	}
	value, ok := new(big.Int).SetString(r.Value, 10)
	if !ok {
		return nil, fmt.Errorf("failed to parse value")
	}
	return xfsgo.NewTransactionByStd(&xfsgo.StdTransaction{
		Version:   uint32(version),
		To:        toaddr,
		GasPrice:  gasprice,
		GasLimit:  gaslimit,
		Data:      data,
		Nonce:     uint64(nonce),
		Value:     value,
		Signature: signature,
	}), nil
}

func ParseTransactionGasParams(r *TransactionGasParams) (*xfsgo.StdTransaction, error) {
	gasprice, ok := new(big.Int).SetString(r.GasPrice, 10)
	if !ok && r.GasPrice != "" {
		return nil, fmt.Errorf("failed to parse gasprice")
	}
	gaslimit, ok := new(big.Int).SetString(r.GasLimit, 10)
	if !ok && r.GasLimit != "" {
		return nil, fmt.Errorf("failed to parse gaslimit")
	}
	value, ok := new(big.Int).SetString(r.Value, 10)
	if !ok && r.Value != "" {
		return nil, fmt.Errorf("failed to parse value")
	}
	return &xfsgo.StdTransaction{
		GasPrice: gasprice,
		GasLimit: gaslimit,
		Value:    value,
	}, nil
}
