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
	"log"
	"math/big"
	"os"
	"time"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/CESSProject/cess-scheduler/pkg/utils"
	"github.com/pkg/errors"
)

// task_ Common is used to judge whether the balance of
// your wallet meets the operation requirements.
func (n *Node) task_common(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Logs.Pnc("error", utils.RecoverError(err))
		}
	}()

	n.Logs.Common("info", errors.New(">>> Start task_common <<<"))

	tikBalance := time.NewTicker(time.Minute)
	defer tikBalance.Stop()

	for {
		select {
		case <-tikBalance.C:
			accountinfo, err := n.Chn.GetAccountInfo(n.Chn.GetPublicKey())
			if err == nil {
				if accountinfo.Data.Free.CmpAbs(new(big.Int).SetUint64(configs.MinimumBalance)) == -1 {
					n.Logs.Common("info", errors.New("Insufficient balance, program exited"))
					log.Printf("Insufficient balance, program exited.\n")
					os.Exit(1)
				}
			}
		}
	}
}
