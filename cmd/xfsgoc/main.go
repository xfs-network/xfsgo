package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/big"
	"os"
	"reflect"
	"xfsgo/common"
	"xfsgo/common/ahash"
	"xfsgo/vm"
)

func writeStringParams(w vm.Buffer, s vm.CTypeString) {
	slen := len(s)
	var slenbuf [8]byte
	binary.LittleEndian.PutUint64(slenbuf[:], uint64(slen))
	_, _ = w.Write(slenbuf[:])
	_, _ = w.Write(s)
}

type StdToken struct {
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	Decimals    int    `json:"decimals"`
	TotalSupply string `json:"totalSupply"`
}

type ArgObj struct {
	Type string `json:"type"`
}

type ABIObj struct {
	Name       string    `json:"name"`
	Argc       int       `json:"argc"`
	Args       []*ArgObj `json:"args"`
	ReturnType string    `json:"returnType"`
}

func jsonDump(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}

func writeUint16(buf *bytes.Buffer, n uint16) error {
	var data [2]byte
	binary.LittleEndian.PutUint16(data[:], n)
	_, err := buf.Write(data[:])
	if err != nil {
		return err
	}
	return nil
}

func errout(err error, t string, a ...interface{}) {
	if err != nil {
		ta := fmt.Sprintf(t, a...)
		fmt.Printf("%s. err: %s\n", ta, err)
		os.Exit(1)
	}
}

var isStdtoken bool
var isBin bool
var isAbi bool
var outfile string

func init() {
	flag.BoolVar(&isStdtoken, "stdtoken", false, "")
	flag.BoolVar(&isAbi, "abi", false, "")
	flag.BoolVar(&isBin, "bin", false, "")
	flag.StringVar(&outfile, "out", "", "")
}

func packStdTokenParams(t *StdToken) ([]byte, error) {
	buffer := vm.NewBuffer(nil)
	_ = buffer.WriteString(t.Name)
	_ = buffer.WriteString(t.Symbol)
	n := t.Decimals >> 8
	if n != 0 {
		return nil, fmt.Errorf("decimals value must be uint8")
	}
	ub := [1]byte{byte(t.Decimals)}
	_, _ = buffer.Write(ub[:])
	bigTotalSupply := new(big.Int)
	bigTotalSupply, ok := bigTotalSupply.SetString(t.TotalSupply, 10)
	if !ok {
		return nil, fmt.Errorf("Failed parse totalSupply")
	}
	totalSupplyU256 := vm.NewUint256(bigTotalSupply)
	_, _ = buffer.Write(totalSupplyU256[:])
	return buffer.Bytes(), nil
}

type BuiltinCompiler struct {
	builtins map[uint8]reflect.Type
}

func NewBuiltinCompiler() *BuiltinCompiler {
	xvm := vm.NewXVM(nil)
	c := &BuiltinCompiler{
		builtins: xvm.GetBuiltins(),
	}
	return c
}

func parseMethodArgs(m reflect.Method) (int, []*ArgObj) {
	mt := m.Type
	argc := mt.NumIn()
	argc = argc - 1
	argobjs := make([]*ArgObj, 0)
	margc := 0
	for i := 1; i < argc+1; i++ {
		argItem := mt.In(i)
		argTypeName := argItem.Name()
		switch argItem {
		case reflect.TypeOf(&vm.ContractContext{}):
			continue
		}
		argObj := &ArgObj{
			Type: argTypeName,
		}
		argobjs = append(argobjs, argObj)
		margc += 1
	}
	return margc, argobjs
}

func (c *BuiltinCompiler) exportABI(id uint8) (map[string]*ABIObj, error) {
	if ct, exists := c.builtins[id]; exists {
		abiobjs := make(map[string]*ABIObj)
		for i := 0; i < ct.NumMethod(); i++ {
			methodItem := ct.Method(i)
			methodName := methodItem.Name
			namehash := ahash.SHA256([]byte(methodName))
			namehashstr := common.BytesToHexString(namehash)
			if methodItem.Type.Kind() == reflect.Func &&
				methodName != "BuiltinId" {
				argc, argobjs := parseMethodArgs(methodItem)
				out0 := methodItem.Type.Out(0)
				out0name := out0.Name()
				// mv := cv.MethodByName(aname)
				item := &ABIObj{
					Name:       methodName,
					Argc:       argc,
					Args:       argobjs,
					ReturnType: out0name,
				}
				if methodName == "Create" {
					zorestr := common.BytesToHexString(common.HashZ[:])
					abiobjs[zorestr] = item
					continue
				}
				abiobjs[namehashstr] = item
			}
		}
		return abiobjs, nil
	}
	return nil, errors.New("Not found builtin contract id")
}

func outbin(writer *bytes.Buffer, w io.Writer) {
	data := writer.Bytes()
	out := hex.EncodeToString(data)
	_, err := fmt.Fprintf(w, "0x%s\n", out)
	errout(err, "Failed write: ")
	os.Exit(0)
}

func outabi(abi map[string]*ABIObj, w io.Writer) {
	abijson, err := json.Marshal(abi)
	errout(err, "Failed export abi data")
	_, err = fmt.Fprintln(w, string(abijson))
	errout(err, "Failed write: ")
	os.Exit(0)
}

func usage() {
	name := os.Args[0]
	fmt.Printf(`Usage: %s [options]
Options:
  -stdtoken          Built in contract like ERC20
  -abi               Print abi format structure
  -bin               Print contract bin code
  -out <filename>    Set output filepath
  -h, -help          Display this informatio
  -version           Print version info
`, name)
}

func main() {
	flag.Usage = usage
	flag.Parse()
	var err error
	binwriter := bytes.NewBuffer(nil)
	err = writeUint16(&*binwriter, vm.MagicNumberXVM)
	errout(err, "Unknown wrong")
	compiler := NewBuiltinCompiler()
	// args := flag.Args()
	// if isStdtoken && (len(args) > 0 || args[0] != "") {
	out := os.Stdout
	if outfile != "" {
		file, err := os.OpenFile(outfile, os.O_WRONLY|os.O_CREATE, 0644)
		errout(err, "Failed write file")
		out = file
	}
	if isStdtoken && isBin {
		// fileData, err := ioutil.ReadFile(args[0])
		// errout(err, "Unable to read file: %s", args[0])
		// var inputToken StdToken
		// err = json.Unmarshal(fileData, &inputToken)
		// errout(err, "Unable to parse json sechme: %s", args[0])
		binwriter.Write([]byte{0x01})
		// _, err = packStdTokenParams(&inputToken)
		// errout(err, "Failed pack params")
		// writer.Write(data)
		outbin(binwriter, out)
	} else if isStdtoken && isAbi {
		abi, err := compiler.exportABI(0x01)
		errout(err, "Failed export abi data")
		outabi(abi, out)
	}
	flag.Usage()
}
