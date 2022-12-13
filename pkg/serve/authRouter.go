/*
   Copyright 2022 CESS (Cumulus Encrypted Storage System) authors

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

package serve

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/CESSProject/cess-scheduler/pkg/db"
	"github.com/CESSProject/cess-scheduler/pkg/utils"
)

type AuthRouter struct {
	BaseRouter
	Cache db.Cacher
}

type MsgAuth struct {
	Account string `json:"account"`
	Msg     string `json:"msg"`
	Sign    []byte `json:"sign"`
}

// AuthRouter Handle
func (this *AuthRouter) Handle(ctx context.CancelFunc, request IRequest) {
	fmt.Println("Call AuthRouter Handle")
	fmt.Println("recv from client : msgId=", request.GetMsgID())
	if request.GetMsgID() != Msg_Auth {
		fmt.Println("MsgId error")
		ctx()
		return
	}

	remote := request.GetConnection().RemoteAddr().String()
	val, err := this.Cache.Get([]byte(remote))
	if err != nil {
		this.Cache.Put([]byte(remote), utils.Int64ToBytes(time.Now().Unix()))
	} else {
		if time.Since(time.Unix(utils.BytesToInt64(val), 0)).Minutes() < 1 {
			ctx()
			return
		} else {
			this.Cache.Delete([]byte(remote))
		}
	}

	var msg MsgAuth
	err = json.Unmarshal(request.GetData(), &msg)
	if err != nil {
		ctx()
		return
	}

	puk, err := utils.DecodePublicKeyOfCessAccount(msg.Account)
	if err != nil {
		puk, err = utils.DecodePublicKeyOfSubstrateAccount(msg.Account)
		if err != nil {
			ctx()
			return
		}
	}

	ok, err := VerifySign(puk, []byte(msg.Msg), msg.Sign)
	if err != nil || !ok {
		ctx()
		return
	}

	token := utils.GetRandomcode(configs.TokenLength)
	err = request.GetConnection().SendMsg(Msg_OK, []byte(token))
	if err != nil {
		ctx()
		return
	}
	Tokens.Add(token)
}
