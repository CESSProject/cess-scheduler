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

package chain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewChainClient(t *testing.T) {
	rpcAddr := "wss://testnet-rpc0.cess.cloud/ws/"
	secret := "swear theme bounce soccer hungry gesture hurdle asset typical call balcony wrist"
	stash := "cXfg2SYcq85nyZ1U4ccx6QnAgSeLQB8aXZ2jstbw9CPGSmhXY"
	time := time.Duration(time.Second * time.Duration(20))
	_, err := NewChainClient(rpcAddr, secret, stash, time)
	assert.NoError(t, err)
}
