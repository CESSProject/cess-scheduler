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
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-scheduler/internal/pattern"
	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/logger"
	"github.com/CESSProject/cess-scheduler/pkg/utils"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

func task_SubmitFillerMeta(
	ch chan bool,
	logs logger.Logger,
	cli chain.Chainer,
	fillerDir string,
) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			logs.Log("panic", "error", utils.RecoverError(err))
		}
	}()

	logs.Log("sfm", "info", errors.New("-----> Start task_SubmitFillerMeta"))

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
		if time.Since(t_active).Seconds() > 10 {
			t_active = time.Now()
			for k, v := range pattern.FillerMap.Fillermetas {
				addr, _ := utils.EncodePublicKeyAsCessAccount([]byte(k))
				if len(v) >= 8 {
					txhash, err = cli.SubmitFillerMeta(types.NewAccountID([]byte(k)), v[:8])
					if txhash == "" {
						logs.Log("sfm", "error", err)
						continue
					}
					pattern.FillerMap.Delete(k)
					pattern.DeleteSpacemap(k)
					for i := 0; i < 8; i++ {
						os.Remove(filepath.Join(fillerDir, string(v[i].Id)))
					}
					logs.Log("sfm", "info", errors.Errorf("[%v] %v", addr, txhash))
				} else {
					ok := pattern.IsExitSpacem(k)
					if !ok && len(v) > 0 {
						txhash, err = cli.SubmitFillerMeta(types.NewAccountID([]byte(k)), v[:])
						if txhash == "" {
							logs.Log("sfm", "error", err)
							continue
						}
						pattern.FillerMap.Delete(k)
						for _, vv := range v {
							os.Remove(filepath.Join(fillerDir, string(vv.Id)))
						}
						logs.Log("sfm", "info", errors.Errorf("[%v] %v", addr, txhash))
					}
				}
			}
		}
	}
}
