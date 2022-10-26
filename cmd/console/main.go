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
	"fmt"
	"os"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   configs.Name,
	Short: configs.Description,
}

// init
func init() {
	rootCmd.AddCommand(
		versionCommand(),
		defaultCommand(),
		runCommand(),
		updateCommand(),
	)
	rootCmd.PersistentFlags().StringP("config", "c", "", "Specify the configuration file")
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func versionCommand() *cobra.Command {
	cc := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(configs.Version)
			os.Exit(0)
		},
		DisableFlagsInUseLine: true,
	}
	return cc
}

func defaultCommand() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "default",
		Short:                 "Generate configuration file template",
		Run:                   defaultCmd,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func runCommand() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "run",
		Short:                 "Operation scheduling service",
		Run:                   runCmd,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func updateCommand() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "update <ipv4> <port>",
		Short:                 "Update scheduling service ip and port",
		Run:                   updateCmd,
		DisableFlagsInUseLine: true,
	}
	return cc
}
