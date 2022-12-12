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
	"github.com/pkg/errors"
)

// task_GenerateFiller is used to generate filler
// and store it in the channel for standby
func (n *Node) task_GenerateFiller(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Logs.Pnc("error", utils.RecoverError(err))
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
	n.Logs.GenFiller("info", errors.New(">>> Start task_GenerateFiller <<<"))
	for {
		for len(C_Filler) < configs.Num_Filler_Reserved {
			// calc filler path
			fillerpath = filepath.Join(n.FillerDir, fmt.Sprintf("%v", time.Now().UnixNano()))
			n.Logs.GenFiller("info", fmt.Errorf("%v", fillerpath))
			_, err = os.Stat(fillerpath)
			if err == nil {
				time.Sleep(time.Second)
				continue
			}

			// generate filler
			err = generateFiller(fillerpath, configs.FillerSize)
			if err != nil {
				n.Logs.GenFiller("error", err)
				os.Remove(fillerpath)
				continue
			}

			// judge filler size
			fstat, _ := os.Stat(fillerpath)
			if fstat.Size() != configs.FillerSize {
				n.Logs.GenFiller("error", fmt.Errorf("filler size err: %v", err))
				os.Remove(fillerpath)
				continue
			}

			// calculate file tag info
			var commitResponse pbc.SigGenResponse
			matrix, num := pbc.SplitV2(fillerpath, configs.SIZE_1MiB)

			select {
			case commitResponse = <-pbc.PbcKey.SigGen(matrix, num):
			}

			if commitResponse.StatueMsg.StatusCode != pbc.Success {
				n.Logs.Upfile("error", fmt.Errorf("[%v] Failed to calculate the file tag", fillerpath))
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
			newFillerPath = filepath.Join(n.FillerDir, fillerHash)
			err = os.Rename(fillerpath, newFillerPath)
			if err != nil {
				os.Remove(fillerpath)
				continue
			}

			// filler tag
			tagPath = newFillerPath + ".tag"
			err = generateFillerTag(tagPath, commitResponse)
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
			n.Logs.GenFiller("info", fmt.Errorf("Produced a filler: %v", fillerHash))
		}
		time.Sleep(time.Second)
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

func generateFillerTag(fpath string, fileTag pbc.SigGenResponse) error {
	tagFs, err := os.OpenFile(fpath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer tagFs.Close()
	tag_data := TagInfo{
		T:           fileTag.T,
		Phi:         fileTag.Phi,
		SigRootHash: fileTag.SigRootHash,
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
