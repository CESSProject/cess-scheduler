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

package node

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/gin-gonic/gin"
)

// v0.5.4 added
type PoDR2PubData struct {
	Result SigGenResponse `json:"result"`
	Status StatusInfo     `json:"status"`
}

type SigGenResponse struct {
	T           T        `json:"t"`
	Phi         []string `json:"phi"`
	SigRootHash string   `json:"sig_root_hash"`
	Spk         Spk      `json:"spk"`
}

type T struct {
	Tag      Tag    `json:"tag"`
	SigAbove string `json:"sig_above"`
}

type Tag struct {
	Name string `json:"name"`
	N    int64  `json:"n"`
	U    string `json:"u"`
}

type Spk struct {
	E string `json:"e"`
	N string `json:"n"`
}

type StatusInfo struct {
	StatusCode uint   `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

func (n *Node) AddRoute() {
	n.CallBack.POST(configs.GetTagRoute_Callback, n.GetTag)
}

func (n *Node) GetTag(c *gin.Context) {
	var (
		err error
	)
	val, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, nil)
		return
	}

	var result PoDR2PubData
	err = json.Unmarshal(val, &result)
	if err != nil {
		c.JSON(http.StatusBadRequest, nil)
		return
	}
	c.JSON(http.StatusOK, nil)

	if result.Status.StatusCode != configs.SgxReportSuc {
		log.Printf("Sgx processing result failed: %d", result.Status.StatusCode)
		return
	}

	go func() {
		if len(Ch_Tag) == 1 {
			_ = <-Ch_Tag
		}
		Ch_Tag <- result
	}()
}
