// Copyright 2018 The xfsgo Authors
// This file is part of the xfsgo library.
//
// The xfsgo library is free software: you can redistribute it and/or modify
// it under the terms of the MIT Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The xfsgo library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// MIT Lesser General Public License for more details.
//
// You should have received a copy of the MIT Lesser General Public License
// along with the xfsgo library. If not, see <https://mit-license.org/>.

package common

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/sirupsen/logrus"
	"math"
	"xfsgo/common/ahash"
	"xfsgo/common/rawencode"
)

func MakeStateKey(addr Address, key []byte) []byte {
    return ahash.SHA256(append(addr[:], key...))
}

func ObjSHA256(obj rawencode.RawEncoder) ([]byte, []byte, error) {
	txData, err := rawencode.Encode(obj)
	if err != nil {
		return nil, nil, err
	}
	txHash := ahash.SHA256(txData)
	return txData, txHash, nil
}

func BytesMixed(src []byte, lenBits int, buffer *bytes.Buffer) error {
	srcLen := len(src)
	if uint32(srcLen) > uint32(math.MaxUint32) {
		return errors.New("data to long")
	}
	var lenBuf [4]byte
	lenBuf[0] = uint8(srcLen & 0xff)
	lenBuf[1] = uint8((srcLen & 0xff00) >> 8)
	lenBuf[2] = uint8((srcLen & 0xff0000) >> 16)
	lenBuf[3] = uint8((srcLen & 0xff000000) >> 32)
	buffer.Write(lenBuf[0:lenBits])
	buffer.Write(src)
	return nil
}

func ReadMixedBytes(buf *bytes.Buffer) ([]byte, error) {
	dataLenB, err := buf.ReadByte()
	if err != nil {
		return nil, err
	}
	dataLen := int(dataLenB)
	var dst = make([]byte, dataLen)
	_, err = buf.Read(dst)
	if err != nil {
		return nil, err
	}
	return dst, nil
}

func Safeclose(fn func() error) {
	if err := fn(); err != nil {
		logrus.Error(err)
	}
}

func Objcopy(src interface{}, dst interface{}) error {
	if src == nil {
		return nil
	}
	var err error
	bs, err := json.Marshal(src)
	if err != nil {
		return err
	}
	err = json.Unmarshal(bs, dst)
	return err
}
