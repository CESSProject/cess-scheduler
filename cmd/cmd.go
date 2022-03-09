/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"os"
	"scheduler-mining/configs"
	"scheduler-mining/initlz"
	"scheduler-mining/internal/chain"
	"scheduler-mining/internal/logger"
	"scheduler-mining/internal/proof"
	"scheduler-mining/rpc"
	"scheduler-mining/tools"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	Name        = "schdl-ser"
	Description = "Scheduler service of CESS platform"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   Name,
	Short: Description,
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

func init() {
	rootCmd.PersistentFlags().StringVarP(&configs.ConfigFilePath, "config", "c", "", "Custom profile")
	rootCmd.AddCommand(
		Command_Default(),
		Command_Version(),
		Command_Register(),
		Command_Obtain(),
		Command_Run(),
	)
}

func Command_Version() *cobra.Command {
	cc := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run:   Command_Version_Runfunc,
	}
	return cc
}

func Command_Default() *cobra.Command {
	cc := &cobra.Command{
		Use:   "default",
		Short: "Generate profile template",
		Run:   Command_Default_Runfunc,
	}
	return cc
}

func Command_Register() *cobra.Command {
	cc := &cobra.Command{
		Use:   "register",
		Short: "Register miner information to cess chain",
		Run:   Command_Register_Runfunc,
	}
	return cc
}

func Command_State() *cobra.Command {
	cc := &cobra.Command{
		Use:   "state",
		Short: "List miners' own information",
		Run:   Command_State_Runfunc,
	}
	return cc
}

func Command_Obtain() *cobra.Command {
	cc := &cobra.Command{
		Use:   "obtain",
		Short: "Get cess test coin",
		Run:   Command_Obtain_Runfunc,
	}
	return cc
}

func Command_Run() *cobra.Command {
	cc := &cobra.Command{
		Use:   "run",
		Short: "Operation scheduling service",
		Run:   Command_Run_Runfunc,
	}
	return cc
}

func Command_Version_Runfunc(cmd *cobra.Command, args []string) {
	fmt.Println(configs.Version)
	os.Exit(0)
}

func Command_Default_Runfunc(cmd *cobra.Command, args []string) {
	tools.WriteStringtoFile(configs.ConfigFile_Templete, configs.DefaultConfigurationFileName)
	os.Exit(0)
}

func Command_Register_Runfunc(cmd *cobra.Command, args []string) {
	refreshProfile(cmd)
	register()
}

func Command_State_Runfunc(cmd *cobra.Command, args []string) {
	//TODO
	refreshProfile(cmd)
}

func Command_Obtain_Runfunc(cmd *cobra.Command, args []string) {
	//TODO
	refreshProfile(cmd)
}

func Command_Run_Runfunc(cmd *cobra.Command, args []string) {
	//TODO
	refreshProfile(cmd)
	// init
	initlz.SystemInit()

	// start-up
	proof.Chain_Main()

	// rpc service
	rpc.Rpc_Main()
}

//
func refreshProfile(cmd *cobra.Command) {
	configpath1, _ := cmd.Flags().GetString("config")
	configpath2, _ := cmd.Flags().GetString("c")
	if configpath1 != "" {
		configs.ConfigFilePath = configpath1
	} else {
		configs.ConfigFilePath = configpath2
	}
	parseProfile()
}

func parseProfile() {
	var (
		err          error
		confFilePath string
	)
	if configs.ConfigFilePath == "" {
		confFilePath = "./conf.toml"
	} else {
		confFilePath = configs.ConfigFilePath
	}

	f, err := os.Stat(confFilePath)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m The '%v' file does not exist\n", 41, confFilePath)
		os.Exit(configs.Exit_ConfFileNotExist)
	}
	if f.IsDir() {
		fmt.Printf("\x1b[%dm[err]\x1b[0m The '%v' is not a file\n", 41, confFilePath)
		os.Exit(configs.Exit_ConfFileNotExist)
	}

	viper.SetConfigFile(confFilePath)
	viper.SetConfigType("toml")

	err = viper.ReadInConfig()
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m The '%v' file type error\n", 41, confFilePath)
		os.Exit(configs.Exit_ConfFileTypeError)
	}
	err = viper.Unmarshal(configs.Confile)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m The '%v' file format error\n", 41, confFilePath)
		os.Exit(configs.Exit_ConfFileFormatError)
	}
	fmt.Println(configs.Confile)
}

//
func register() {

	res := tools.Base58Encoding(configs.Confile.SchedulerInfo.ServiceAddr + ":" + configs.Confile.SchedulerInfo.ServicePort)

	logger.InfoLogger.Sugar().Infof("Start registration......\n    CessAddr:%v\n    ServiceAddr:%v\n    TransactionPrK:%v\n",
		configs.Confile.CessChain.ChainAddr, res, configs.Confile.SchedulerInfo.TransactionPrK)

	ok, err := chain.RegisterToChain(
		configs.Confile.SchedulerInfo.TransactionPrK,
		configs.ChainTx_FileMap_Add_schedule,
		res,
	)
	if !ok || err != nil {
		logger.InfoLogger.Sugar().Infof("Registration failed......,err:%v", err)
		logger.ErrLogger.Sugar().Errorf("%v", err)
		fmt.Printf("\x1b[%dm[err]\x1b[0m Registration failed, Please try again later. [%v]\n", 41, err)
		os.Exit(-1)
	}
	fmt.Println("success")
	os.Exit(0)
}
