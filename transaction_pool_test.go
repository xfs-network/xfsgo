package xfsgo

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"xfsgo/common"
	"xfsgo/crypto"
	"xfsgo/test"
)

func transaction(val string, nonce uint64, gasLimit *big.Int, key *ecdsa.PrivateKey) *Transaction {

	stdTxVal, _ := common.BaseCoin2Atto(val)
	if gasLimit == nil {
		gasLimit = test.TestTxGasLimit
	}

	stdTx := new(StdTransaction)
	stdTx.GasPrice = test.TestTxGasPrice
	stdTx.GasLimit = gasLimit
	stdTx.To = common.Address{}
	stdTx.Value = stdTxVal
	stdTx.Nonce = uint64(0)
	tx := NewTransactionByStd(stdTx)
	_ = tx.SignWithPrivateKey(key)
	return tx
}

func setupTxPool() (*TxPool, *ecdsa.PrivateKey) {
	stateDb := test.NewMemStorage()

	chainDb := test.NewMemStorage()

	extraDb := test.NewMemStorage()

	event := NewEventBus()
	if _, err := WriteTestGenesisBlock(test.TestGenesisBits, stateDb, chainDb); err != nil {
		fmt.Printf("WriteTestGenesisBlock Error:%v\n", err)
		return nil, nil
	}
	bc, err := NewBlockChainN(stateDb, chainDb, extraDb, event, false)
	if err != nil {
		fmt.Printf("NewBlockChain Error:%v\n", err)
		return nil, nil
	}
	key, err := crypto.GenPrvKey()
	if err != nil {
		fmt.Printf("GenPrvKey Error:%v\n", err)
		return nil, nil
	}

	txPool := NewTxPool(nil, bc.CurrentStateTree, bc.LatestGasLimit, test.TestTxPoolGasPrice, event)
	return txPool, key
}

//func TestInvalidTx(t *testing.T) {
//	pool, key := setupTxPool()
//	tx := transaction("1", 0, nil, key)
//	from, _ := tx.FromAddr()
//	attoBal, _ := common.BaseCoin2Atto("100")
//	pool.currentState().AddBalance(from, attoBal)
//
//	if err := pool.Add(tx); err != nil {
//		t.Errorf("add txpool Error:%v", err)
//		return
//	}
//	fmt.Println("add successfully")
//
//}
//
//func TestTransactionQueue(t *testing.T) {
//	pool, key := setupTxPool()
//
//	tx := transaction("1", 0, nil, key)
//	from, _ := tx.FromAddr()
//	attoBal, _ := common.BaseCoin2Atto("100")
//	pool.currentState().AddBalance(from, attoBal)
//	pool.appendQueueTx(tx.Hash(), tx)
//
//	pool.checkQueue()
//	if len(pool.pending) != 1 {
//		t.Error("expected valid txs to be 1 is", len(pool.pending))
//	}
//
//	tx = transaction("2", 1, nil, key)
//	from, _ = tx.FromAddr()
//	pool.currentState().AddNonce(from, 2)
//	pool.appendQueueTx(tx.Hash(), tx)
//	pool.checkQueue()
//	if _, ok := pool.pending[tx.Hash()]; ok {
//		t.Error("expected transaction to be in tx pool")
//	}
//
//	if len(pool.queue[from]) > 0 {
//		t.Error("expected transaction queue to be empty. is", len(pool.queue[from]))
//	}
//}
//
//func TestRemoveTx(t *testing.T) {
//
//	pool, key := setupTxPool()
//	tx := transaction("1", 0, nil, key)
//
//	from, _ := tx.FromAddr()
//	attoBal, _ := common.BaseCoin2Atto("100")
//	pool.currentState().AddBalance(from, attoBal)
//	pool.appendQueueTx(tx.Hash(), tx)
//	pool.addTx(tx.Hash(), from, tx)
//	if len(pool.queue) != 1 {
//		t.Error("expected queue to be 1, got", len(pool.queue))
//	}
//
//	if len(pool.pending) != 1 {
//		t.Error("expected txs to be 1, got", len(pool.pending))
//	}
//
//	pool.RemoveTx(tx.Hash())
//
//	if len(pool.queue) > 0 {
//		t.Error("expected queue to be 0, got", len(pool.queue))
//	}
//
//	if len(pool.pending) > 0 {
//		t.Error("expected txs to be 0, got", len(pool.pending))
//	}
//
//}
//
//func TestTransactionDoubleNonce(t *testing.T) {
//	pool, key := setupTxPool()
//	pool.resetState()
//
//	addr := crypto.DefaultPubKey2Addr(key.PublicKey)
//	attoBal, _ := common.BaseCoin2Atto("100")
//	pool.currentState().AddBalance(addr, attoBal)
//	tx := transaction("1", 0, nil, key)
//	tx2 := transaction("1", 0, big.NewInt(26000), key)
//	if err := pool.Add(tx); err != nil {
//		t.Error("didn't expect error", err)
//	}
//	if err := pool.Add(tx2); err != nil {
//		t.Error("didn't expect error", err)
//	}
//
//	fmt.Printf("len:%v\n", len(pool.pending))
//
//	pool.checkQueue()
//	if len(pool.pending) != 2 {
//		t.Error("expected 2 pending txs. Got", len(pool.pending))
//	}
//}
//
//func TestMissingNonce(t *testing.T) {
//	pool, key := setupTxPool()
//	addr := crypto.DefaultPubKey2Addr(key.PublicKey)
//	attoBal, _ := common.BaseCoin2Atto("100")
//	pool.currentState().AddBalance(addr, attoBal)
//	tx := transaction("1", 1, nil, key)
//	if err := pool.add(tx); err != nil {
//		t.Error("didn't expect error", err)
//	}
//	if len(pool.pending) != 0 {
//		t.Error("expected 0 pending transactions, got", len(pool.pending))
//	}
//	if len(pool.queue[addr]) != 1 {
//		t.Error("expected 1 queued transaction, got", len(pool.queue[addr]))
//	}
//}
//
//func TestNonceRecovery(t *testing.T) {
//	const n = 10
//	pool, key := setupTxPool()
//
//	addr := crypto.DefaultPubKey2Addr(key.PublicKey)
//	attoBal, _ := common.BaseCoin2Atto("100")
//	pool.currentState().AddNonce(addr, n)
//	pool.currentState().AddBalance(addr, attoBal)
//	pool.resetState()
//	tx := transaction("1", n, nil, key)
//	if err := pool.Add(tx); err != nil {
//		t.Error(err)
//	}
//	// simulate some weird re-order of transactions and missing nonce(s)
//	pool.currentState().AddNonce(addr, n-1)
//	pool.resetState()
//	if fn := pool.pendingState.GetNonce(addr); fn != n+1 {
//		t.Errorf("expected nonce to be %d, got %d", n+1, fn)
//	}
//}
