/*
   Copyright 2022 CESS (Cumulus Encrypted Storage System) authors

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
	"log"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/CESSProject/cess-scheduler/node"
	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/confile"
	"github.com/CESSProject/cess-scheduler/pkg/db"
	"github.com/CESSProject/cess-scheduler/pkg/logger"
	"github.com/CESSProject/cess-scheduler/pkg/serve"
	"github.com/CESSProject/cess-scheduler/pkg/utils"
	"github.com/spf13/cobra"
)

// runCmd is used to start the service
//
// Usage:
//
//	scheduler run
func runCmd(cmd *cobra.Command, args []string) {
	var (
		err      error
		logDir   string
		cacheDir string
		n        = node.New()
	)

	//Build profile instances
	n.Cfile, err = buildConfigFile(cmd)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	//Build chain instance
	n.Chn, err = buildChain(n.Cfile, configs.TimeOut_WaitBlock)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	//Build data directory
	logDir, cacheDir, n.FileDir, err = buildDir(n.Cfile, n.Chn)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	//Build cache instance
	n.Cach, err = buildCache(cacheDir)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	//Build log instance
	n.Logs, err = buildLogs(logDir)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	//Build server instance
	n.Ser = buildServer(
		"Scheduler Server",
		n.Cfile.GetServicePortNum(),
		n.Chn,
		n.Logs,
		n.Cach,
		n.FileDir,
	)

	// run
	n.Run()
}

func buildConfigFile(cmd *cobra.Command) (confile.IConfile, error) {
	var conFilePath string
	configpath1, _ := cmd.Flags().GetString("config")
	configpath2, _ := cmd.Flags().GetString("c")
	if configpath1 != "" {
		conFilePath = configpath1
	} else {
		conFilePath = configpath2
	}

	cfg := confile.NewConfigfile()
	if err := cfg.Parse(conFilePath); err != nil {
		return nil, err
	}
	return cfg, nil
}

func buildChain(cfg confile.IConfile, timeout time.Duration) (chain.IChain, error) {
	var isReg bool
	// connecting chain
	client, err := chain.NewChainClient(cfg.GetRpcAddr(), cfg.GetCtrlPrk(), cfg.GetStashAcc(), timeout)
	if err != nil {
		return nil, err
	}

	// judge the balance
	accountinfo, err := client.GetAccountInfo(client.GetPublicKey())
	if err != nil {
		return nil, err
	}

	if accountinfo.Data.Free.CmpAbs(new(big.Int).SetUint64(configs.MinimumBalance)) == -1 {
		return nil, fmt.Errorf("Account balance is less than %v pico\n", configs.MinimumBalance)
	}

	// sync block
	for {
		ok, err := client.GetSyncStatus()
		if err != nil {
			return nil, err
		}
		if !ok {
			break
		}
		log.Println("In sync block...")
		time.Sleep(configs.BlockInterval)
	}
	log.Println("Complete synchronization of primary network block data")

	// whether to register
	schelist, err := client.GetAllSchedulerInfo()
	if err != nil && err.Error() != chain.ERR_RPC_EMPTY_VALUE.Error() {
		return nil, err
	}

	for _, v := range schelist {
		if v.ControllerUser == client.NewAccountId(client.GetPublicKey()) {
			isReg = true
			break
		}
	}

	// register
	if !isReg {
		if err := register(cfg, client); err != nil {
			return nil, err
		}
	}
	return client, nil
}

func register(cfg confile.IConfile, client chain.IChain) error {
	country, _, _ := utils.ParseCountryFromIp(cfg.GetServiceAddr())

	txhash, err := client.Register(cfg.GetStashAcc(), cfg.GetServiceAddr(), cfg.GetServicePort(), country)
	if err != nil {
		if err.Error() == chain.ERR_RPC_EMPTY_VALUE.Error() {
			return fmt.Errorf("[err] Please check your wallet balance")
		} else {
			if txhash != "" {
				msg := configs.HELP_common + fmt.Sprintf(" %v\n", txhash)
				msg += configs.HELP_register
				return fmt.Errorf("[pending] %v\n", msg)
			}
			return err
		}
	}
	return nil
}

func buildDir(cfg confile.IConfile, client chain.IChain) (string, string, string, error) {
	ctlAccount, err := client.GetCessAccount()
	if err != nil {
		return "", "", "", err
	}
	baseDir := filepath.Join(cfg.GetDataDir(), ctlAccount, configs.BaseDir)

	_, err = os.Stat(baseDir)
	if err != nil {
		err = os.MkdirAll(baseDir, configs.DirPermission)
		if err != nil {
			return "", "", "", err
		}
	}

	logDir := filepath.Join(baseDir, configs.LogDir)
	if err := os.MkdirAll(logDir, configs.DirPermission); err != nil {
		return "", "", "", err
	}

	cacheDir := filepath.Join(baseDir, configs.CacheDir)
	if err := os.MkdirAll(cacheDir, configs.DirPermission); err != nil {
		return "", "", "", err
	}

	fileDir := filepath.Join(baseDir, configs.FileDir)
	if err := os.MkdirAll(fileDir, configs.DirPermission); err != nil {
		return "", "", "", err
	}

	log.Println(baseDir)
	return logDir, cacheDir, fileDir, nil
}

func buildCache(cacheDir string) (db.ICache, error) {
	return db.NewCache(cacheDir, 0, 0, configs.NameSpace)
}

func buildLogs(logDir string) (logger.ILog, error) {
	var logs_info = make(map[string]string)
	for _, v := range configs.LogFiles {
		logs_info[v] = filepath.Join(logDir, v+".log")
	}
	return logger.NewLogs(logs_info)
}

func buildServer(name string, port int, chain chain.IChain, logs logger.ILog, cach db.ICache, filedir string) serve.IServer {
	// NewServer
	s := serve.NewServer(name, "0.0.0.0", port)

	// Configure Routes
	s.AddRouter(serve.Msg_Ping, &serve.PingRouter{})
	s.AddRouter(serve.Msg_Auth, &serve.AuthRouter{Cach: cach})
	s.AddRouter(serve.Msg_File, &serve.FileRouter{Chain: chain, Logs: logs, Cach: cach, FileDir: filedir})
	s.AddRouter(serve.Msg_Progress, &serve.StorageProgressRouter{Cach: cach})
	return s
}
