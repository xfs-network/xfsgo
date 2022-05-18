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
	"errors"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"
	"xfsgo/common"

	"github.com/sirupsen/logrus"
)

const (
	maxQueued = 64 // max limit of queued txs per address
)

var (
	valueErr         = errors.New("value cannot be less than zero")
	gasPriceErr      = errors.New("gas price too low")
	gasLimitOutErr   = errors.New("gas limit exceeds block gas limit")
	invalidSenderErr = errors.New("invalid sender")
	nonceErr         = errors.New("nonce too low")
	balanceErr       = errors.New("account not enough balance")
	gasLimitErr      = errors.New("gas limit too low")
	// ErrUnderpriced is returned if a transaction's gas price is below the minimum
	// configured for the transaction pool.
	ErrUnderpriced = errors.New("transaction underpriced")
	// ErrReplaceUnderpriced is returned if a transaction is attempted to be replaced
	// with a different one without the required price bump.
	ErrReplaceUnderpriced = errors.New("replacement transaction underpriced")
)

type stateFn func() *StateTree
type gasLimitFn func() *big.Int

type TxPoolConfig struct {
	TxPoolMaxSize    uint64
	PriceBump        int64
	Lifetime         time.Duration
	EvictionInterval time.Duration // Time interval to check for evictable transactions
}

func defaultTxPoolConfig() *TxPoolConfig {
	return &TxPoolConfig{
		TxPoolMaxSize:    100,
		PriceBump:        10,
		Lifetime:         3 * time.Hour,
		EvictionInterval: time.Minute,
	}
}

// TxPool contains all currently known transactions. Transactions
// enter the pool when they are received from the network or submitted
// locally. They exit the pool when they are included in the blockchain.
//
// The pool separates processable transactions (which can be applied to the
// current state) and waiting transactions. Transactions move between those
// two states over time as they are received and processed.
type TxPool struct {
	quit         chan bool
	config       *TxPoolConfig
	currentState stateFn // The state function which will allow us to do some pre checkes
	pendingState *ManagedState
	eventBus     *EventBus
	mu           sync.RWMutex
	gasLimitFn   gasLimitFn // The current gas limit function callback
	minGasPrice  *big.Int
	priced       []*Transaction
	pending      map[common.Hash]*Transaction // processable transactions
	queue        map[common.Address]map[common.Hash]*Transaction
	beats        map[common.Address]time.Time // Last heartbeat from each known account
	wg           sync.WaitGroup               // for shutdown sync
}

// NewTxPool creates a new transaction pool to gather, sort and filter inbound
// transactions from the network.
func NewTxPool(config *TxPoolConfig, currentStateFn stateFn, gasLimitFn gasLimitFn, gasPrice *big.Int, eventBus *EventBus) *TxPool {
	pool := &TxPool{
		pending:      make(map[common.Hash]*Transaction),
		queue:        make(map[common.Address]map[common.Hash]*Transaction),
		quit:         make(chan bool),
		beats:        make(map[common.Address]time.Time),
		priced:       make([]*Transaction, 0),
		gasLimitFn:   gasLimitFn,
		config:       defaultTxPoolConfig(),
		minGasPrice:  gasPrice,
		currentState: currentStateFn,
		pendingState: NewManageState(currentStateFn()),
	}

	if config != nil {
		pool.config = config
	}
	pool.eventBus = eventBus
	pool.resetState()
	pool.wg.Add(2)
	go pool.eventLoop()
	go pool.expirationLoop()
	return pool
}

func (pool *TxPool) GetGasLimit() *big.Int {
	return pool.gasLimitFn()
}

func (pool *TxPool) GetGasPrice() *big.Int {
	return pool.minGasPrice
}

func (pool *TxPool) add(tx *Transaction) error {
	txHash := tx.Hash()
	if pool.pending[txHash] != nil {
		return fmt.Errorf("know transaction (%s)", txHash.Hex())
	}
	if err := pool.validateTx(tx); err != nil {
		return err
	}

	// If the transaction pool is full, discard underpriced transactions
	if pool.GetTxPoolSize() >= pool.config.TxPoolMaxSize {
		// If the new transaction is underpriced, don't accept it
		if tx.GasPrice.Cmp(pool.priced[len(pool.priced)-1].GasPrice) < 1 {
			// Discarding underpriced transaction
			return ErrUnderpriced
		}
		// New transaction is better than our worse ones, make room for it
		drop := int(pool.GetTxPoolSize()) - int(pool.config.TxPoolMaxSize-1)
		dropmap := pool.priced[len(pool.priced)-drop:]
		for _, tx := range dropmap {
			pool.RemoveTx(tx)
		}
		pool.priced = pool.priced[:len(pool.priced)-drop]
		// pool.updatePriced(newPriced)
	}

	// Meet the price increase range and replace the transaction
	if transfer := pool.pending[txHash]; transfer != nil {
		if transfer.Nonce == tx.Nonce {
			threshold := new(big.Int).Div(new(big.Int).Mul(tx.GasPrice, big.NewInt(100+pool.config.PriceBump)), big.NewInt(100))
			if threshold.Cmp(tx.GasPrice) >= 0 {
				return ErrReplaceUnderpriced
			} else {
				pool.pending[txHash] = tx
				tx := pool.GetPendingTxs()
				SortByPriceAndNonce(tx)
			}
		}
	}

	pool.appendQueueTx(txHash, tx)
	pool.addPriced(tx)
	go pool.eventBus.Publish(TxPreEvent{Tx: tx})
	return nil
}

func (pool *TxPool) addPriced(tx *Transaction) {
	pool.priced = append(pool.priced, tx)
	sort.Sort(TxByPrice(pool.priced))
}

// func (pool *TxPool) updatePriced(tx []*Transaction) {
// 	pool.priced = tx
// 	sort.Sort(TxByPrice(pool.priced))
// }

// expirationLoop is a loop that periodically iterates over all accounts with
// queued transactions and drop all that have been inactive for a prolonged amount
// of time.
func (pool *TxPool) expirationLoop() {
	defer pool.wg.Done()

	evict := time.NewTicker(pool.config.EvictionInterval)
	defer evict.Stop()

	for {
		select {
		case <-evict.C:
			pool.mu.Lock()
			for addr := range pool.queue {
				// Any non-locals old enough should be removed
				if time.Since(pool.beats[addr]) > pool.config.Lifetime {
					for _, tx := range pool.queue[addr] {
						pool.RemoveTx(tx)
					}
				}
			}
			pool.mu.Unlock()
		case <-pool.quit:
			return
		}
	}
}

func (pool *TxPool) validateTx(tx *Transaction) error {
	var (
		from common.Address
		err  error
	)
	// Drop transactions under our own minimal accepted gas price
	if pool.minGasPrice.Cmp(tx.GasPrice) > 0 {
		return gasPriceErr
	}
	if from, err = tx.FromAddr(); err != nil {
		return invalidSenderErr
	}
	logrus.Debugf("Validation transaction: hash=%x, from=%s", tx.Hash(), from.B58String())
	if !pool.currentState().HashAccount(from) {
		return balanceErr
	}

	// Last but not least check for nonce errors
	if pool.currentState().GetNonce(from) > tx.Nonce {
		return nonceErr
	}

	// Check the transaction doesn't exceed the current
	// block limit gas.
	// tx gasLimit compare txpool gasLimit
	if pool.gasLimitFn().Cmp(tx.GasLimit) < 0 {
		return gasLimitOutErr
	}
	if tx.Value.Sign() < 0 {
		return valueErr
	}
	if pool.currentState().GetBalance(from).Cmp(tx.Cost()) < 0 {
		return balanceErr
	}
	if tx.GasLimit.Cmp(common.CalcTxInitialCost(tx.Data)) < 0 {
		return gasLimitErr
	}
	return nil
}

// validatePool checks entire the pending trsactions in the tx pool
// whether they are valid according to the consensus
// rules and adheres to some limits of the local node (price and size).

func (pool *TxPool) validatePool() {
	//get the current state of the  tx pool
	state := pool.currentState()
	// traversals all peeding transactions
	// delete pending transactions that has expired (low nonce)
	for hash, tx := range pool.pending {
		from, _ := tx.FromAddr()
		if state.GetNonce(from) > tx.Nonce {
			delete(pool.pending, hash)
		}
	}
}

// func (pool *TxPool) getNonceAt(address common.Address) uint64 {
// 	state := pool.currentState()
// 	return state.GetNonce(address)
// }

func (pool *TxPool) checkQueue() {
	state := pool.pendingState

	var addq txQueue
	for address, txs := range pool.queue {
		// guessed nonce is the nonce currently kept by the tx pool (pending state)
		guessedNonce := state.GetNonce(address)
		// true nonce is the nonce known by the last state
		trueNonce := pool.currentState().GetNonce(address)
		addq := addq[:0]
		for hash, tx := range txs {
			if tx.Nonce < trueNonce {
				// Drop queued transactions whose nonce is lower than
				// the account nonce because they have been processed.
				delete(txs, hash)
			} else {
				// Collect the remaining transactions for the next pass.
				addq = append(addq, txQueueEntry{hash, address, tx})
			}
		}
		// Find the next consecutive nonce range starting at the
		// current account nonce.
		sort.Sort(addq)
		for i, e := range addq {
			// start deleting the transactions from the queue if they exceed the limit
			if i > maxQueued {
				delete(pool.queue[address], e.hash)
				continue
			}

			if e.Nonce > guessedNonce {
				if len(addq)-i > maxQueued {
					for j := i + maxQueued; j < len(addq); j++ {
						delete(txs, addq[j].hash)
					}
				}
				break
			}
			delete(txs, e.hash)
			pool.addTx(e.hash, address, e.Transaction)
		}
		// Delete the entire queue entry if it became empty.
		if len(txs) == 0 {
			delete(pool.queue, address)
		}
	}
}

func (pool *TxPool) appendQueueTx(hash common.Hash, tx *Transaction) {
	from, _ := tx.FromAddr()
	if pool.queue[from] == nil {
		pool.queue[from] = make(map[common.Hash]*Transaction)
	}

	if transfer := pool.queue[from][hash]; transfer != nil {
		if transfer.Nonce == tx.Nonce {
			threshold := new(big.Int).Div(new(big.Int).Mul(tx.GasPrice, big.NewInt(100+10)), big.NewInt(100))
			if threshold.Cmp(tx.GasPrice) >= 0 {
				logrus.Errorf(ErrReplaceUnderpriced.Error())
				return
			} else {
				delete(pool.queue[from], hash)
				pool.queue[from][tx.Hash()] = tx
				return
			}
		}
	}
	pool.queue[from][hash] = tx
}

// addTx will add a transaction to the pending (processable queue) list of transactions
func (pool *TxPool) addTx(hash common.Hash, addr common.Address, tx *Transaction) {

	if _, ok := pool.pending[hash]; !ok {
		pool.pending[hash] = tx

		// Increment the nonce on the pending state. This can only happen if
		// the nonce is +1 to the previous one.
		pool.beats[addr] = time.Now()
		pool.pendingState.SetNonce(addr, tx.Nonce+1)
		// Notify the subscribers. This events is posted in a goroutine
		// because it's possible that somewhere during the post "Remove transaction"
		// gets called which will then wait for the global tx pool lock and deadlock.
		go pool.eventBus.Publish(TxPreEvent{Tx: tx})
	}
}

func (pool *TxPool) resetState() {
	// reset state manager of peeding transactions
	pool.pendingState = NewManageState(pool.currentState())

	// check tx pool and update peeding queue
	pool.validatePool()

	// Loop over the pending transactions and base the nonce of the new
	// pending transaction set.
	for _, tx := range pool.pending {
		if addr, err := tx.FromAddr(); err == nil {
			// Set the nonce. Transaction nonce can never be lower
			// than the state nonce; validatePool took care of that.
			if pool.pendingState.GetNonce(addr) < tx.Nonce {
				pool.pendingState.SetNonce(addr, tx.Nonce+1)
			}

		}
	}

	// Check the queue and move transactions over to the pending if possible
	// or remove those that have become invalid
	pool.checkQueue()
}

func (pool *TxPool) Forward(address common.Address, nonce uint64) []*Transaction {
	removed := make([]*Transaction, 0)
	for _, v := range pool.queue[address] {
		if nonce > v.Nonce {
			removed = append(removed, v)
		}
	}
	return removed
}

func (pool *TxPool) Add(tx *Transaction) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	err := pool.add(tx)
	if err == nil {
		// check and validate the queueue
		pool.checkQueue()
	}
	// pool.promoteExecutables([]common.Address{from})
	return err
}

// eventLoop is the transaction pool's main event loop, waiting for and reacting to
// outside blockchain events
func (pool *TxPool) eventLoop() {
	defer pool.wg.Done()
	chainHeadEventSub := pool.eventBus.Subscript(ChainHeadEvent{})
	GasPriceChangedSub := pool.eventBus.Subscript(GasPriceChanged{})
	defer func() {
		GasPriceChangedSub.Unsubscribe()
		chainHeadEventSub.Unsubscribe()
	}()
	for {
		select {
		case e := <-chainHeadEventSub.Chan():
			pool.mu.Lock()
			// handle ChainHeadEvent
			// update the state of tx pool when receive blockchain event to update the latest state
			event := e.(ChainHeadEvent)
			block := event.Block
			_ = block
			pool.resetState()
			pool.mu.Unlock()
		case e := <-GasPriceChangedSub.Chan():
			event := e.(GasPriceChanged)
			pool.mu.Lock()
			pool.minGasPrice = event.Price
			pool.mu.Unlock()

		}
	}
}

func (pool *TxPool) GetPendingTxs() []*Transaction {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	pool.checkQueue()
	pool.validatePool()
	txs := make([]*Transaction, 0)
	for _, v := range pool.pending {
		txs = append(txs, v)
	}
	return txs
}

func (pool *TxPool) GetQueues() []*Transaction {
	pool.mu.RLock()
	defer pool.mu.RUnlock()
	txs := make([]*Transaction, 0)
	for _, v := range pool.queue {
		for _, vs := range v {
			txs = append(txs, vs)
		}
	}
	return txs
}

func (pool *TxPool) GetTransaction(tranHash common.Hash) *Transaction {
	pool.mu.RLock()
	defer pool.mu.RUnlock()
	if tx, exists := pool.pending[tranHash]; exists {
		return tx
	}
	for _, v := range pool.queue {
		if tx, exists := v[tranHash]; exists {
			return tx
		}
	}
	return nil
}

func (pool *TxPool) State() *ManagedState {
	pool.mu.RLock()
	defer pool.mu.RUnlock()
	return pool.pendingState
}

// func (pool *TxPool) ModifyTranGas(gasLimit, gasPrice *big.Int, hash string) error {
// 	tran := pool.GetTransaction(hash)
// 	tran.GasLimit = gasLimit
// 	tran.GasLimit = gasPrice
// 	return pool.Add(tran)
// }

// GetPendingTxsSize get txpool pending list amount
func (pool *TxPool) GetPendingTxsSize() int {
	return len(pool.pending)
}

// GetQueueSize get txpool queue list amount
func (pool *TxPool) GetQueueSize() int {
	var queuesize = 0
	for address := range pool.queue {
		queuesize = queuesize + len(pool.queue[address])
	}
	return queuesize
}

// GetTxPoolSize get txpool all limit amount
func (pool *TxPool) GetTxPoolSize() uint64 {
	return uint64(pool.GetPendingTxsSize() + pool.GetQueueSize())
}

func (pool *TxPool) RemoveTransactions(txs []*Transaction) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	for _, tx := range txs {
		pool.RemoveTx(tx)
	}
}

func (pool *TxPool) RemoveTx(transfer *Transaction) {

	txHash := transfer.Hash()
	// delete from pending pool
	delete(pool.pending, txHash)

	// delete from queue
	for address, txs := range pool.queue {
		if _, ok := txs[txHash]; ok {
			if len(txs) == 1 {
				// if only one tx, remove entire address entry.
				delete(pool.queue, address)
			} else {
				delete(txs, txHash)
			}
			break
		}
	}
	// Update the account nonce if needed
	if nonce := transfer.Nonce; pool.pendingState.GetNonce(transfer.FromAddress()) > nonce {
		pool.pendingState.SetNonce(transfer.FromAddress(), nonce)
	}
}

type txQueue []txQueueEntry

type txQueueEntry struct {
	hash common.Hash
	addr common.Address
	*Transaction
}

func (q txQueue) Len() int           { return len(q) }
func (q txQueue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q txQueue) Less(i, j int) bool { return q[i].Nonce < q[j].Nonce }
