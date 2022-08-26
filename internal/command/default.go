package command

import (
	"cess-scheduler/configs"
	"cess-scheduler/tools"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// Generate configuration file template
func Command_Default_Runfunc(cmd *cobra.Command, args []string) {
	tools.WriteStringtoFile(
		configs.ConfigurationFileTemplete,
		configs.ConfigurationFileTemplateName,
	)
	pwd, err := os.Getwd()
	if err != nil {
		log.Printf("[err] %v\n", err)
		os.Exit(1)
	}
	path := filepath.Join(pwd, configs.ConfigurationFileTemplateName)
	log.Printf("[ok] %v\n", path)
	os.Exit(0)
}
