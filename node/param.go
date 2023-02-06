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
	"sync"
	"time"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/CESSProject/cess-scheduler/pkg/pbc"
)

type TagInfo struct {
	T           pbc.T
	Phi         []pbc.Sigma `json:"phi"`           //Φ = {σi}
	SigRootHash []byte      `json:"sig_root_hash"` //BLS

}

type Filler struct {
	Hash       string
	FillerPath string
	TagPath    string
}

type BlacklistMiner struct {
	Lock *sync.Mutex
	List map[uint64]int64
}

type FileStoreInfo struct {
	FileId      string         `json:"file_id"`
	FileState   string         `json:"file_state"`
	Scheduler   string         `json:"scheduler"`
	FileSize    int64          `json:"file_size"`
	IsUpload    bool           `json:"is_upload"`
	IsCheck     bool           `json:"is_check"`
	IsShard     bool           `json:"is_shard"`
	IsScheduler bool           `json:"is_scheduler"`
	Miners      map[int]string `json:"miners,omitempty"`
}

var (
	C_Filler    chan Filler
	blackMiners *BlacklistMiner
)

func init() {
	C_Filler = make(chan Filler, configs.Num_Filler_Reserved)
	blackMiners = &BlacklistMiner{
		Lock: new(sync.Mutex),
		List: make(map[uint64]int64, 100),
	}
}

func (b *BlacklistMiner) Add(peerid uint64) {
	b.Lock.Lock()
	b.List[peerid] = time.Now().Unix()
	b.Lock.Unlock()
}

func (b *BlacklistMiner) Delete(peerid uint64) {
	b.Lock.Lock()
	delete(b.List, peerid)
	b.Lock.Unlock()
}

func (b *BlacklistMiner) IsExist(peerid uint64) bool {
	b.Lock.Lock()
	defer b.Lock.Unlock()
	v, ok := b.List[peerid]
	if !ok {
		return false
	}
	if time.Since(time.Unix(v, 0)).Seconds() > 10 {
		delete(b.List, peerid)
		return false
	}
	return true
}
