package core

import (
	"xfsgo/common"
)

type IBlock interface {
	HashPrevBlock() common.Hash
	HeaderHash() common.Hash
	Height() uint64
	StateRoot() common.Hash
	Coinbase() common.Address
	TransactionRoot() common.Hash
	ReceiptsRoot() common.Hash
	Bits() uint32
	Nonce() uint32
	ExtraNonce() uint64
}
