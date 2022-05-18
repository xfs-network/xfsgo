package miner

import (
	"testing"
	"time"
	"xfsgo"
	"xfsgo/common"
	"xfsgo/test"
)

//func TestMinerStartStop(t *testing.T) {
//	miner := createMiner(t)
//	waitForMiningState(t, miner, false)
//	miner.Start(test.TestMinerWorkers)
//	waitForMiningState(t, miner, true)
//	miner.Stop()
//	waitForMiningState(t, miner, false)
//}
//
//func TestSetWorkers(t *testing.T) {
//	miner := createMiner(t)
//	waitForMiningState(t, miner, false)
//	miner.Start(test.TestMinerWorkers)
//	miner.SetWorkers(uint32(5))
//	waitForMiningState(t, miner, true)
//	miner.Stop()
//	waitForMiningState(t, miner, false)
//}

func waitForMiningState(t *testing.T, m *Miner, mining bool) {
	t.Helper()
	var state bool
	for i := 0; i < 100; i++ {
		time.Sleep(1 * time.Second)
		if state = m.GetMinStatus(); state == mining {
			return
		}
	}
	t.Fatalf("Mining() == %t, want %t", state, mining)
}

func createMiner(t *testing.T) *Miner {

	stateDb := test.NewMemStorage()

	chainDb := test.NewMemStorage()

	extraDb := test.NewMemStorage()

	event := xfsgo.NewEventBus()
	if _, err := xfsgo.WriteTestGenesisBlock(test.TestGenesisBits, stateDb, chainDb); err != nil {
		t.Error(err)
		return nil
	}
	bc, err := xfsgo.NewBlockChainN(stateDb, chainDb, extraDb, event, false)
	if err != nil {
		t.Error(err)
		return nil
	}

	txPool := xfsgo.NewTxPool(nil, bc.CurrentStateTree, bc.LatestGasLimit, test.TestTxPoolGasPrice, event)
	config := &Config{
		Coinbase:   common.Hex2Address(test.TestMinerCoinbase),
		Numworkers: test.TestMinerWorkers,
	}

	return NewMiner(config, stateDb, bc, event, txPool, test.TestTxPoolGasPrice, test.TestTxPoolGasLimit)

}
