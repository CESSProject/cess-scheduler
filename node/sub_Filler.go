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
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/CESSProject/cess-scheduler/internal/pattern"
	"github.com/CESSProject/cess-scheduler/pkg/pbc"
	"github.com/CESSProject/cess-scheduler/pkg/utils"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

func (node *Node) task_GenerateFiller(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			node.Logs.Log("panic", "error", utils.RecoverError(err))
		}
	}()

	node.Logs.Log("gf", "info", errors.New("-----> Start task_GenerateFiller"))

	var (
		err        error
		uid        string
		fillerpath string
	)
	for {
		for len(pattern.C_Filler) < configs.Num_Filler_Reserved {
			for {
				time.Sleep(time.Second)
				uid, _ = utils.GetGuid(int64(utils.RandomInRange(0, 1024)))
				if uid == "" {
					continue
				}
				fillerpath = filepath.Join(node.FillerDir, fmt.Sprintf("%s", uid))
				_, err = os.Stat(fillerpath)
				if err != nil {
					break
				}
			}
			err = generateFiller(fillerpath, configs.FillerSize)
			if err != nil {
				node.Logs.Log("gf", "error", err)
				os.Remove(fillerpath)
				time.Sleep(time.Second * time.Duration(utils.RandomInRange(5, 30)))
				continue
			}

			fstat, _ := os.Stat(fillerpath)
			if fstat.Size() != configs.FillerSize {
				node.Logs.Log("gf", "error", errors.Errorf("filler size err: %v", err))
				os.Remove(fillerpath)
				time.Sleep(time.Second * time.Duration(utils.RandomInRange(5, 30)))
				continue
			}

			// call the sgx service interface to get the tag
			// calculate file tag info
			var PoDR2commit pbc.PoDR2Commit
			var commitResponse pbc.PoDR2CommitResponse
			PoDR2commit.FilePath = fillerpath
			PoDR2commit.BlockSize = configs.BlockSize

			commitResponseCh, err := PoDR2commit.PoDR2ProofCommit(
				pbc.Key_Ssk,
				string(pbc.Key_SharedParams),
				int64(configs.ScanBlockSize),
			)
			if err != nil {
				node.Logs.Log("gf", "error", err)
				time.Sleep(time.Second * time.Duration(utils.RandomInRange(5, 30)))
				continue
			}

			select {
			case commitResponse = <-commitResponseCh:
			}
			if commitResponse.StatueMsg.StatusCode != pbc.Success {
				os.Remove(fillerpath)
				node.Logs.Log("gf", "error", errors.New("PoDR2ProofCommit false"))
				time.Sleep(time.Second * time.Duration(utils.RandomInRange(5, 30)))
				continue
			}

			var fillerEle pattern.Filler
			fillerEle.FillerId = uid
			fillerEle.Path = fillerpath
			fillerEle.T = commitResponse.T
			fillerEle.Sigmas = commitResponse.Sigmas
			pattern.C_Filler <- fillerEle
			node.Logs.Log("gf", "info", errors.Errorf("Produced a filler: %v", uid))
			time.Sleep(time.Second)
		}
		time.Sleep(time.Second)
	}
}

func (node *Node) task_SubmitFillerMeta(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			node.Logs.Log("panic", "error", utils.RecoverError(err))
		}
	}()

	node.Logs.Log("sfm", "info", errors.New("-----> Start task_SubmitFillerMeta"))

	var (
		err    error
		txhash string
	)
	t_active := time.Now()
	for {
		time.Sleep(time.Second)
		for len(pattern.C_FillerMeta) > 0 {
			var tmp = <-pattern.C_FillerMeta
			pattern.FillerMap.Add(string(tmp.Acc[:]), tmp)
		}
		if time.Since(t_active).Seconds() > 60 {
			t_active = time.Now()
			for k, v := range pattern.FillerMap.Fillermetas {
				addr, _ := utils.EncodePublicKeyAsCessAccount([]byte(k))
				if len(v) >= 8 {
					txhash, err = node.Chain.SubmitFillerMeta(types.NewAccountID([]byte(k)), v[:8])
					if txhash == "" {
						pattern.ChainStatus.Store(false)
						node.Logs.Log("sfm", "error", err)
						continue
					}
					pattern.ChainStatus.Store(true)
					pattern.FillerMap.Delete(k)
					for i := 0; i < 8; i++ {
						os.Remove(filepath.Join(node.FillerDir, string(v[i].Id)))
					}
					node.Logs.Log("sfm", "info", errors.Errorf("[%v] %v", addr, txhash))
				} else {
					if len(v) > 0 {
						txhash, err = node.Chain.SubmitFillerMeta(types.NewAccountID([]byte(k)), v[:])
						if txhash == "" {
							pattern.ChainStatus.Store(false)
							node.Logs.Log("sfm", "error", err)
							continue
						}
						pattern.ChainStatus.Store(true)
						pattern.FillerMap.Delete(k)
						for _, vv := range v {
							os.Remove(filepath.Join(node.FillerDir, string(vv.Id)))
						}
						node.Logs.Log("sfm", "info", errors.Errorf("[%v] %v", addr, txhash))
					}
				}
			}
		}
	}
}

func generateFiller(fpath string, fsize uint64) error {
	f, err := os.OpenFile(fpath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	rows := fsize / configs.FillerLineLength
	for i := uint64(0); i < rows; i++ {
		f.WriteString(utils.RandStr(configs.FillerLineLength-1) + "\n")
	}
	err = f.Sync()
	if err != nil {
		os.Remove(fpath)
		return err
	}
	return nil
}
