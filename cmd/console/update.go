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

package console

import (
	"cess-scheduler/configs"
	"cess-scheduler/pkg/chain"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"storj.io/common/base58"
)

// Schedule update ip function
func updateCmd(cmd *cobra.Command, args []string) {
	refreshProfile(cmd)
	if len(os.Args) >= 4 {
		_, err := strconv.Atoi(os.Args[3])
		if err != nil {
			log.Printf("\x1b[%dm[err]\x1b[0m Please fill in the correct port number.\n", 41)
			os.Exit(1)
		}
		res := base58.Encode([]byte(os.Args[2] + ":" + os.Args[3]))
		chain.ChainInit()
		txhash, err := chain.UpdatePublicIp(configs.C.CtrlPrk, res)
		if err != nil {
			if err.Error() == chain.ERR_Empty {
				log.Println("[err] Please check your wallet balance.")
			} else {
				if txhash != "" {
					msg := configs.HELP_common + fmt.Sprintf(" %v\n", txhash)
					msg += configs.HELP_update
					log.Printf("[pending] %v\n", msg)
				} else {
					log.Printf("[err] %v.\n", err)
				}
			}
			os.Exit(1)
		}
		log.Printf("\x1b[%dm[ok]\x1b[0m success\n", 42)
		os.Exit(0)
	}
	log.Printf("\x1b[%dm[err]\x1b[0m You should enter something like 'scheduler update ip port'\n", 41)
	os.Exit(1)
}
