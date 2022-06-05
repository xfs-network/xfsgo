package vm

import (
	"bytes"
	"encoding/binary"
	"errors"
	"reflect"
	"xfsgo/common"
	"xfsgo/common/ahash"
	"xfsgo/core"
	"xfsgo/crypto"
)

type VM interface {
	Run(common.Address, common.Address, []byte, []byte) error
	Create(common.Address, []byte) error
	Call(common.Address, common.Address, []byte) error
	CallReturn(common.Address, common.Address, []byte, []byte) error
}

const (
	MagicNumberXVM = uint16(9168)
)

var (
	errUnknownMagicNumber  = errors.New("unknown magic number")
	errUnknownContractId   = errors.New("unknown contract type")
	errUnknownContractExec = errors.New("unknown contract exec")
	errInvalidContractCode = errors.New("invalid contract code")
)

type xvm struct {
	stateTree core.StateTree
	builtins  map[uint8]reflect.Type
	logger    Logger
}

func NewXVM(st core.StateTree) *xvm {
	vm := &xvm{
		stateTree: st,
		builtins:  make(map[uint8]reflect.Type),
		logger:    NewLogger(),
	}
	vm.registerBuiltinId(new(token))
	vm.registerBuiltinId(new(nftoken))
	return vm
}
func (vm *xvm) newBuiltinContractExec(
	id uint8, from, address common.Address, code []byte) (*builtinContractExec, error) {
	if ct, exists := vm.builtins[id]; exists {
		return &builtinContractExec{
			contractT: ct,
			caller:    from,
			stateTree: vm.stateTree,
			address:   address,
			code:      code,
			logger:    vm.logger,
			resultBuf: bytes.NewBuffer(nil),
		}, nil
	}
	return nil, errUnknownContractId
}
func (vm *xvm) registerBuiltinId(b BuiltinContract) {
	bid := b.BuiltinId()
	if _, exists := vm.builtins[bid]; !exists {
		rt := reflect.TypeOf(b)
		vm.builtins[bid] = rt
	}
}

func (vm *xvm) GetBuiltins() map[uint8]reflect.Type {
	return vm.builtins
}

func readXVMCode(code []byte, input []byte) (c []byte, id uint8, err error) {
	if code == nil && input != nil {
		code = make([]byte, 3)
		copy(code[:], input[:])
	}
	if code == nil || len(code) < 3 {
		return code, 0, errInvalidContractCode
	}
	m := binary.LittleEndian.Uint16(code[:2])
	if m != MagicNumberXVM {
		return code, 0, errUnknownMagicNumber
	}
	c = code
	id = code[2]
	return
}
func (vm *xvm) Run(fromAddr, addr common.Address, code []byte, input []byte) (err error) {
	var create = code == nil
	code, id, err := readXVMCode(code, input)
	if err != nil && create {
		vm.stateTree.AddNonce(addr, 1)
		vm.stateTree.SetCode(addr, input)
		return nil
	} else if err != nil {
		return nil
	}
	var exec ContractExec
	if id != 0 {
		if exec, err = vm.newBuiltinContractExec(
			id, fromAddr, addr, code); err != nil {
			return
		}
	}
	if exec == nil {
		return errUnknownContractExec
	}
	if create {
		var realInput = make([]byte, len(input)-3)
		copy(realInput[:], input[3:])
		if err = exec.Create(realInput); err != nil {
			return err
		}
		vm.stateTree.AddNonce(addr, 1)
		vm.stateTree.SetCode(addr, code)
		return nil
	}
	if err = exec.Call(input); err != nil {
		return err
	}
	return nil
}
func (vm *xvm) Create(addr common.Address, input []byte) error {
	nonce := vm.stateTree.GetNonce(addr)
	fromAddressHashBytes := ahash.SHA256(addr[:])
	fromAddressHash := common.Bytes2Hash(fromAddressHashBytes)
	caddr := crypto.CreateAddress(fromAddressHash, nonce)
	if err := vm.Run(addr, caddr, nil, input); err != nil {
		return err
	}
	return nil
}

func (vm *xvm) Call(from, address common.Address, input []byte) error {
	code := vm.stateTree.GetCode(address)
	if code == nil {
		return nil
	}
	if err := vm.Run(from, address, code, input); err != nil {
		return err
	}

	return nil
}
func (vm *xvm) CallReturn(from, to common.Address, input []byte, result *[]byte) error {
	code := vm.stateTree.GetCode(to)
	data, id, err := readXVMCode(code, input)
	if err != nil {
		return err
	}
	exec, err := vm.newBuiltinContractExec(id, from, to, data)
	if err != nil {
		return err
	}
	if len(input)-3 < 0 {
		return errors.New("non standard code")
	}
	return exec.CallReturn(input[3:], result)
}

func (vm *xvm) GetLogger() Logger {
	return vm.logger
}
