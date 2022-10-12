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
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/CESSProject/cess-scheduler/internal/pattern"
	"github.com/CESSProject/cess-scheduler/pkg/logger"
	"github.com/CESSProject/cess-scheduler/pkg/pbc"
	"github.com/CESSProject/cess-scheduler/pkg/utils"
	"github.com/pkg/errors"
)

func task_GenerateFiller(ch chan bool, logs logger.Logger, fillerDir string) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			logs.Log("panic", "error", utils.RecoverError(err))
		}
	}()

	logs.Log("gf", "info", errors.New("-----> Start task_GenerateFiller"))

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
				fillerpath = filepath.Join(fillerDir, fmt.Sprintf("%s", uid))
				_, err = os.Stat(fillerpath)
				if err != nil {
					break
				}
			}
			err = generateFiller(fillerpath, configs.FillerSize)
			if err != nil {
				logs.Log("gf", "error", err)
				os.Remove(fillerpath)
				time.Sleep(time.Second * time.Duration(utils.RandomInRange(5, 30)))
				continue
			}

			fstat, _ := os.Stat(fillerpath)
			if fstat.Size() != configs.FillerSize {
				logs.Log("gf", "error", errors.Errorf("filler size err: %v", err))
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
				logs.Log("gf", "error", err)
				time.Sleep(time.Second * time.Duration(utils.RandomInRange(5, 30)))
				continue
			}

			select {
			case commitResponse = <-commitResponseCh:
			}
			if commitResponse.StatueMsg.StatusCode != pbc.Success {
				os.Remove(fillerpath)
				logs.Log("gf", "error", errors.New("PoDR2ProofCommit false"))
				time.Sleep(time.Second * time.Duration(utils.RandomInRange(5, 30)))
				continue
			}

			var fillerEle pattern.Filler
			fillerEle.FillerId = uid
			fillerEle.Path = fillerpath
			fillerEle.T = commitResponse.T
			fillerEle.Sigmas = commitResponse.Sigmas
			pattern.C_Filler <- fillerEle
			logs.Log("gf", "info", errors.Errorf("Produced a filler: %v", uid))
			time.Sleep(time.Second)
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
