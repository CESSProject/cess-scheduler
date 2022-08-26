package command

import (
	"cess-scheduler/configs"
	"cess-scheduler/internal/chain"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"storj.io/common/base58"
)

// Schedule update ip function
func Command_Update_Runfunc(cmd *cobra.Command, args []string) {
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
