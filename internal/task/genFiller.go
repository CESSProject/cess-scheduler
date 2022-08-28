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
	"cess-scheduler/configs"
	"cess-scheduler/internal/pattern"
	"cess-scheduler/pkg/logger"
	"cess-scheduler/pkg/utils"
	"cess-scheduler/tools"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

func task_GenerateFiller(ch chan bool, logs logger.Logger, fillerDir string) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			logs.Log("panic", "err", utils.RecoverError(err))
		}
	}()

	logs.Log("gf", "info", errors.New("-----> Start task_GenerateFiller"))

	var (
		err        error
		uid        string
		fillerpath string
	)
	for {
		for len(pattern.C_Filler) < pattern.C_Filler_Maxlen {
			for {
				uid, _ = tools.GetGuid(int64(tools.RandomInRange(0, 1024)))
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
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 30)))
				continue
			}

			fstat, _ := os.Stat(fillerpath)
			if fstat.Size() != configs.FillerSize {
				logs.Log("gf", "error", errors.Errorf("filler size err: %v", err))
				os.Remove(fillerpath)
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 30)))
				continue
			}

			// call the sgx service interface to get the tag
			// TODO:

			logs.Log("gf", "info", errors.Errorf("Produced a filler: %v", uid))
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
	rows := fsize / 4096
	for i := uint64(0); i < rows; i++ {
		f.WriteString(tools.RandStr(4095) + "\n")
	}
	err = f.Sync()
	if err != nil {
		os.Remove(fpath)
		return err
	}
	return nil
}
