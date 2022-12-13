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
	"path/filepath"

	"github.com/CESSProject/cess-scheduler/pkg/utils"
)

// FileRouter
type FileRouter struct {
	BaseRouter
	FileDir string
}

type MsgFile struct {
	Token    string `json:"token"`
	FileHash string `json:"filehash"`
	Data     []byte `json:"data"`
}

// FileRouter Handle
func (f *FileRouter) Handle(ctx context.CancelFunc, request IRequest) {
	fmt.Println("Call FileRouter Handle")
	fmt.Println("recv from client : msgId=", request.GetMsgID())

	if request.GetMsgID() != Msg_File {
		fmt.Println("MsgId error")
		ctx()
		return
	}

	var msg MsgFile
	err := json.Unmarshal(request.GetData(), &msg)
	if err != nil {
		fmt.Println("Msg format error")
		ctx()
		return
	}
	fpath := filepath.Join(f.FileDir, msg.FileHash)

	hash, _ := utils.CalcPathSHA256(fpath)
	if hash == msg.FileHash {
		request.GetConnection().SendBuffMsg(Msg_OK_FILE, nil)
		return
	}

	fs, err := os.OpenFile(msg.FileHash, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		fmt.Println("OpenFile  error")
		ctx()
		return
	}
	defer fs.Close()

	fs.Write(msg.Data)
	err = fs.Sync()
	if err != nil {
		fmt.Println("Sync  error")
		ctx()
		return
	}

	err = request.GetConnection().SendMsg(Msg_OK, nil)
	if err != nil {
		fmt.Println(err)
	}
}
