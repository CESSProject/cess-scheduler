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

package node

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/utils"
	cesskeyring "github.com/CESSProject/go-keyring"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

// task_Space is used to fill the miner space
func (n *Node) task_Space(ch chan<- bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Logs.Pnc("error", utils.RecoverError(err))
		}
	}()

	var (
		err            error
		result         bool
		allMinerPubkey []types.AccountID
		chal           = make(chan bool)
		fillers        = make([]string, 0)
	)

	n.Logs.Spc("info", errors.New(">>> Start task_Space <<<"))

	for {
		// get all miner addresses
		allMinerPubkey, err = n.Chain.GetAllStorageMiner()
		if err != nil {
			n.Logs.Spc("err", fmt.Errorf("[GetAllStorageMiner] %v", err))
			time.Sleep(configs.BlockInterval)
			continue
		}

		// disrupt the order of miners
		utils.RandSlice(allMinerPubkey)

		for i := 0; i < len(allMinerPubkey); i++ {
			if len(fillers) == 0 {
				var filler = <-C_Filler
				fillers = append(fillers, filler.TagPath)
				fillers = append(fillers, filler.FillerPath)
				n.Logs.Spc("info", fmt.Errorf("Consumed a filler: %v", filepath.Base(filler.FillerPath)))
			}
			go storagefiller(chal, n, allMinerPubkey[i], fillers)
			result = <-chal
			if result {
				fillers = make([]string, 0)
			}
		}
		time.Sleep(configs.BlockInterval)
	}
}

func storagefiller(ch chan bool, n *Node, pkey types.AccountID, fillers []string) {
	defer func() {
		if err := recover(); err != nil {
			n.Logs.Pnc("error", utils.RecoverError(err))
		}
	}()

	var (
		err         error
		txhash      string
		fileHash    string
		minerinfo   chain.Cache_MinerInfo
		fillerMetas = make([]chain.FillerMetaInfo, 0)
	)

	// sign message
	msg := utils.GetRandomcode(16)
	kr, _ := cesskeyring.FromURI(n.Chain.GetMnemonicSeed(), cesskeyring.NetSubstrate{})
	sign, err := kr.Sign(kr.SigningContext([]byte(msg)))
	if err != nil {
		n.Logs.Spc("err", err)
		ch <- false
		return
	}

	if !n.Chain.GetChainStatus() {
		ch <- false
		return
	}

	minercache, err := n.Cache.Get(pkey[:])
	if err != nil {
		ch <- false
		return
	}

	err = json.Unmarshal(minercache, &minerinfo)
	if err != nil {
		n.Cache.Delete(pkey[:])
		ch <- false
		return
	}

	// if blackMiners.IsExist(minerinfo.Peerid) {
	// 	return
	// }

	workLock.Lock()
	defer func() {
		workLock.Unlock()
		n.Logs.Spc("info", errors.New("workLock.Unlock"))
	}()
	n.Logs.Spc("info", errors.New("workLock.Lock"))

	tcpConn, err := dialTcpServer(minerinfo.Ip)
	if err != nil {
		blackMiners.Add(minerinfo.Peerid)
		ch <- false
		return
	}

	cli := NewClient(NewTcp(tcpConn), "", nil)

	//for i := 0; i < configs.Num_Filler_Reserved; i++ {
	cli.SetFiles(fillers)
	n.Logs.Spc("info", fmt.Errorf("[SetFiles] %v", fillers))

	err = cli.SendFile(n, "", FileType_filler, n.Chain.GetPublicKey(), []byte(msg), sign[:])
	if err != nil {
		ch <- false
		blackMiners.Add(minerinfo.Peerid)
		n.Logs.Spc("err", fmt.Errorf("[SendFile] %v", err))
		return
	}

	n.Logs.Spc("info", fmt.Errorf("[SendFile] suc"))

	fileHash = strings.TrimSuffix(filepath.Base(fillers[0]), configs.TagFileExt)
	fillerMetas = append(fillerMetas, combineFillerMeta(fileHash, pkey[:]))
	n.Logs.Spc("info", fmt.Errorf("filler hash: %v", fileHash))
	//}

	// submit filler meta
	for len(fillerMetas) > 0 {
		txhash, err = n.Chain.SubmitFillerMeta(types.NewAccountID(pkey[:]), fillerMetas)
		if txhash == "" {
			n.Logs.Spc("err", fmt.Errorf("[SubmitFillerMeta] %v", err))
			time.Sleep(configs.BlockInterval)
			continue
		}
		for i := 0; i < len(fillerMetas); i++ {
			fpath := filepath.Join(n.FillerDir, fmt.Sprintf("%s", string(fillerMetas[i].Hash[:])))
			os.Remove(fpath)
			os.Remove(fpath + configs.TagFileExt)
		}
		n.Logs.Spc("info", fmt.Errorf("[SubmitFillerMeta] %v", txhash))
		break
	}
	ch <- true
}

func dialTcpServer(address string) (*net.TCPConn, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, err
	}
	dialer := net.Dialer{Timeout: configs.Tcp_Dial_Timeout}
	netCon, err := dialer.Dial("tcp", tcpAddr.String())
	if err != nil {
		return nil, err
	}
	conTcp, ok := netCon.(*net.TCPConn)
	if !ok {
		netCon.Close()
		return nil, errors.New("network conversion failed")
	}
	return conTcp, nil
}
