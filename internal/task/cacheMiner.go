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

package task

import (
	"encoding/json"
	"time"

	"github.com/CESSProject/cess-scheduler/internal/pattern"
	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/db"
	"github.com/CESSProject/cess-scheduler/pkg/logger"
	"github.com/CESSProject/cess-scheduler/pkg/rpc"
	"github.com/CESSProject/cess-scheduler/pkg/utils"
	"github.com/pkg/errors"
)

func task_SyncMinersInfo(
	ch chan bool,
	logs logger.Logger,
	cli chain.Chainer,
	db db.Cache,
) {
	defer func() {
		if err := recover(); err != nil {
			logs.Log("panic", "error", utils.RecoverError(err))
		}
		ch <- true
	}()
	logs.Log("smi", "info", errors.New("-----> Start task_SyncMinersInfo"))

	for {
		allMinerAcc, _ := cli.GetAllStorageMiner()
		if len(allMinerAcc) == 0 {
			time.Sleep(time.Second * 6)
			continue
		}

		for i := 0; i < len(allMinerAcc); i++ {
			addr, err := utils.EncodePublicKeyAsCessAccount(allMinerAcc[i][:])
			if err != nil {
				logs.Log("smi", "error", errors.Errorf("%v, %v", allMinerAcc[i], err))
				continue
			}

			ok, err := db.Has(allMinerAcc[i][:])
			if err != nil {
				logs.Log("smi", "error", errors.Errorf("[%v] %v", addr, err))
				continue
			}

			var cm chain.Cache_MinerInfo
			mdata, err := cli.GetStorageMinerInfo(allMinerAcc[i][:])
			if err != nil {
				logs.Log("smi", "error", errors.Errorf("[%v] %v", addr, err))
				continue
			}

			if ok {
				err = rpc.Dial(string(mdata.Ip), time.Duration(time.Second*5))
				if err != nil {
					logs.Log("smi", "error", errors.Errorf("[%v] %v", addr, err))
					db.Delete(allMinerAcc[i][:])
				}
				cm.Peerid = uint64(mdata.PeerId)
				cm.Ip = string(mdata.Ip)
				cm.Pubkey = allMinerAcc[i][:]
				value, err := json.Marshal(&cm)
				if err != nil {
					logs.Log("smi", "error", errors.Errorf("[%v] %v", addr, err))
					continue
				}
				err = db.Put(allMinerAcc[i][:], value)
				if err != nil {
					logs.Log("smi", "error", errors.Errorf("[%v] %v", addr, err))
				}
				continue
			}

			if string(mdata.State) == "exit" {
				continue
			}

			err = rpc.Dial(string(mdata.Ip), time.Duration(time.Second*5))
			if err != nil {
				logs.Log("smi", "error", errors.Errorf("[%v] %v", addr, err))
				continue
			}

			cm.Peerid = uint64(mdata.PeerId)
			cm.Ip = string(mdata.Ip)
			cm.Pubkey = allMinerAcc[i][:]

			value, err := json.Marshal(&cm)
			if err != nil {
				logs.Log("smi", "error", errors.Errorf("[%v] %v", addr, err))
				continue
			}

			err = db.Put(allMinerAcc[i][:], value)
			if err != nil {
				logs.Log("smi", "error", errors.Errorf("[%v] %v", addr, err))
			}

			logs.Log("smi", "info", errors.Errorf("[%v] Cache is stored", addr))
			pattern.DeleteBliacklist(string(allMinerAcc[i][:]))
		}
		time.Sleep(time.Second * time.Duration(utils.RandomInRange(60, 180)))
	}
}
