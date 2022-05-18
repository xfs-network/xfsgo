package backend

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"sync"
	"testing"
	"time"
	"xfsgo"
	"xfsgo/common"
	"xfsgo/p2p"
	"xfsgo/p2p/discover"
)

type testPackReader struct {
	data  []byte
	mtype uint8
}

func newTestPackReader(mt uint8, data []byte) *testPackReader {
	return &testPackReader{
		mtype: mt,
		data:  data,
	}
}
func (pr *testPackReader) Type() uint8 {
	return pr.mtype
}
func (pr *testPackReader) Read(p []byte) (int, error) { return 0, nil }
func (pr *testPackReader) ReadAll() ([]byte, error) {
	return pr.data, nil
}
func (pr *testPackReader) RawReader() io.Reader {
	return nil
}
func (pr *testPackReader) DataReader() io.Reader {
	return nil
}

type testPeer struct {
	t            *testing.T
	mgr          chainMgr
	head         common.Hash
	height       uint64
	id           discover.NodeId
	readerCh     chan p2p.MessageReader
	ignoreHashes map[common.Hash]struct{}
	ignoresRw    sync.RWMutex
	close        chan struct{}
}

func (p *testPeer) Head() common.Hash {
	return p.head
}
func (p *testPeer) ID() discover.NodeId {
	return p.id
}
func (p *testPeer) Height() uint64 {
	return p.height
}
func (p *testPeer) SetHead(hash common.Hash) {
	p.head = hash
}
func (p *testPeer) SetHeight(height uint64) {
	p.height = height
}
func (p *testPeer) P2PPeer() p2p.Peer { return nil }
func (p *testPeer) Handshake(head common.Hash, height uint64) error {
	return nil
}

func (p *testPeer) RequestHashesFromNumber(from uint64, count uint64) error {
	time.Sleep(3 * time.Second)
	last := p.mgr.GetBlockByNumber(from + count - 1)
	if last == nil {
		bHeader := p.mgr.CurrentBHeader()
		tempBlock := &xfsgo.Block{Header: bHeader, Transactions: nil, Receipts: nil}
		last = tempBlock
		count = last.Height() - from + 1
	}
	if last.Height() < from {
		data, err := json.Marshal(nil)
		if err != nil {
			p.t.Error(err)
		}
		//p.t.Logf("send data: %s", data)
		packet := newTestPackReader(BlockHashesMsg, data)
		p.readerCh <- packet
		return nil
	}
	hashes := []common.Hash{last.HeaderHash()}
	hashes = append(hashes, p.mgr.GetBlockHashesFromHash(last.HeaderHash(), count-1)...)

	for i := 0; i < len(hashes)/2; i++ {
		hashes[i], hashes[len(hashes)-1-i] = hashes[len(hashes)-1-i], hashes[i]
	}
	data, err := json.Marshal(hashes)
	if err != nil {
		p.t.Error(err)
	}
	//p.t.Logf("send data: %s", data)
	packet := newTestPackReader(BlockHashesMsg, data)
	p.readerCh <- packet
	return nil
}
func (p *testPeer) SendBlockHashes(hashes RemoteHashes) error {
	p.t.Logf("SendBlockHashes ... ")
	return nil
}
func (p *testPeer) RequestBlocks(hashes RemoteHashes) error {
	p.t.Logf("Handle request blocks: count=%d", len(hashes))
	time.Sleep(3 * time.Second)
	blocks := make(RemoteBlocks, 0)
	for _, hash := range hashes {
		block := p.mgr.GetBlockByHashWithoutRec(hash)
		if block == nil {
			continue
		}
		newBlock := coverBlock2RemoteBlock(block)
		blocks = append(blocks, newBlock)
	}
	data, err := json.Marshal(blocks)
	if err != nil {
		p.t.Error(err)
	}
	packet := newTestPackReader(BlocksMsg, data)
	p.readerCh <- packet
	return nil
}
func (p *testPeer) SendBlocks(blocks RemoteBlocks) error    { return nil }
func (p *testPeer) SendNewBlock(data *RemoteBlock) error    { return nil }
func (p *testPeer) SendTransactions(data RemoteTxs) error   { return nil }
func (p *testPeer) SendTxhash(data TxHashs) error           { return nil }
func (p *testPeer) SendReceiptsData(data ReceiptsSet) error { return nil }
func (p *testPeer) Close() {
	close(p.close)
}
func (p *testPeer) GetProtocolMsgCh() (chan p2p.MessageReader, error) {
	select {
	case <-p.close:
		return nil, fmt.Errorf("closed")
	default:
	}
	return p.readerCh, nil
}
func (p *testPeer) AddIgnoreHash(hash common.Hash) {
	p.ignoresRw.Lock()
	defer p.ignoresRw.Unlock()
	if _, exists := p.ignoreHashes[hash]; !exists {
		p.ignoreHashes[hash] = struct{}{}
	}
}
func (p *testPeer) HasIgnoreHash(hash common.Hash) (exists bool) {
	p.ignoresRw.RLock()
	defer p.ignoresRw.RUnlock()
	_, exists = p.ignoreHashes[hash]
	return
}
func (p *testPeer) Reset() {
	p.ignoresRw.Lock()
	defer p.ignoresRw.Unlock()
	p.ignoreHashes = make(map[common.Hash]struct{})
}
func (p *testPeer) SendObject(t uint8, obj interface{}) error {
	return nil
}
func (p *testPeer) SendData(t uint8, data []byte) error {
	return nil
}

func newTestPeer(t *testing.T, mgr chainMgr, id discover.NodeId) *testPeer {
	last := mgr.CurrentBHeader()
	return &testPeer{
		t:            t,
		mgr:          mgr,
		head:         last.HeaderHash(),
		height:       last.Height,
		id:           id,
		readerCh:     make(chan p2p.MessageReader),
		ignoreHashes: make(map[common.Hash]struct{}),
		close:        make(chan struct{}),
	}
}

type testChainMgr struct {
	coinbase common.Address
	genesis  *xfsgo.Block
	last     *xfsgo.Block
	blocks   map[common.Hash]*xfsgo.Block
	txs      map[common.Hash]*xfsgo.Transaction
	receipts map[common.Hash]*xfsgo.Receipt
}

func (t *testChainMgr) NewEmptyBlocks(num int) {
	for i := 0; i < num; i++ {
		t.NewBlock(nil, nil)
	}
}
func (t *testChainMgr) NewEmptyBlock() {
	t.NewBlock(nil, nil)
}
func (t *testChainMgr) NewBlock(txs []*xfsgo.Transaction, receipts []*xfsgo.Receipt) {
	gasLimitFn := func() *big.Int { return new(big.Int) }
	gasUsedFn := func() *big.Int { return new(big.Int) }
	bitsFn := func() uint32 { return 0 }
	nonceFn := func() uint32 { return 0 }
	extraNonceFn := func() uint64 { return 0 }
	timenow := time.Now()
	header := &xfsgo.BlockHeader{
		Version:       0,
		Height:        t.last.Height() + 1,
		HashPrevBlock: t.last.HeaderHash(),
		Timestamp:     uint64(timenow.Unix()),
		Coinbase:      t.coinbase,
		GasLimit:      gasLimitFn(),
		GasUsed:       gasUsedFn(),
		Bits:          bitsFn(),
		Nonce:         nonceFn(),
		ExtraNonce:    extraNonceFn(),
	}
	blk := xfsgo.NewBlock(header, txs, receipts)
	_ = t.writeBlock(blk)
}
func (t *testChainMgr) CurrentBHeader() *xfsgo.BlockHeader {
	return t.last.Header
}

func (t *testChainMgr) GenesisBHeader() *xfsgo.BlockHeader {
	return t.genesis.Header
}

func (t *testChainMgr) GetBlockByNumber(num uint64) *xfsgo.Block {
	for _, v := range t.blocks {
		vh := v.GetHeader()
		if vh.Height == num {
			return v
		}
	}
	return nil
}
func (t *testChainMgr) GetBlockHashesFromHash(hash common.Hash, max uint64) (hashes []common.Hash) {
	var (
		block  *xfsgo.Block
		exists bool
	)
	if block, exists = t.blocks[hash]; !exists {
		return
	}
	for i := uint64(0); i < max; i++ {
		if block, exists = t.blocks[block.HashPrevBlock()]; !exists {
			break
		}
		hashes = append(hashes, block.HeaderHash())
	}
	return
}

func (t *testChainMgr) GetBlockByHashWithoutRec(hash common.Hash) *xfsgo.Block {
	var (
		block  *xfsgo.Block
		exists bool
	)
	if block, exists = t.blocks[hash]; !exists {
		return nil
	}
	return block
}
func (t *testChainMgr) GetReceiptByHash(hash common.Hash) *xfsgo.Receipt {
	var (
		receipt *xfsgo.Receipt
		exists  bool
	)
	if receipt, exists = t.receipts[hash]; !exists {
		return nil
	}
	return receipt
}
func (t *testChainMgr) GetBlockByHash(hash common.Hash) *xfsgo.Block {
	var (
		block  *xfsgo.Block
		exists bool
	)
	if block, exists = t.blocks[hash]; !exists {
		return nil
	}
	return block
}
func (t *testChainMgr) InsertChain(block *xfsgo.Block) error {
	header := block.Header
	_ = header
	return t.writeBlock(block)
}

func (t *testChainMgr) Slice(start, end uint64) *testChainMgr {
	mgr := &testChainMgr{
		genesis:  t.genesis,
		coinbase: t.coinbase,
		blocks:   make(map[common.Hash]*xfsgo.Block),
		txs:      make(map[common.Hash]*xfsgo.Transaction),
		receipts: make(map[common.Hash]*xfsgo.Receipt),
		last:     t.last,
	}
	if start == 0 || end <= start || end-start > uint64(len(t.blocks)) {
		return t.Copy()
	}
	mgr.blocks[mgr.genesis.HeaderHash()] = mgr.genesis
	for i := start; i < uint64(len(t.blocks)) && i < end; i++ {
		s := t.GetBlockByNumber(i)
		mgr.blocks[s.HeaderHash()] = s
		mgr.last = s
	}
	return mgr
}
func (t *testChainMgr) Copy() *testChainMgr {
	mgr := &testChainMgr{
		genesis:  t.genesis,
		coinbase: t.coinbase,
		blocks:   make(map[common.Hash]*xfsgo.Block),
		txs:      make(map[common.Hash]*xfsgo.Transaction),
		receipts: make(map[common.Hash]*xfsgo.Receipt),
		last:     t.last,
	}
	for k, v := range t.blocks {
		mgr.blocks[k] = v
	}
	for k, v := range t.txs {
		mgr.txs[k] = v
	}
	for k, v := range t.receipts {
		mgr.receipts[k] = v
	}
	return mgr
}
func (t *testChainMgr) writeBlock(block *xfsgo.Block) error {
	if _, exists := t.blocks[block.HeaderHash()]; exists {
		return nil
	}
	t.last = block
	for _, tx := range block.Transactions {
		t.txs[tx.Hash()] = tx
	}
	for _, r := range block.Receipts {
		t.receipts[r.TxHash] = r
	}
	t.blocks[block.HeaderHash()] = block
	return nil
}
func (t *testChainMgr) SetBoundaries(syncStatsOrigin, syncStatsHeight uint64) error {
	return nil
}

func newTestChainMgr(genesis *xfsgo.Block, coinbase common.Address) *testChainMgr {
	mgr := &testChainMgr{
		genesis:  genesis,
		coinbase: coinbase,
		blocks:   make(map[common.Hash]*xfsgo.Block),
		txs:      make(map[common.Hash]*xfsgo.Transaction),
		receipts: make(map[common.Hash]*xfsgo.Receipt),
	}
	mgr.blocks[genesis.HeaderHash()] = genesis
	mgr.last = genesis
	return mgr
}

type testMessageReader struct {
	version uint8
	mType   uint8
	raw     io.Reader
	data    io.Reader
}

func (m *testMessageReader) Type() uint8 {
	return m.mType
}

func (m *testMessageReader) Read(p []byte) (int, error) {
	return m.data.Read(p)
}
func (m *testMessageReader) ReadAll() ([]byte, error) {
	return io.ReadAll(m.data)
}

func (m *testMessageReader) RawReader() io.Reader {
	return m.raw
}

func (m *testMessageReader) DataReader() io.Reader {
	return m.data
}
