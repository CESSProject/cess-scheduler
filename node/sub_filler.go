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
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/CESSProject/cess-scheduler/pkg/pbc"
	"github.com/CESSProject/cess-scheduler/pkg/utils"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

// task_GenerateFiller is used to generate filler
// and store it in the channel for standby
func (node *Node) task_GenerateFiller(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			node.Logs.Pnc("error", utils.RecoverError(err))
		}
	}()
	var (
		err           error
		fillerpath    string
		newFillerPath string
		fillerHash    string
		tagPath       string
		fillerEle     Filler
	)
	node.Logs.GenFiller("info", errors.New(">>> Start task_GenerateFiller <<<"))
	for {
		for len(C_Filler) < configs.Num_Filler_Reserved {
			// calc filler path
			fillerpath = filepath.Join(node.FillerDir, fmt.Sprintf("%v", time.Now().UnixNano()))
			node.Logs.GenFiller("info", fmt.Errorf("%v", fillerpath))
			_, err = os.Stat(fillerpath)
			if err == nil {
				time.Sleep(time.Second)
				continue
			}

			// generate filler
			err = generateFiller(fillerpath, configs.FillerSize)
			if err != nil {
				node.Logs.GenFiller("error", err)
				os.Remove(fillerpath)
				continue
			}

			// judge filler size
			fstat, _ := os.Stat(fillerpath)
			if fstat.Size() != configs.FillerSize {
				node.Logs.GenFiller("error", fmt.Errorf("filler size err: %v", err))
				os.Remove(fillerpath)
				continue
			}

			// calculate filler tag
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
				node.Logs.GenFiller("error", err)
				os.Remove(fillerpath)
				continue
			}

			select {
			case commitResponse = <-commitResponseCh:
			}
			if commitResponse.StatueMsg.StatusCode != pbc.Success {
				os.Remove(fillerpath)
				continue
			}

			// calc filler hash
			fillerHash, err = utils.CalcPathSHA256(fillerpath)
			if err != nil {
				os.Remove(fillerpath)
				continue
			}

			// rename filler
			newFillerPath = filepath.Join(node.FillerDir, fillerHash)
			err = os.Rename(fillerpath, newFillerPath)
			if err != nil {
				os.Remove(fillerpath)
				continue
			}

			// filler tag
			tagPath = newFillerPath + ".tag"
			err = generateFillerTag(tagPath, commitResponse.T, commitResponse.Sigmas)
			if err != nil {
				os.Remove(newFillerPath)
				os.Remove(tagPath)
				continue
			}

			// save filler metainfo to channel
			fillerEle.Hash = fillerHash
			fillerEle.FillerPath = newFillerPath
			fillerEle.TagPath = tagPath
			C_Filler <- fillerEle
			node.Logs.GenFiller("info", fmt.Errorf("Produced a filler: %v", fillerHash))
		}
		time.Sleep(time.Second)
	}
}

// task_SubmitFillerMeta records the fillermeta on the chain
func (node *Node) task_SubmitFillerMeta(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			node.Logs.Pnc("error", utils.RecoverError(err))
		}
	}()
	var (
		err    error
		count  int
		txhash string
	)
	node.Logs.FillerMeta("info", errors.New(">>> Start task_SubmitFillerMeta <<<"))

	t_active := time.Now()
	for {
		time.Sleep(time.Second)
		for len(C_FillerMeta) > 0 {
			var tmp = <-C_FillerMeta
			FillerMap.Add(string(tmp.Acc[:]), tmp)
		}
		if time.Since(t_active).Seconds() > configs.SubmitFillermetaInterval {
			t_active = time.Now()
			for k, v := range FillerMap.Fillermetas {
				addr, _ := utils.EncodePublicKeyAsCessAccount([]byte(k))
				if len(v) > 0 {
					count = 0
					if len(v) >= configs.Max_SubFillerMeta {
						count = configs.Max_SubFillerMeta
					} else {
						count = len(v)
					}
					txhash, err = node.Chain.SubmitFillerMeta(types.NewAccountID([]byte(k)), v[:count])
					if txhash == "" {
						node.Logs.FillerMeta("error", err)
						time.Sleep(configs.BlockInterval)
						continue
					}
					FillerMap.Delete(k)
					for i := 0; i < count; i++ {
						os.Remove(filepath.Join(node.FillerDir, string(v[i].Hash[:])))
					}
					node.Logs.FillerMeta("info", fmt.Errorf("[%v] %v", addr, txhash))
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

func generateFillerTag(fpath string, fileTagT pbc.FileTagT, sigmas [][]byte) error {
	tagFs, err := os.OpenFile(fpath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer tagFs.Close()
	tag_data := TagInfo{
		T:      fileTagT,
		Sigmas: sigmas,
	}
	tag_data_b, err := json.Marshal(&tag_data)
	if err != nil {
		return err
	}
	tagFs.Write(tag_data_b)
	err = tagFs.Sync()
	if err != nil {
		return err
	}
	return nil
}
