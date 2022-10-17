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
	"time"

	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/rpc"
	"github.com/CESSProject/cess-scheduler/pkg/utils"
	"github.com/pkg/errors"
)

func (node *Node) task_SyncMinersInfo(ch chan bool) {
	defer func() {
		if err := recover(); err != nil {
			node.Logs.Log("panic", "error", utils.RecoverError(err))
		}
		ch <- true
	}()
	node.Logs.Log("smi", "info", errors.New("-----> Start task_SyncMinersInfo"))

	for {
		allMinerAcc, _ := node.Chain.GetAllStorageMiner()
		if len(allMinerAcc) == 0 {
			time.Sleep(time.Second * 6)
			continue
		}

		for i := 0; i < len(allMinerAcc); i++ {
			addr, err := utils.EncodePublicKeyAsCessAccount(allMinerAcc[i][:])
			if err != nil {
				node.Logs.Log("smi", "error", errors.Errorf("%v, %v", allMinerAcc[i], err))
				continue
			}

			ok, err := node.Cache.Has(allMinerAcc[i][:])
			if err != nil {
				node.Logs.Log("smi", "error", errors.Errorf("[%v] %v", addr, err))
				continue
			}

			var cm chain.Cache_MinerInfo
			mdata, err := node.Chain.GetStorageMinerInfo(allMinerAcc[i][:])
			if err != nil {
				node.Logs.Log("smi", "error", errors.Errorf("[%v] %v", addr, err))
				continue
			}

			if ok {
				if string(mdata.State) == "exit" {
					node.Cache.Delete(allMinerAcc[i][:])
					continue
				}

				err = rpc.Dial(string(mdata.Ip), time.Duration(time.Second*5))
				if err != nil {
					node.Logs.Log("smi", "error", errors.Errorf("[%v] %v", addr, err))
					node.Cache.Delete(allMinerAcc[i][:])
				}
				cm.Peerid = uint64(mdata.PeerId)
				cm.Ip = string(mdata.Ip)
				value, err := json.Marshal(&cm)
				if err != nil {
					node.Logs.Log("smi", "error", errors.Errorf("[%v] %v", addr, err))
					continue
				}
				err = node.Cache.Put(allMinerAcc[i][:], value)
				if err != nil {
					node.Logs.Log("smi", "error", errors.Errorf("[%v] %v", addr, err))
				}
				continue
			}

			if string(mdata.State) == "exit" {
				continue
			}

			err = rpc.Dial(string(mdata.Ip), time.Duration(time.Second*5))
			if err != nil {
				node.Logs.Log("smi", "error", errors.Errorf("[%v] %v", addr, err))
				continue
			}

			cm.Peerid = uint64(mdata.PeerId)
			cm.Ip = string(mdata.Ip)

			value, err := json.Marshal(&cm)
			if err != nil {
				node.Logs.Log("smi", "error", errors.Errorf("[%v] %v", addr, err))
				continue
			}

			err = node.Cache.Put(allMinerAcc[i][:], value)
			if err != nil {
				node.Logs.Log("smi", "error", errors.Errorf("[%v] %v", addr, err))
			}

			node.Logs.Log("smi", "info", errors.Errorf("[%v] Cache is stored", addr))
			//pattern.DeleteBliacklist(string(allMinerAcc[i][:]))
		}
		time.Sleep(time.Second)
	}
}
