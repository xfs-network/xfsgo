package backend

import (
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
	"xfsgo"
	"xfsgo/common"
	"xfsgo/common/priqueue"
	"xfsgo/p2p/discover"
)

var (
	queueBlockSize          = maxHashesFetch * 8
	blockCacheLimit         = maxBlocksFetch * 8
	errNotFoundFetchPending = errors.New("not found fetch pending")
	errInvalidChain         = errors.New("retrieved hash chain is invalid")
)

type queueBlock struct {
	rawBlock   *xfsgo.Block
	originPeer discover.NodeId
}
type syncQueue struct {
	lock        sync.RWMutex
	hashCounter int
	hashPool    map[common.Hash]int
	blockPool   map[common.Hash]int
	pendPool    map[discover.NodeId]*fetchBlockRequest
	hashQueue   *priqueue.Priqueue
	blockCache  []*queueBlock
	blockOffset uint64
}

type fetchBlockRequest struct {
	peer   syncpeer
	hashes map[common.Hash]int
	time   time.Time
}

func newSyncQueue() *syncQueue {
	return &syncQueue{
		hashCounter: 0,
		hashPool:    make(map[common.Hash]int),
		blockPool:   make(map[common.Hash]int),
		pendPool:    make(map[discover.NodeId]*fetchBlockRequest),
		hashQueue:   priqueue.New(int(queueBlockSize)),
		blockCache:  make([]*queueBlock, blockCacheLimit),
		blockOffset: 0,
	}
}

func (queue *syncQueue) Reset() {
	queue.lock.Lock()
	defer queue.lock.Unlock()
	queue.hashCounter = 0
	queue.hashPool = make(map[common.Hash]int)
	queue.blockPool = make(map[common.Hash]int)
	queue.pendPool = make(map[discover.NodeId]*fetchBlockRequest)
	queue.hashQueue.Reset()
	queue.blockCache = make([]*queueBlock, blockCacheLimit)
	queue.blockOffset = 0
}
func (queue *syncQueue) Insert(hashes []common.Hash) []common.Hash {
	queue.lock.Lock()
	defer queue.lock.Unlock()
	inserts := make([]common.Hash, 0, len(hashes))
	for _, hash := range hashes {
		// Skip anything we already have
		if old, ok := queue.hashPool[hash]; ok {
			logrus.Debugf("Hash %x already scheduled at index %v", hash, old)
			continue
		}
		// Update the counters and insert the hash
		queue.hashCounter = queue.hashCounter + 1
		inserts = append(inserts, hash)
		queue.hashPool[hash] = queue.hashCounter
		queue.hashQueue.Push(hash, -queue.hashCounter)
	}
	return inserts
}
func (queue *syncQueue) Size() (int, int) {
	queue.lock.RLock()
	defer queue.lock.RUnlock()
	return len(queue.hashPool), len(queue.blockPool)
}
func (queue *syncQueue) Pending() int {
	queue.lock.RLock()
	defer queue.lock.RUnlock()
	return queue.hashQueue.Size()
}

func (queue *syncQueue) FetchPending() int {
	queue.lock.RLock()
	defer queue.lock.RUnlock()
	return len(queue.pendPool)
}
func (queue *syncQueue) Cancel(request *fetchBlockRequest) {
	queue.lock.Lock()
	defer queue.lock.Unlock()

	for hash, index := range request.hashes {
		queue.hashQueue.Push(hash, index)
	}
	delete(queue.pendPool, request.peer.ID())
}
func (queue *syncQueue) Throttle() bool {
	queue.lock.RLock()
	defer queue.lock.RUnlock()

	// Calculate the currently in-flight block requests
	pending := 0
	for _, request := range queue.pendPool {
		pending += len(request.hashes)
	}
	// Throttle if more blocks are in-flight than free space in the cache
	return pending >= len(queue.blockCache)-len(queue.blockPool)
}
func (queue *syncQueue) TakeBlocks() []*queueBlock {
	queue.lock.RLock()
	defer queue.lock.RUnlock()
	blocks := make([]*queueBlock, 0)
	for _, block := range queue.blockCache {
		if block == nil {
			break
		}
		blocks = append(blocks, block)
		delete(queue.blockPool, block.rawBlock.HeaderHash())
	}
	copy(queue.blockCache, queue.blockCache[len(blocks):])
	for k, n := len(queue.blockCache)-len(blocks), len(queue.blockCache); k < n; k++ {
		queue.blockCache[k] = nil
	}
	queue.blockOffset += uint64(len(blocks))
	return blocks
}

func (queue *syncQueue) Reserve(p syncpeer, count int) *fetchBlockRequest {
	queue.lock.Lock()
	defer queue.lock.Unlock()
	if queue.hashQueue.Empty() {
		return nil
	}
	if _, exists := queue.pendPool[p.ID()]; exists {
		return nil
	}
	space := len(queue.blockCache) - len(queue.blockPool)
	for _, request := range queue.pendPool {
		space -= len(request.hashes)
	}
	send := make(map[common.Hash]int)
	skip := make(map[common.Hash]int)
	for i := 0; i < space && len(send) < count && !queue.hashQueue.Empty(); i++ {
		hash, priority := queue.hashQueue.Pop()
		if p.HasIgnoreHash(hash.(common.Hash)) {
			skip[hash.(common.Hash)] = priority
		} else {
			send[hash.(common.Hash)] = priority
		}
	}
	for hash, index := range skip {
		queue.hashQueue.Push(hash, index)
	}
	if len(send) == 0 {
		return nil
	}
	request := &fetchBlockRequest{
		peer:   p,
		hashes: send,
		time:   time.Now(),
	}
	queue.pendPool[p.ID()] = request
	return request
}
func (queue *syncQueue) Expire(timeout time.Duration) []discover.NodeId {
	queue.lock.Lock()
	defer queue.lock.Unlock()

	// Iterate over the expired requests and return each to the queue
	peers := make([]discover.NodeId, 0)
	for id, request := range queue.pendPool {
		if time.Since(request.time) > timeout {
			for hash, index := range request.hashes {
				queue.hashQueue.Push(hash, index)
			}
			peers = append(peers, id)
		}
	}
	// Remove the expired requests from the pending pool
	for _, id := range peers {
		delete(queue.pendPool, id)
	}
	return peers
}
func (queue *syncQueue) Deliver(p syncpeer, blocks []*xfsgo.Block) error {
	queue.lock.Lock()
	defer queue.lock.Unlock()
	pid := p.ID()
	request := queue.pendPool[pid]
	if request == nil {
		return errNotFoundFetchPending
	}
	delete(queue.pendPool, pid)
	if len(blocks) == 0 {
		for hash, _ := range request.hashes {
			request.peer.AddIgnoreHash(hash)
		}
	}
	errs := make([]error, 0)
	for _, block := range blocks {
		// Skip any blocks that were not requested
		hash := block.HeaderHash()
		if _, ok := request.hashes[hash]; !ok {
			errs = append(errs, fmt.Errorf("non-requested block %x", hash))
			continue
		}

		index := int(int64(block.Height()) - int64(queue.blockOffset))
		if index >= len(queue.blockCache) || index < 0 {
			return errInvalidChain
		}
		queue.blockCache[index] = &queueBlock{
			rawBlock:   block,
			originPeer: p.ID(),
		}
		delete(request.hashes, hash)
		delete(queue.hashPool, hash)
		queue.blockPool[hash] = int(block.Height())
	}
	for hash, index := range request.hashes {
		queue.hashQueue.Push(hash, index)
	}
	if len(errs) != 0 {
		return fmt.Errorf("deliver errs %v", errs)
	}
	return nil
}

func (queue *syncQueue) Prepare(offset uint64) {
	queue.lock.Lock()
	defer queue.lock.Unlock()
	if queue.blockOffset < offset {
		queue.blockOffset = offset
	}
}
