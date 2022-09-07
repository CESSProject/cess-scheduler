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
	"log"
	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/CESSProject/cess-scheduler/internal/com"
	"github.com/CESSProject/cess-scheduler/internal/task"
	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/configfile"
	"github.com/CESSProject/cess-scheduler/pkg/db"
	"github.com/CESSProject/cess-scheduler/pkg/logger"
	"github.com/btcsuite/btcutil/base58"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/spf13/cobra"
)

// runCmd is used to start the service
//
// Usage:
//
//	scheduler run
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
		log.Println("[err]", err)
		os.Exit(1)
	}

	// judge the balance
	accountinfo, err := c.GetAccountInfo(c.GetPublicKey())
	if err != nil {
		log.Printf("[err] Failed to get account information\n")
		os.Exit(1)
	}

	if accountinfo.Data.Free.CmpAbs(
		new(big.Int).SetUint64(configs.MinimumBalance),
	) == -1 {
		log.Printf("[err] Account balance is less than %v pico\n", configs.MinimumBalance)
		os.Exit(1)
	}

	// whether to register
	schelist, err := c.GetAllSchedulerInfo()
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
	logDir, dbDir, fillerDir, fileDir, err := creatDataDir(confile, c)
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
	// sync block
	for {
		ok, err := c.GetSyncStatus()
		if err != nil {
			log.Println(err)
			os.Exit(1)
		}
		if !ok {
			break
		}
		log.Println("In sync block...")
		time.Sleep(time.Second * configs.BlockInterval)
	}
	log.Println("Sync complete")

	// run task
	go task.Run(confile, c, db, logs, fillerDir)
	com.Start(confile, c, db, logs, fillerDir, fileDir)
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

func creatDataDir(
	confile configfile.Configfiler,
	c chain.Chainer,
) (string, string, string, string, error) {
	ctlAccount, err := c.GetCessAccount()
	if err != nil {
		return "", "", "", "", err
	}
	baseDir := filepath.Join(confile.GetDataDir(), ctlAccount, configs.BaseDir)
	log.Println(baseDir)
	_, err = os.Stat(baseDir)
	if err != nil {
		err = os.MkdirAll(baseDir, os.ModeDir)
		if err != nil {
			return "", "", "", "", err
		}
	}

	logDir := filepath.Join(baseDir, "log")
	_, err = os.Stat(logDir)
	if err == nil {
		bkp := logDir + fmt.Sprintf("_%v", time.Now().Unix())
		os.Rename(logDir, bkp)
	}
	if err := os.MkdirAll(logDir, os.ModeDir); err != nil {
		return "", "", "", "", err
	}

	dbDir := filepath.Join(baseDir, "db")
	os.RemoveAll(dbDir)
	if err := os.MkdirAll(dbDir, os.ModeDir); err != nil {
		return "", "", "", "", err
	}

	fillerDir := filepath.Join(baseDir, "filler")
	os.RemoveAll(fillerDir)
	if err := os.MkdirAll(fillerDir, os.ModeDir); err != nil {
		return "", "", "", "", err
	}

	fileDir := filepath.Join(baseDir, "file")
	os.RemoveAll(fileDir)
	if err := os.MkdirAll(fillerDir, os.ModeDir); err != nil {
		return "", "", "", "", err
	}
	return logDir, dbDir, fillerDir, fileDir, nil
}
