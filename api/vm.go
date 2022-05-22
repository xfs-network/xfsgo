package api

import (
	"xfsgo"
	"xfsgo/common"
	"xfsgo/storage/badger"
	"xfsgo/vm"
)

type VMHandler struct {
	StateDb badger.IStorage
	Chain   *xfsgo.BlockChain
}
type VMCallData struct {
	StateRoot string `json:"stateRoot"`
	From      string `json:"from"`
	To        string `json:"to"`
	Data      string `json:"data"`
}

func (v *VMHandler) Call(args VMCallData, result *string) error {
	currentHeader := v.Chain.CurrentBHeader()
	stateRoot := currentHeader.StateRoot
	stateTree := xfsgo.NewStateTree(v.StateDb, stateRoot[:])
	vmo := vm.NewXVM(stateTree)
	fromAddr := common.StrB58ToAddress(args.From)
	toAddr := common.StrB58ToAddress(args.To)
	data, err := common.HexToBytes(args.Data)
	if err != nil {
		return xfsgo.NewRPCErrorCause(-32001, err)
	}
	var buffer []byte
	if err = vmo.CallReturn(fromAddr, toAddr, data, &buffer); err != nil {
		return xfsgo.NewRPCErrorCause(-32001, err)
	}
	*result = common.BytesToHexString(buffer)
	return nil
}
