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
	"log"
	"os"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/spf13/cobra"
)

// Scheduler registration
func registerCmd(cmd *cobra.Command, args []string) {
	refreshProfile(cmd)
	chain.ChainInit()
	register()
}

// Scheduler registration function
func register() {
	sd, err := chain.GetSchedulerInfoOnChain()
	if err != nil {
		if err.Error() != chain.ERR_Empty {
			log.Printf("\x1b[%dm[err]\x1b[0m Please try again later. [%v]\n", 41, err)
			os.Exit(1)
		}
	}
	for _, v := range sd {
		if v.ControllerUser == types.NewAccountID(configs.PublicKey) {
			log.Printf("\x1b[%dm[ok]\x1b[0m The account is already registered.\n", 42)
			os.Exit(0)
		}
	}
	rgst()
	os.Exit(0)
}
