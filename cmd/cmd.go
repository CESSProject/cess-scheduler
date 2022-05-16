/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"cess-scheduler/configs"
	"cess-scheduler/internal/chain"
	"cess-scheduler/internal/logger"
	. "cess-scheduler/internal/logger"
	"cess-scheduler/internal/proof"
	"cess-scheduler/internal/rpc"
	"cess-scheduler/tools"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	Name        = "cess-scheduler"
	Description = "Implementation of Scheduling Service for Consensus Nodes"
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
	rootCmd.AddCommand(
		Command_Default(),
		Command_Version(),
		Command_Register(),
		Command_Obtain(),
		Command_Run(),
	)
	rootCmd.PersistentFlags().StringVarP(&configs.ConfigFilePath, "config", "c", "", "Custom profile")
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
		Short:                 "Register scheduler information to the chain",
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
	chain.ChainInit()
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
	chain.ChainInit()
	sd, _, err := chain.GetSchedulerInfoOnChain()
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please try again later. [%v]\n", 41, err)
		os.Exit(1)
	}
	keyring, err := signature.KeyringPairFromSecret(configs.C.CtrlPrk, 0)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please try again later. [%v]\n", 41, err)
		os.Exit(1)
	}
	for _, v := range sd {
		if v.ControllerUser == types.NewAccountID(keyring.PublicKey) {
			reg = true
		}
	}

	if !reg {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Unregistered.\n", 41)
		os.Exit(1)
	}

	hashs, err := tools.CalcHash([]byte(configs.C.CtrlPrk))
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}

	baseDir := filepath.Join(configs.C.DataDir, tools.GetStringWithoutNumbers(hashs), configs.BaseDir)
	f, err := os.Stat(baseDir)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m '%v' not found\n", 41, baseDir)
		os.Exit(1)
	}
	if !f.IsDir() {
		fmt.Printf("\x1b[%dm[err]\x1b[0m '%v' is not a directory\n", 41, baseDir)
		os.Exit(1)
	}
	configs.LogFileDir = filepath.Join(baseDir, configs.LogFileDir)
	configs.FileCacheDir = filepath.Join(baseDir, configs.FileCacheDir)
	configs.DbFileDir = filepath.Join(baseDir, configs.DbFileDir)
	configs.SpaceCacheDir = filepath.Join(baseDir, configs.SpaceCacheDir)
	// start-up
	logger.LoggerInit()
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
	err = viper.Unmarshal(configs.C)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m The '%v' file format error\n", 41, confFilePath)
		os.Exit(1)
	}

	_, err = os.Stat(configs.C.DataDir)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
}

// Scheduler registration function
func register() {
	sd, code, err := chain.GetSchedulerInfoOnChain()
	if err != nil {
		if code != configs.Code_404 {
			fmt.Printf("\x1b[%dm[err]\x1b[0m Please try again later. [%v]\n", 41, err)
			os.Exit(1)
		}
	}
	keyring, err := signature.KeyringPairFromSecret(configs.C.CtrlPrk, 0)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Please try again later. [%v]\n", 41, err)
		os.Exit(1)
	}
	for _, v := range sd {
		if v.ControllerUser == types.NewAccountID(keyring.PublicKey) {
			fmt.Printf("\x1b[%dm[ok]\x1b[0m The account is already registered.\n", 42)
			os.Exit(0)
		}
	}

	eip, err := tools.GetExternalIp()
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}

	hashs, err := tools.CalcHash([]byte(configs.C.CtrlPrk))
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}

	baseDir := filepath.Join(configs.C.DataDir, tools.GetStringWithoutNumbers(hashs), configs.BaseDir)
	_, err = os.Stat(baseDir)
	if err == nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m '%v' directory conflict\n", 41, baseDir)
		os.Exit(1)
	}

	if configs.C.ServiceAddr != "" {
		if eip != configs.C.ServiceAddr {
			fmt.Printf("\x1b[%dm[err]\x1b[0mYou can use \"curl ifconfig.co\" to view the external network ip address\n", 41)
			os.Exit(1)
		}
	}

	res := tools.Base58Encoding(configs.C.ServiceAddr + ":" + configs.C.ServicePort)

	_, err = chain.RegisterToChain(
		configs.C.CtrlPrk,
		chain.ChainTx_FileMap_Add_schedule,
		configs.C.StashAcc,
		res,
	)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Registration failed, Please try again later. [%v]\n", 41, err)
		os.Exit(1)
	}
	fmt.Printf("\x1b[%dm[ok]\x1b[0m Registration success\n", 42)

	configs.LogFileDir = filepath.Join(baseDir, configs.LogFileDir)
	if err = tools.CreatDirIfNotExist(configs.LogFileDir); err != nil {
		goto Err
	}
	configs.FileCacheDir = filepath.Join(baseDir, configs.FileCacheDir)
	if err = tools.CreatDirIfNotExist(configs.FileCacheDir); err != nil {
		goto Err
	}
	configs.DbFileDir = filepath.Join(baseDir, configs.DbFileDir)
	if err = tools.CreatDirIfNotExist(configs.DbFileDir); err != nil {
		goto Err
	}
	configs.SpaceCacheDir = filepath.Join(baseDir, configs.SpaceCacheDir)
	if err = tools.CreatDirIfNotExist(configs.SpaceCacheDir); err != nil {
		goto Err
	}
	logger.LoggerInit()
	Out.Sugar().Infof("Registration message:")
	Out.Sugar().Infof("ChainAddr:%v", configs.C.RpcAddr)
	Out.Sugar().Infof("ServiceAddr:%v", res)
	Out.Sugar().Infof("DataDir:%v", configs.C.DataDir)
	Out.Sugar().Infof("ControllerAccountPhrase:%v", configs.C.CtrlPrk)
	Out.Sugar().Infof("StashAccountAddress:%v", configs.C.StashAcc)
	os.Exit(0)
Err:
	fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
	os.Exit(1)
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
