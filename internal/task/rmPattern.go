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
	"log"
	"math/big"
	"os"
	"time"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/CESSProject/cess-scheduler/internal/pattern"
	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/logger"
	"github.com/CESSProject/cess-scheduler/pkg/utils"
	"github.com/pkg/errors"
)

func task_ClearAuthMap(ch chan bool, logs logger.Logger, c chain.Chainer) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			logs.Log("panic", "error", utils.RecoverError(err))
		}
	}()
	logs.Log("common", "info", errors.New("-----> Start task_ClearAuthMap"))

	var count uint8
	for {
		count++
		if count >= 5 {
			accountinfo, err := c.GetAccountInfo(c.GetPublicKey())
			if err == nil {
				if accountinfo.Data.Free.CmpAbs(new(big.Int).SetUint64(configs.MinimumBalance)) == -1 {
					logs.Log("common", "info", errors.New("Insufficient balance, program exited"))
					log.Printf("Insufficient balance, program exited.\n")
					os.Exit(1)
				}
			}
			count = 0
			logs.Log("common", "info", errors.New("Connected miners:"))
			logs.Log("common", "info", errors.Errorf("%v", pattern.GetConnectedSpacem()))
			logs.Log("common", "info", errors.New("Black miners:"))
			logs.Log("common", "info", errors.Errorf("%v", pattern.GetBlacklist()))
		}
		pattern.DeleteExpiredAuth()
		pattern.DeleteExpiredSpacem()
		time.Sleep(time.Minute)
	}
}
