package vm

import "xfsgo/common"

type BuiltinContract interface {
	BuiltinId() (id uint8)
}

type ContractContext struct {
	caller common.Address
	logger Logger
}
