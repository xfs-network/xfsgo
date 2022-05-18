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

package api

import (
	"xfsgo"
)

type NFTApiHandler struct {
	BlockChain    *xfsgo.BlockChain
	TxPendingPool *xfsgo.TxPool
}

type NFTCreateArgs struct {
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	Decimals    string `json:"decimals"`
	TotalSupply string `json:"total_supply"`
}

func (handler *NFTApiHandler) Create(args NFTCreateArgs, resp **string) error {
	*resp = nil
	return nil
}
