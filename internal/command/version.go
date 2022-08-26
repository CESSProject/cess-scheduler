package command

import (
	"cess-scheduler/configs"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Print version number and exit
func Command_Version_Runfunc(cmd *cobra.Command, args []string) {
	fmt.Println(configs.Version)
	os.Exit(0)
}
