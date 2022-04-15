/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"cess-scheduler/configs"
	"cess-scheduler/initlz"
	"cess-scheduler/internal/chain"
	. "cess-scheduler/internal/logger"
	"cess-scheduler/internal/proof"
	"cess-scheduler/internal/rpc"
	"cess-scheduler/tools"
	"fmt"
	"os"
	"os/signal"

	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	Name        = "cess-scheduler"
	Description = "An implementation of the CESS scheduler for consensus nodes."
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

// init
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
		Use:                   "version",
		Short:                 "Print version information",
		Run:                   Command_Version_Runfunc,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func Command_Default() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "default",
		Short:                 "Generate profile template",
		Run:                   Command_Default_Runfunc,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func Command_Register() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "register",
		Short:                 "Register miner information to cess chain",
		Run:                   Command_Register_Runfunc,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func Command_Obtain() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "obtain <pubkey> <faucet address>",
		Short:                 "Get cess test coin",
		Run:                   Command_Obtain_Runfunc,
		DisableFlagsInUseLine: true,
	}
	return cc
}

func Command_Run() *cobra.Command {
	cc := &cobra.Command{
		Use:                   "run",
		Short:                 "Operation scheduling service",
		Run:                   Command_Run_Runfunc,
		DisableFlagsInUseLine: true,
	}
	return cc
}

// Print version number and exit
func Command_Version_Runfunc(cmd *cobra.Command, args []string) {
	fmt.Println(configs.Version)
	os.Exit(0)
}

// Generate configuration file template
func Command_Default_Runfunc(cmd *cobra.Command, args []string) {
	tools.WriteStringtoFile(configs.ConfigFile_Templete, configs.DefaultConfigurationFileName)
	os.Exit(0)
}

// Scheduler registration
func Command_Register_Runfunc(cmd *cobra.Command, args []string) {
	refreshProfile(cmd)
	initlz.SystemInit()
	register()
}

// obtain tCESS
func Command_Obtain_Runfunc(cmd *cobra.Command, args []string) {
	//refreshProfile(cmd)
	if len(os.Args) < 3 {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please enter the faucet token address.\n", 41)
		os.Exit(1)
	}
	err := chain.ObtainFromFaucet(os.Args[3], os.Args[2])
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err.Error())
		os.Exit(1)
	} else {
		fmt.Println("success")
		os.Exit(0)
	}
}

// start service
func Command_Run_Runfunc(cmd *cobra.Command, args []string) {
	var reg bool
	refreshProfile(cmd)
	// init
	initlz.SystemInit()

	sd, err := chain.GetSchedulerInfoOnChain(configs.ChainModule_FileMap, configs.ChainModule_FileMap_SchedulerInfo)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please try again later. [%v]\n", 41, err)
		os.Exit(1)
	}
	keyring, err := signature.KeyringPairFromSecret(configs.Confile.SchedulerInfo.ControllerAccountPhrase, 0)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please try again later. [%v]\n", 41, err)
		os.Exit(-1)
	}
	for _, v := range sd {
		if v.Acc == types.NewAccountID(keyring.PublicKey) {
			reg = true
		}
	}

	if !reg {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Unregistered.\n", 41)
		os.Exit(1)
	}
	// start-up
	exit_interrupt()
	proof.Chain_Main()

	// rpc service
	rpc.Rpc_Main()
}

// Parse the configuration file
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
		os.Exit(1)
	}
	if f.IsDir() {
		fmt.Printf("\x1b[%dm[err]\x1b[0m The '%v' is not a file\n", 41, confFilePath)
		os.Exit(1)
	}

	viper.SetConfigFile(confFilePath)
	viper.SetConfigType("toml")

	err = viper.ReadInConfig()
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m The '%v' file type error\n", 41, confFilePath)
		os.Exit(1)
	}
	err = viper.Unmarshal(configs.Confile)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m The '%v' file format error\n", 41, confFilePath)
		os.Exit(1)
	}
	//fmt.Println(configs.Confile)
}

// Scheduler registration function
func register() {
	sd, err := chain.GetSchedulerInfoOnChain(configs.ChainModule_FileMap, configs.ChainModule_FileMap_SchedulerInfo)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please try again later. [%v]\n", 41, err)
		os.Exit(1)
	}
	keyring, err := signature.KeyringPairFromSecret(configs.Confile.SchedulerInfo.ControllerAccountPhrase, 0)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please try again later. [%v]\n", 41, err)
		os.Exit(1)
	}
	for _, v := range sd {
		if v.Acc == types.NewAccountID(keyring.PublicKey) {
			fmt.Printf("\x1b[%dm[ok]\x1b[0m The account is already registered.\n", 42)
			os.Exit(0)
		}
	}

	eip, err := tools.GetExternalIp()
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}

	if eip != configs.Confile.SchedulerInfo.ServiceAddr {
		fmt.Printf("\x1b[%dm[err]\x1b[0mYou can use \"curl ifconfig.co\" to view the external network ip address\n", 41)
		os.Exit(1)
	}

	res := tools.Base58Encoding(configs.Confile.SchedulerInfo.ServiceAddr + ":" + configs.Confile.SchedulerInfo.ServicePort)

	Out.Sugar().Infof("Registration message:")
	Out.Sugar().Infof("CessAddr:%v", configs.Confile.CessChain.ChainAddr)
	Out.Sugar().Infof("ServiceAddr:%v", res)
	Out.Sugar().Infof("ControllerAccountPhrase:%v", configs.Confile.SchedulerInfo.ControllerAccountPhrase)
	Out.Sugar().Infof("StashAccountAddress:%v", configs.Confile.SchedulerInfo.StashAccountAddress)

	_, err = chain.RegisterToChain(
		configs.Confile.SchedulerInfo.ControllerAccountPhrase,
		configs.ChainTx_FileMap_Add_schedule,
		configs.Confile.SchedulerInfo.StashAccountAddress,
		res,
	)
	if err != nil {
		Out.Sugar().Infof("Registration failed......,err:%v", err)
		Err.Sugar().Errorf("%v", err)
		fmt.Printf("\x1b[%dm[err]\x1b[0m Registration failed, Please try again later. [%v]\n", 41, err)
		os.Exit(1)
	}
	fmt.Println("success")
	os.Exit(0)
}

// Catch the system unexpected exit signal.
// Execute defer statement.
func exit_interrupt() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	go func() {
		for range signalChan {
			panic(signalChan)
		}
	}()
}
