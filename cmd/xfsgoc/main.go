package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"xfsgo/common"
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
    Name string `json:name`
    Symbol string `json:symbol`
    Decimals int `json:decimals`
    TotalSupply string `json:totalSupply`
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

func errout(err error, t string, a ...interface{}){
    if err != nil {
        ta := fmt.Sprintf(t, a...)
        fmt.Printf("%s. err: %s\n", ta, err)
        os.Exit(1)
    }
}
var isStdtoken bool

func init(){
    flag.BoolVar(&isStdtoken, "stdtoken", true, "")
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

func main(){
    
    flag.Parse()
    args := flag.Args()
    var err error
    writer := bytes.NewBuffer(nil);
    err = writeUint16(&*writer, vm.MagicNumberXVM)
    errout(err, "Unknown wrong")
    if isStdtoken && (len(args) > 0 || args[0] != "") {
        fileData, err := ioutil.ReadFile(args[0])
        errout(err, "Unable to read file: %s", args[0])
        var inputToken StdToken
        err = json.Unmarshal(fileData, &inputToken)
        errout(err, "Unable to parse json sechme: %s", args[0])
        writer.Write([]byte{0x01})
        writer.Write(common.ZeroHash[:])
        data, err := packStdTokenParams(&inputToken)
        errout(err, "Failed pack params")
        writer.Write(data)
    }
    data := writer.Bytes()
    out := hex.EncodeToString(data)
    fmt.Printf("0x%s\n", out)
}
