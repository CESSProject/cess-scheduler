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
	"os"
)

// FileRouter
type FileRouter struct {
	BaseRouter
}

type MsgFile struct {
	FileHash string `json:"filehash"`
	Data     []byte `json:"data"`
}

// FileRouter Handle
func (this *FileRouter) Handle(ctx context.CancelFunc, request IRequest) {
	fmt.Println("Call FileRouter Handle")

	fmt.Println("recv from client : msgId=", request.GetMsgID())
	if request.GetMsgID() != 2 {
		fmt.Println("MsgId is not 1")
		return
	}

	var msg MsgFile
	err := json.Unmarshal(request.GetData(), &msg)
	if err != nil {
		fmt.Println("Msg format error")
		return
	}

	fs, err := os.OpenFile(msg.FileHash, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		fmt.Println("OpenFile  error")
		return
	}
	fs.Write(msg.Data)
	fs.Sync()
	fs.Close()

	err = request.GetConnection().SendBuffMsg(200, nil)
	if err != nil {
		fmt.Println(err)
	}
}
