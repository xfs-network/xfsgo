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

func (v *VMHandler) Call(args VMCallData, result **string) error {
	currentHeader := v.Chain.CurrentBHeader()
	stateRoot := currentHeader.StateRoot
	var err error
	if args.StateRoot != "" {
		stateRoot = common.Hex2Hash(args.StateRoot)
	}
	stateTree, err := xfsgo.NewStateTreeN(v.StateDb, stateRoot[:])
	if err != nil {
		return xfsgo.LoadStateTreeError("Load status tree error: %s, from: %x", err, stateTree)
	}
	vmo := vm.NewXVM(stateTree)
	var fromAddress common.Address
	if args.From != "" {
		fromAddress = common.StrB58ToAddress(args.From)
	}
	var toAddress common.Address
	if args.To == "" {
		return xfsgo.RequireParamError("Require param 'to'")
	}
	toAddress = common.StrB58ToAddress(args.To)
	var data []byte
	if args.Data == "" {
		return xfsgo.RequireParamError("Require param 'data'")
	}
	data, err = common.HexToBytes(args.Data)
	if err != nil {
		return xfsgo.ParamsParseError("Parse param 'data' error: %s", err)
	}
	var buffer []byte
	if err = vmo.CallReturn(fromAddress, toAddress, data, &buffer); err != nil {
		return xfsgo.NewRPCErrorCause(-32001, err)
	}
	resultstring := common.BytesToHexString(buffer)
	if resultstring == "" {
		*result = nil
		return nil
	}
	*result = &resultstring
	return nil
}
