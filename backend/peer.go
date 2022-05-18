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

package backend

import (
	"encoding/json"
	"errors"
	"github.com/sirupsen/logrus"
	"math/big"
	"sync"
	"time"
	"xfsgo"
	"xfsgo/common"
	"xfsgo/p2p"
	"xfsgo/p2p/discover"
)

type syncpeer interface {
	Head() common.Hash
	ID() discover.NodeId
	Height() uint64
	SetHead(hash common.Hash)
	SetHeight(height uint64)
	P2PPeer() p2p.Peer
	Close()
	SendData(mType uint8, data []byte) error
	SendObject(mType uint8, data interface{}) error
	Handshake(head common.Hash, height uint64, genesis common.Hash) error
	RequestHashesFromNumber(from uint64, count uint64) error
	SendBlockHashes(hashes RemoteHashes) error
	RequestBlocks(hashes RemoteHashes) error
	SendBlocks(blocks RemoteBlocks) error
	SendNewBlock(data *RemoteBlock) error
	SendTransactions(data RemoteTxs) error
	SendTxhash(data TxHashs) error
	SendReceiptsData(data ReceiptsSet) error
	GetProtocolMsgCh() (chan p2p.MessageReader, error)
	AddIgnoreHash(hash common.Hash)
	HasIgnoreHash(hash common.Hash) bool
	HasBlock(common.Hash) bool
	AddBlock(common.Hash)
	AddTx(common.Hash)
	HasTx(common.Hash) bool
	Reset()
}

type peer struct {
	lock            sync.RWMutex
	p2pPeer         p2p.Peer
	version         uint32
	network         uint32
	head            common.Hash
	height          uint64
	ignoreHashes    map[common.Hash]struct{}
	knownBlocksLock sync.RWMutex
	knownBlocks     map[common.Hash]struct{}
	knownTxsLock    sync.RWMutex
	knownTxs        map[common.Hash]struct{}
}

const (
	MsgCodeVersion              uint8 = 5
	GetBlockHashesFromNumberMsg uint8 = 6
	BlockHashesMsg              uint8 = 7
	GetBlocksMsg                uint8 = 8
	BlocksMsg                   uint8 = 9
	NewBlockMsg                 uint8 = 10
	TxMsg                       uint8 = 11
	GetReceipts                 uint8 = 12
	ReceiptsData                uint8 = 13
)

var (
	errHandshakeFailed = errors.New("protocol handshake failed")
)

func newPeer(p p2p.Peer, version uint32, network uint32) *peer {
	pt := &peer{
		p2pPeer:      p,
		version:      version,
		network:      network,
		ignoreHashes: make(map[common.Hash]struct{}),
		knownBlocks:  make(map[common.Hash]struct{}),
		knownTxs:     make(map[common.Hash]struct{}),
	}
	return pt
}
func (p *peer) HasBlock(hash common.Hash) (r bool) {
	p.knownBlocksLock.RLock()
	defer p.knownBlocksLock.RUnlock()
	_, r = p.knownBlocks[hash]
	return
}
func (p *peer) AddBlock(hash common.Hash) {
	p.addKnownBlock(hash)
}

func (p *peer) HasTx(hash common.Hash) (r bool) {
	p.knownTxsLock.RLock()
	defer p.knownTxsLock.RUnlock()
	_, r = p.knownTxs[hash]
	return
}
func (p *peer) AddTx(hash common.Hash) {
	p.addKnownTx(hash)
}
func (p *peer) GetProtocolMsgCh() (chan p2p.MessageReader, error) {
	return p.p2pPeer.GetProtocolMsgCh()
}
func (p *peer) ID() discover.NodeId {
	return p.p2pPeer.ID()
}
func (p *peer) Head() (out common.Hash) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	copy(out[:], p.head[:])
	return out
}
func (p *peer) Close() {
	p.p2pPeer.Close()
}
func (p *peer) Height() uint64 {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.height
}
func (p *peer) SetHead(hash common.Hash) {
	p.lock.Lock()
	defer p.lock.Unlock()
	copy(p.head[:], hash[:])
}

func (p *peer) SetHeight(height uint64) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.height = height
}

func (p *peer) P2PPeer() p2p.Peer {
	return p.p2pPeer
}
func (p *peer) AddIgnoreHash(hash common.Hash) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if _, exists := p.ignoreHashes[hash]; !exists {
		p.ignoreHashes[hash] = struct{}{}
	}
}
func (p *peer) HasIgnoreHash(hash common.Hash) (exists bool) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	_, exists = p.ignoreHashes[hash]
	return
}
func (p *peer) Reset() {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.ignoreHashes = make(map[common.Hash]struct{})
}

type statusData struct {
	Version uint32      `json:"version"`
	Network uint32      `json:"network"`
	Head    common.Hash `json:"head"`
	Height  uint64      `json:"height"`
	Genesis common.Hash `json:"genesis"`
}

type getBlockHashesFromNumberData struct {
	From  uint64 `json:"from"`
	Count uint64 `json:"count"`
}

type AllSyncData struct {
	ID     string      `json:"id"`
	Head   common.Hash `json:"head"`
	Height uint64      `json:"height"`
}

type RemoteBlockHeader struct {
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
	// pow consensus.
	Bits       uint32      `json:"bits"`
	Nonce      uint32      `json:"nonce"`
	ExtraNonce uint64      `json:"extranonce"`
	Hash       common.Hash `json:"hash"`
}
type RemoteBlockTx struct {
	Version   uint32         `json:"version"`
	From      common.Address `json:"from"`
	To        common.Address `json:"to"`
	GasPrice  *big.Int       `json:"gas_price"`
	GasLimit  *big.Int       `json:"gas_limit"`
	Data      []byte         `json:"data"`
	Nonce     uint64         `json:"nonce"`
	Value     *big.Int       `json:"value"`
	Signature []byte         `json:"signature"`
	Hash      common.Hash    `json:"hash"`
}

type RemoteTxs []*RemoteBlockTx

type RemoteBlock struct {
	Header       *RemoteBlockHeader `json:"header"`
	Transactions RemoteTxs          `json:"transactions"`
	Receipts     []*xfsgo.Receipt   `json:"receipts"`
}

type ReceiptsSet []*xfsgo.Receipt
type RemoteHashes []common.Hash
type RemoteBlocks []*RemoteBlock

type TxHashs []common.Hash

func coverTx2RemoteTx(tx *xfsgo.Transaction) (re *RemoteBlockTx) {
	if tx == nil {
		return nil
	}
	_ = common.Objcopy(tx, &re)
	re.Hash = tx.Hash()
	re.From, _ = tx.FromAddr()
	return
}

func coverTxs2RemoteBlockTxs(txs []*xfsgo.Transaction) (re RemoteTxs) {
	if txs == nil || len(txs) < 0 {
		return nil
	}
	re = make(RemoteTxs, 0)
	for _, tx := range txs {
		re = append(re, coverTx2RemoteTx(tx))
	}
	return
}

func coverTxs2RemoteBlockTxHashs(txs []*xfsgo.Transaction) (re TxHashs) {
	if txs == nil || len(txs) < 0 {
		return nil
	}
	re = make(TxHashs, 0)
	for _, tx := range txs {
		txObj := coverTx2RemoteTx(tx)
		re = append(re, txObj.Hash)
	}
	return
}

func coverBlockHeader2RemoteBlockHeader(header *xfsgo.BlockHeader) (re *RemoteBlockHeader) {
	if header == nil {
		return nil
	}
	_ = common.Objcopy(header, &re)
	re.Hash = header.HeaderHash()
	return
}
func coverBlock2RemoteBlock(block *xfsgo.Block) (re *RemoteBlock) {
	if block == nil {
		return nil
	}
	re = new(RemoteBlock)
	re.Header = coverBlockHeader2RemoteBlockHeader(block.Header)
	re.Transactions = coverTxs2RemoteBlockTxs(block.Transactions)
	re.Receipts = block.Receipts
	return
}

// Handshake runs the protocol handshake using messages(hash value and height of current block).
// to verifies whether the peer matchs the prptocol that attempts to add the connection as a peer.
func (p *peer) Handshake(head common.Hash, height uint64, genesis common.Hash) error {

	go func() {
		if err := p2p.SendMsgData(p.p2pPeer, MsgCodeVersion, &statusData{
			Version: p.version,
			Network: p.network,
			Head:    head,
			Height:  height,
			Genesis: genesis,
		}); err != nil {
			return
		}
	}()
	// r := p.p2pPeer.Reader()
	msgCh, err := p.p2pPeer.GetProtocolMsgCh()
	if err != nil {
		return err
	}
	for {
		select {
		case msg := <-msgCh:
			msgCode := msg.Type()
			switch msgCode {
			case MsgCodeVersion:
				data, _ := msg.ReadAll()
				status := statusData{}
				if err := json.Unmarshal(data, &status); err != nil {
					return err
				}
				if status.Genesis != genesis {
					logrus.Debugf("Sync peer handshake failed: wantGenesisHash=%x, gotGenesisHash=%x, from=%s",
						genesis, status.Genesis, p.P2PPeer().RemoteNode().ID)
					return errHandshakeFailed
				}
				if status.Version != p.version {
					logrus.Debugf("Sync peer handshake failed: wantVersion=%d, gotVersion=%d, from=%s",
						p.version, status.Version, p.P2PPeer().RemoteNode().ID)
					return errHandshakeFailed
				}
				if status.Network != p.network {
					logrus.Debugf("Sync peer handshake failed: wantHetwork=%d, gotNetwork=%d, from=%s",
						p.network, status.Network, p.P2PPeer().RemoteNode().ID)
					return errHandshakeFailed
				}
				p.head = status.Head
				p.height = status.Height
				pid := p.P2PPeer().RemoteNode().ID
				logrus.Debugf("Successfully handshake by sync transport: height=%d, head=%x, id=%x", status.Height, p.head[len(p.head)-4:], pid[len(pid)-4:])
				return nil
			}
		case <-time.After(3 * time.Second):
			return errors.New("time out")
		}
	}
}

// RequestHashesFromNumber fetches a batch of hashes from a peer, starting at from, getting count
func (p *peer) RequestHashesFromNumber(from uint64, count uint64) error {
	if err := p2p.SendMsgData(p.p2pPeer, GetBlockHashesFromNumberMsg, &getBlockHashesFromNumberData{
		From:  from,
		Count: count,
	}); err != nil {
		return err
	}
	return nil
}

// SendBlockHashes sends a batch of hashes from a peer
func (p *peer) SendBlockHashes(hashes RemoteHashes) error {
	if err := p2p.SendMsgData(p.p2pPeer, BlockHashesMsg, &hashes); err != nil {
		return err
	}
	return nil
}

// RequestBlocks fetches a batch of blocks based on the hash values
func (p *peer) RequestBlocks(hashes RemoteHashes) error {
	if err := p2p.SendMsgData(p.p2pPeer, GetBlocksMsg, &hashes); err != nil {
		return err
	}
	return nil
}

// SendBlocks sends a batch of blocks
func (p *peer) SendBlocks(blocks RemoteBlocks) error {
	if err := p2p.SendMsgData(p.p2pPeer, BlocksMsg, &blocks); err != nil {
		return err
	}
	return nil
}

func (p *peer) addKnownBlock(hash common.Hash) {
	p.knownBlocksLock.Lock()
	defer p.knownBlocksLock.Unlock()
	if _, exists := p.knownBlocks[hash]; !exists {
		p.knownBlocks[hash] = struct{}{}
	}
}
func (p *peer) addKnownTx(hash common.Hash) {
	p.knownTxsLock.Lock()
	defer p.knownTxsLock.Unlock()
	if _, exists := p.knownTxs[hash]; !exists {
		p.knownTxs[hash] = struct{}{}
	}
}

// SendNewBlock sends a new block
func (p *peer) SendNewBlock(data *RemoteBlock) error {
	p.addKnownBlock(data.Header.Hash)
	if err := p2p.SendMsgData(p.p2pPeer, NewBlockMsg, data); err != nil {
		return err
	}
	return nil
}

// SendTransactions sends a batch of transactions
func (p *peer) SendTransactions(data RemoteTxs) error {
	for _, tx := range data {
		p.addKnownTx(tx.Hash)
	}
	if err := p2p.SendMsgData(p.p2pPeer, TxMsg, &data); err != nil {
		return err
	}
	return nil
}
func (p *peer) SendTxhash(data TxHashs) error {
	if err := p2p.SendMsgData(p.p2pPeer, GetReceipts, &data); err != nil {
		return err
	}
	return nil

}
func (p *peer) SendReceiptsData(data ReceiptsSet) error {
	if err := p2p.SendMsgData(p.p2pPeer, ReceiptsData, &data); err != nil {
		return err
	}
	return nil
}

func (p *peer) SendData(mType uint8, data []byte) error {
	return p.p2pPeer.WriteMessage(mType, data)
}
func (p *peer) SendObject(mType uint8, data interface{}) error {
	return p.p2pPeer.WriteMessageObj(mType, data)
}

type peerSet struct {
	mu    sync.RWMutex
	peers map[discover.NodeId]syncpeer
}

func newPeerSet() *peerSet {
	return &peerSet{
		peers: make(map[discover.NodeId]syncpeer),
	}
}

func (ps *peerSet) appendPeer(p syncpeer) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if _, exists := ps.peers[p.ID()]; exists {
	}
	ps.peers[p.ID()] = p
}

func (ps *peerSet) peerMap() map[discover.NodeId]syncpeer {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.peers
}

func (ps *peerSet) peerList() []syncpeer {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	all := make([]syncpeer, 0, len(ps.peers))
	for _, v := range ps.peers {
		all = append(all, v)
	}
	return all
}

func (ps *peerSet) rmPeer(id discover.NodeId) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if _, exists := ps.peers[id]; !exists {
		return
	}
	delete(ps.peers, id)
}

func (ps *peerSet) dropPeer(pid discover.NodeId) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if p, exists := ps.peers[pid]; exists {
		delete(ps.peers, pid)
		p.Close()
	}
}
func (ps *peerSet) setHeight(id discover.NodeId, height uint64) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if _, exists := ps.peers[id]; !exists {
		return
	}
	ps.peers[id].SetHeight(height)
}

func (ps *peerSet) setHead(id discover.NodeId, head common.Hash) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if _, exists := ps.peers[id]; !exists {
		return
	}
	ps.peers[id].SetHead(head)
}

func (ps *peerSet) get(id discover.NodeId) syncpeer {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	if p, exists := ps.peers[id]; exists {
		return p
	}
	return nil
}

func (ps *peerSet) count() int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return len(ps.peers)
}

func (ps *peerSet) basePeer() syncpeer {
	var (
		base       syncpeer = nil
		baseHeight          = uint64(0)
	)
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	for _, v := range ps.peers {
		if ph := v.Height(); ph > baseHeight {
			base = v
			baseHeight = ph
		}
	}
	return base
}
func (ps *peerSet) reset() {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	for _, v := range ps.peers {
		v.Reset()
	}
}

func (ps *peerSet) listAndShort() []syncpeer {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	all := make([]syncpeer, 0, len(ps.peers))
	for _, v := range ps.peers {
		all = append(all, v)
	}
	return all
}

func (ps *peerSet) empty() bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return len(ps.peers) == 0
}
