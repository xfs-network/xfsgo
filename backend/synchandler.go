package backend

import (
	"xfsgo"
	"xfsgo/common"
	"xfsgo/p2p/discover"
)

type handlerHashesFn func(id discover.NodeId, hashes RemoteHashes)
type handlerBlocksFn func(id discover.NodeId, hashes RemoteBlocks)
type handlerNewBlockFn func(id discover.NodeId, block *RemoteBlock) error
type handlerTransactionsFn func(id discover.NodeId, txs RemoteTxs) error

type syncHandler struct {
	chain                chainMgr
	handlerHashesFn      handlerHashesFn
	handlerBlocksFn      handlerBlocksFn
	handlerNewBlockFn    handlerNewBlockFn
	handlerTransactionFn handlerTransactionsFn
}

func newSyncHandler(chain chainMgr, hashesFn handlerHashesFn,
	blocksFn handlerBlocksFn, newBlockFn handlerNewBlockFn,
	transactionsFn handlerTransactionsFn) *syncHandler {
	return &syncHandler{
		chain:                chain,
		handlerHashesFn:      hashesFn,
		handlerBlocksFn:      blocksFn,
		handlerNewBlockFn:    newBlockFn,
		handlerTransactionFn: transactionsFn,
	}
}

func (handler *syncHandler) handleGetBlockHashes(req *request, p sender) error {
	var args *getBlockHashesFromNumberData
	if err := req.jsonObj(&args); err != nil {
		return err
	}
	last := handler.chain.GetBlockByNumber(args.From + args.Count - 1)
	if last == nil {
		bHeader := handler.chain.CurrentBHeader()
		tempBlock := &xfsgo.Block{Header: bHeader, Transactions: nil, Receipts: nil}
		last = tempBlock
		args.Count = last.Height() - args.From + 1
	}
	if last.Height() < args.From {
		return p.SendObject(BlockHashesMsg, nil)
	}
	hashes := []common.Hash{last.HeaderHash()}
	hashes = append(hashes, handler.chain.GetBlockHashesFromHash(last.HeaderHash(), args.Count-1)...)
	for i := 0; i < len(hashes)/2; i++ {
		hashes[i], hashes[len(hashes)-1-i] = hashes[len(hashes)-1-i], hashes[i]
	}
	return p.SendObject(BlockHashesMsg, &hashes)
}

func (handler *syncHandler) handleGotBlockHashes(req *request, _ sender) error {
	var args RemoteHashes
	if err := req.jsonObj(&args); err != nil {
		return err
	}
	handler.handlerHashesFn(req.peerId, args)
	return nil
}

func (handler *syncHandler) handleGetBlocks(req *request, p sender) error {
	var args RemoteHashes
	if err := req.jsonObj(&args); err != nil {
		return err
	}
	blocks := make(RemoteBlocks, 0)
	for _, hash := range args {
		block := handler.chain.GetBlockByHashWithoutRec(hash)
		if block == nil {
			break
		}
		newBlock := coverBlock2RemoteBlock(block)
		blocks = append(blocks, newBlock)
	}
	return p.SendObject(BlocksMsg, &blocks)
}

func (handler *syncHandler) handleGotBlocks(req *request, _ sender) error {
	var args RemoteBlocks
	if err := req.jsonObj(&args); err != nil {
		return err
	}
	//time.Sleep(1 * time.Second)
	handler.handlerBlocksFn(req.peerId, args)
	return nil
}

func (handler *syncHandler) handleNewBlock(req *request, _ sender) error {
	var args *RemoteBlock
	if err := req.jsonObj(&args); err != nil {
		return err
	}
	return handler.handlerNewBlockFn(req.peerId, args)
}

func (handler *syncHandler) handleTransactions(req *request, _ sender) error {
	var args RemoteTxs
	if err := req.jsonObj(&args); err != nil {
		return err
	}
	return handler.handlerTransactionFn(req.peerId, args)
}

func (handler *syncHandler) handleGetReceipts(req *request, p sender) error {
	var args RemoteHashes
	if err := req.jsonObj(&args); err != nil {
		return err
	}
	data := make(ReceiptsSet, 0)
	for _, item := range args {
		if val := handler.chain.GetReceiptByHash(item); val != nil {
			data = append(data, val)
		}
	}
	return p.SendObject(ReceiptsData, &data)
}

func (handler *syncHandler) handleGotReceipts(req *request, _ sender) error {
	var args ReceiptsSet
	if err := req.jsonObj(&args); err != nil {
		return err
	}
	return nil
}
