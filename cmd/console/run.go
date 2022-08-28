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
	"cess-scheduler/internal/task"
	"cess-scheduler/pkg/chain"
	"cess-scheduler/pkg/configfile"
	"cess-scheduler/pkg/db"
	"cess-scheduler/pkg/logger"
	"cess-scheduler/pkg/rpc"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/spf13/cobra"
	"storj.io/common/base58"
)

// runCmd is used to start the scheduling service
func runCmd(cmd *cobra.Command, args []string) {
	var isReg bool
	// config file
	var configFilePath string
	configpath1, _ := cmd.Flags().GetString("config")
	configpath2, _ := cmd.Flags().GetString("c")
	if configpath1 != "" {
		configFilePath = configpath1
	} else {
		configFilePath = configpath2
	}

	confile := configfile.NewConfigfile()
	if err := confile.Parse(configFilePath); err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// chain client
	c, err := chain.NewChainClient(
		confile.GetRpcAddr(),
		confile.GetCtrlPrk(),
		time.Duration(time.Second*15),
	)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// judge the balance
	accountinfo, err := c.GetAccountInfo()
	if err != nil {
		log.Printf("Failed to get account information.\n")
		os.Exit(1)
	}
	if accountinfo.Data.Free.CmpAbs(
		new(big.Int).SetUint64(configs.MinimumBalance),
	) == -1 {
		log.Printf("Account balance is less than %v pico\n", configs.MinimumBalance)
		os.Exit(1)
	}

	// whether to register
	schelist, err := c.GetSchedulerInfo()
	if err != nil {
		if err.Error() != chain.ERR_Empty {
			log.Printf("%v\n", err)
			os.Exit(1)
		}
	} else {
		for _, v := range schelist {
			if v.ControllerUser == types.NewAccountID(c.GetPublicKey()) {
				isReg = true
				break
			}
		}
	}

	// register
	if !isReg {
		if err := register(confile, c); err != nil {
			os.Exit(1)
		}
	}

	// create data dir
	logDir, dbDir, fillerDir, err := creatDataDir(confile, c)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// cache
	db, err := db.NewLevelDB(dbDir, 0, 0, configs.NameSpace)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	// logs
	var logs_info = make(map[string]string)
	for _, v := range configs.LogName {
		logs_info[v] = filepath.Join(logDir, v+".log")
	}
	logs, err := logger.NewLogs(logs_info)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}

	go task.Run(confile, c, db, logs, fillerDir)
	rpc.Rpc_Main()
}

func register(confile configfile.Configfiler, c chain.Chainer) error {
	txhash, err := c.Register(
		confile.GetStashAcc(),
		base58.Encode(
			[]byte(confile.GetServiceAddr()+":"+confile.GetServicePort()),
		),
	)
	if err != nil {
		if err.Error() == chain.ERR_Empty {
			log.Println("[err] Please check your wallet balance.")
		} else {
			if txhash != "" {
				msg := configs.HELP_common + fmt.Sprintf(" %v\n", txhash)
				msg += configs.HELP_register
				log.Printf("[pending] %v\n", msg)
			} else {
				log.Printf("[err] %v.\n", err)
			}
		}
		return err
	}
	log.Println("[ok] Registration success")
	return nil
}

func creatDataDir(confile configfile.Configfiler, c chain.Chainer) (string, string, string, error) {
	ctlAccount, err := c.GetCessAccount()
	if err != nil {
		return "", "", "", err
	}
	baseDir := filepath.Join(confile.GetDataDir(), ctlAccount, configs.BaseDir)
	log.Println(baseDir)
	_, err = os.Stat(baseDir)
	if err != nil {
		err = os.MkdirAll(baseDir, os.ModeDir)
		if err != nil {
			return "", "", "", err
		}
	}
	return filepath.Join(baseDir, "log"),
		filepath.Join(baseDir, "db"),
		filepath.Join(baseDir, "filler"),
		nil
}

// func register_if() bool {
// 	var reg bool
// 	sd, err := chain.GetSchedulerInfoOnChain()
// 	if err != nil {
// 		if err.Error() == chain.ERR_Empty {
// 			rgst()
// 			return true
// 		}
// 		log.Printf("\x1b[%dm[err]\x1b[0m Please try again later. [%v]\n", 41, err)
// 		os.Exit(1)
// 	}

// 	for _, v := range sd {
// 		if v.ControllerUser == types.NewAccountID(configs.PublicKey) {
// 			reg = true
// 		}
// 	}
// 	if !reg {
// 		rgst()
// 		return true
// 	}

// 	log.Printf("\x1b[%dm[ok]\x1b[0m Registered schedule\n", 42)

// 	addr, err := chain.GetAddressByPrk(configs.C.CtrlPrk)
// 	if err != nil {
// 		log.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
// 		os.Exit(1)
// 	}

// 	baseDir := filepath.Join(configs.C.DataDir, addr, configs.BaseDir)

// 	configs.LogFileDir = filepath.Join(baseDir, configs.LogFileDir)
// 	log.Printf(configs.LogFileDir)
// 	err = os.RemoveAll(configs.LogFileDir)
// 	if err != nil {
// 		log.Println(err)
// 	}
// 	if err = tools.CreatDirIfNotExist(configs.LogFileDir); err != nil {
// 		goto Err
// 	}
// 	//
// 	configs.FileCacheDir = filepath.Join(baseDir, configs.FileCacheDir)
// 	log.Printf(configs.FileCacheDir)
// 	err = os.RemoveAll(configs.FileCacheDir)
// 	if err != nil {
// 		log.Println(err)
// 	}
// 	if err = tools.CreatDirIfNotExist(configs.FileCacheDir); err != nil {
// 		goto Err
// 	}
// 	//
// 	configs.DbFileDir = filepath.Join(baseDir, configs.DbFileDir)
// 	log.Printf(configs.DbFileDir)
// 	err = os.RemoveAll(configs.DbFileDir)
// 	if err != nil {
// 		log.Println(err)
// 	}
// 	if err = tools.CreatDirIfNotExist(configs.DbFileDir); err != nil {
// 		goto Err
// 	}
// 	//
// 	configs.SpaceCacheDir = filepath.Join(baseDir, configs.SpaceCacheDir)
// 	log.Printf(configs.SpaceCacheDir)
// 	err = os.RemoveAll(configs.SpaceCacheDir)
// 	if err != nil {
// 		log.Println(err)
// 	}
// 	if err = tools.CreatDirIfNotExist(configs.SpaceCacheDir); err != nil {
// 		goto Err
// 	}

// 	return false
// Err:
// 	log.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
// 	os.Exit(1)
// 	return false
// }
