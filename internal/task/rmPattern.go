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
	"github.com/CESSProject/cess-scheduler/tools"
)

func task_ClearAuthMap(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()
	Com.Info("-----> Start task_ClearAuthMap")
	var count uint8
	for {
		count++
		if count >= 5 {
			accountinfo, err := chain.GetAccountInfo(configs.PublicKey)
			if err == nil {
				if accountinfo.Data.Free.CmpAbs(new(big.Int).SetUint64(2000000000000)) == -1 {
					Com.Info("Insufficient balance, program exited.")
					log.Printf("Insufficient balance, program exited.\n")
					os.Exit(1)
				}
			}
			count = 0
			Com.Info("Connected miners:")
			Com.Sugar().Info(pattern.GetConnectedSpacem())
			Com.Info("Black miners:")
			Com.Sugar().Info(pattern.GetBlacklist())
		}
		time.Sleep(time.Minute)
		pattern.DeleteExpiredAuth()
		pattern.DeleteExpiredSpacem()
	}
}
