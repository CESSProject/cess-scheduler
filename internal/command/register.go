package command

import (
	"cess-scheduler/configs"
	"cess-scheduler/internal/chain"
	"log"
	"os"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/spf13/cobra"
)

// Scheduler registration
func Command_Register_Runfunc(cmd *cobra.Command, args []string) {
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
