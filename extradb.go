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
	"bytes"
	"encoding/binary"
	"encoding/json"
	"xfsgo/common"
	"xfsgo/common/rawencode"
	"xfsgo/storage/badger"

	"github.com/sirupsen/logrus"
)

var (
	txPre                = []byte("tx:")
	txIndexPre           = []byte("txIndex:")
	receiptPre           = []byte("receipt:")
	blockReceiptsPre     = []byte("bh:receipts:")
	blockTransactionsPre = []byte("bh:Transactions:")
)

type extraDB struct {
	storage badger.IStorage
}

func newExtraDB(db badger.IStorage) *extraDB {
	tdb := &extraDB{
		storage: db,
	}
	return tdb
}

func (db *extraDB) newWriteBatch() *badger.StorageWriteBatch {
	return db.storage.NewWriteBatch()
}

func (db *extraDB) commitBatch(batch *badger.StorageWriteBatch) error {
	return db.storage.CommitWriteBatch(batch)
}

type TxIndex struct {
	BlockHash  common.Hash `json:"block_hash"`
	BlockIndex uint64      `json:"block_index"`
	Index      uint64      `json:"index"`
}

func (t *TxIndex) Encode() ([]byte, error) {
	return json.Marshal(t)
}

func (t *TxIndex) Decode(data []byte) error {
	return json.Unmarshal(data, t)
}

// GetReceipt get Receipt by Receipt hash from extra db
func (db *extraDB) GetReceipt(txHash common.Hash) *Receipt {
	key := append(receiptPre, txHash.Bytes()...)
	data, err := db.storage.GetData(key)
	if err != nil {
		return nil
	}
	r := &Receipt{}
	if err = rawencode.Decode(data, r); err != nil {
		return nil
	}
	return r
}

// GetReceipt get Receipt by Receipt hash from extra db
func (db *extraDB) GetReceiptByHashIndex(txHash common.Hash) *TxIndex {
	key := append(txIndexPre, txHash.Bytes()...)
	data, err := db.storage.GetData(key)
	if err != nil {
		return nil
	}
	r := &TxIndex{}
	if err = rawencode.Decode(data, r); err != nil {
		return nil
	}
	return r
}

// GetBlockReceiptsByBHash get Receipts by blockheader hash from extra db
func (db *extraDB) GetBlockReceiptsByBHash(hash common.Hash) []*Receipt {
	key := append(blockReceiptsPre, hash.Bytes()...)
	data, err := db.storage.GetData(key)
	if err != nil {
		return nil
	}
	tmp := make([]*Receipt, 0)
	buf := bytes.NewBuffer(data)
	for {
		var dataLenBuf [4]byte
		_, err = buf.Read(dataLenBuf[:])
		if err != nil {
			break
		}
		dataLen := binary.LittleEndian.Uint32(dataLenBuf[:])
		var dataBuf = make([]byte, dataLen)
		_, err = buf.Read(dataBuf[:])
		if err != nil {
			break
		}
		r := &Receipt{}
		if err = rawencode.Decode(dataBuf, r); err != nil {
			return nil
		}
		tmp = append(tmp, r)
	}
	return tmp
}

// GetTransactionByTxHash get Transaction by Transaction hash from extra db
func (db *extraDB) GetTransactionByTxHash(txHash common.Hash) *Transaction {
	key := append(txPre, txHash.Bytes()...)
	txData, err := db.storage.GetData(key)
	if err != nil {
		return nil
	}
	tx := &Transaction{}
	if err = rawencode.Decode(txData, tx); err != nil {
		return nil
	}
	return tx
}

// GetBlockTransactionsByBHash get Transactions by blockheader hash from extra db
func (db *extraDB) GetBlockTransactionsByBHash(hash common.Hash) []*Transaction {
	key := append(blockTransactionsPre, hash.Bytes()...)
	data, err := db.storage.GetData(key)
	if err != nil {
		return nil
	}
	transactions := make([]*Transaction, 0)
	buf := bytes.NewBuffer(data)
	for {
		var dataLenBuf [4]byte
		_, err = buf.Read(dataLenBuf[:])
		if err != nil {
			break
		}
		dataLen := binary.LittleEndian.Uint32(dataLenBuf[:])
		var dataBuf = make([]byte, dataLen)
		_, err = buf.Read(dataBuf[:])
		if err != nil {
			break
		}
		r := &Transaction{}
		if err = rawencode.Decode(dataBuf, r); err != nil {
			return nil
		}
		transactions = append(transactions, r)
	}
	return transactions
}

// WriteReceipts Write Receipt with receipt hash
func (db *extraDB) WriteReceiptsWithRecHash(receipts []*Receipt) error {
	for _, receipt := range receipts {
		data, err := rawencode.Encode(receipt)
		if err != nil {
			return err
		}
		key := append(receiptPre, receipt.TxHash.Bytes()...)
		if err = db.storage.SetData(key, data); err != nil {
			return err
		}
	}
	return nil
}

// WriteReceipts Write Receipts with blockheader hash
func (db *extraDB) WriteBlockReceipts(bHash common.Hash, receipts []*Receipt) error {
	buf := bytes.NewBuffer(nil)
	for _, receipt := range receipts {
		data, err := rawencode.Encode(receipt)
		if err != nil {
			return err
		}
		var dataLenBuf [4]byte
		dataLen := uint32(len(data))
		binary.LittleEndian.PutUint32(dataLenBuf[:], dataLen)
		buf.Write(dataLenBuf[:])
		buf.Write(data)
	}
	key := append(blockReceiptsPre, bHash.Bytes()...)
	if err := db.storage.SetData(key, buf.Bytes()); err != nil {
		return err
	}
	return nil
}

// WriteBlockTransactionsWithHash write block's transactions linked blockheader's hash to extra db
func (db *extraDB) WriteBlockTransactionsWithBHash(bHash common.Hash, transactions []*Transaction) error {
	buf := bytes.NewBuffer(nil)
	for _, tx := range transactions {
		data, err := rawencode.Encode(tx)
		if err != nil {
			return err
		}
		var dataLenBuf [4]byte
		dataLen := uint32(len(data))
		binary.LittleEndian.PutUint32(dataLenBuf[:], dataLen)
		buf.Write(dataLenBuf[:])
		buf.Write(data)
	}

	key := append(blockTransactionsPre, bHash.Bytes()...)
	if err := db.storage.SetData(key, buf.Bytes()); err != nil {
		return err
	}
	return nil
}

// WriteBlockTransactions write transactions linked blockheader hash and height and tx index to extra db
func (db *extraDB) WriteBlockTransactionsWithTxIndex(bHash common.Hash, height uint64, transactions []*Transaction) error {
	for i, tx := range transactions {
		txHash := tx.Hash()
		index := &TxIndex{
			BlockHash:  bHash,
			BlockIndex: height,
			Index:      uint64(i),
		}
		indexData, err := rawencode.Encode(index)
		if err != nil {
			return err
		}
		indexKey := append(txIndexPre, txHash[:]...)
		if err = db.storage.SetData(indexKey, indexData); err != nil {
			return err
		}
	}
	return nil
}

// WriteBlockTransactions write transactions linked tx hash to extra db
func (db *extraDB) WriteBlockTransactionWithTxHash(transactions []*Transaction) error {
	for _, tx := range transactions {
		data, err := rawencode.Encode(tx)
		if err != nil {
			return err
		}
		txHash := tx.Hash()
		key := append(txPre, txHash.Bytes()...)
		if err = db.storage.SetData(key, data); err != nil {
			return err
		}
	}
	return nil
}

// DelReceipts del block's receipt by  receipt hashes extraDB
func (db *extraDB) DelReceipts(receipts []*Receipt) error {
	for _, receipt := range receipts {
		key := append(receiptPre, receipt.TxHash.Bytes()...)
		if err := db.storage.DelData(key); err != nil {
			return err
		}
	}
	return nil
}

// DelBlockReceiptsByBHash del block's receipts by block header hash from extraDB
func (db *extraDB) DelBlockReceiptsByBHash(hash common.Hash) error {
	key := append(blockReceiptsPre, hash.Bytes()...)
	if err := db.storage.DelData(key); err != nil {
		return err
	}
	return nil
}

// DelTransaction remove transaction data By tx hash
func (db *extraDB) DelTransactionByTxHash(txHash common.Hash) error {
	txKey := append(txPre, txHash[:]...)
	if err := db.storage.DelData(txKey); err != nil {
		return err
	}

	return nil
}

// DelBlockTransactionsWithBHash del block's transactions linked blockheader's hash from extra db
func (db *extraDB) DelBlockTransactionsByBHash(bHash common.Hash) error {
	key := append(blockTransactionsPre, bHash.Bytes()...)
	if err := db.storage.DelData(key); err != nil {
		return err
	}
	return nil
}

// DelBlockTransactionsByBIndex del transactions linked blockheader hash and height and tx index from extra db
func (db *extraDB) DelBlockTransactionsByBIndex(bHash common.Hash, height uint64, transactions []*Transaction) error {
	for i, tx := range transactions {
		txHash := tx.Hash()
		txdata, err := rawencode.Encode(tx)
		if err != nil {
			logrus.Errorf("Write block transactions err: %v", err)
			return err
		}
		txKey := append(txPre, txHash[:]...)
		if err = db.storage.SetData(txKey, txdata); err != nil {
			return err
		}
		index := &TxIndex{
			BlockHash:  bHash,
			BlockIndex: height,
			Index:      uint64(i),
		}
		indexData, err := rawencode.Encode(index)
		if err != nil {
			return err
		}
		indexKey := append(txIndexPre, txHash[:]...)
		if err = db.storage.SetData(indexKey, indexData); err != nil {
			return err
		}
	}
	return nil
}

// DelBlockTransactionWithTxHash del transaction linked tx hash from extra db
func (db *extraDB) DelBlockTransactionWithTxHash(tx *Transaction) error {
	txHash := tx.Hash()
	key := append(txPre, txHash.Bytes()...)
	if err := db.storage.DelData(key); err != nil {
		return err
	}
	return nil
}
