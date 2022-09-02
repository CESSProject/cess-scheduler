/*
   Copyright 2022 CESS scheduler authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

        http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package com

import (
	"context"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/ethereum/go-ethereum/rpc"
)

func Dial(url string) error {
	wsURL := "ws://" + string(base58.Decode(url))
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	cli, err := rpc.DialWebsocket(ctx, wsURL, "")
	if err != nil {
		return err
	}
	cli.Close()
	return nil
}
