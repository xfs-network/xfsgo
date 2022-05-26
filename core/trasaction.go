package core

import (
	"math/big"
	"xfsgo/common"
)

type ITransaction interface {
	Hash() common.Hash
	Cost() *big.Int
	FromAddress() common.Address
	ToAddress() common.Address
}
