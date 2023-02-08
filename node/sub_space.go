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
	"runtime"
	"sync"
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
		allMinerPubkey []types.AccountID
		wg             = new(sync.WaitGroup)
	)

	for {
		// get all miner addresses
		allMinerPubkey, err = n.Chain.GetAllStorageMiner()
		if err != nil {
			n.Logs.Spc("err", err)
			time.Sleep(time.Second * configs.BlockInterval)
			continue
		}

		// disrupt the order of miners
		utils.RandSlice(allMinerPubkey)

		for i := 0; i < len(allMinerPubkey); i++ {
			wg.Add(1)
			go storagefiller(wg, n, allMinerPubkey[i])
			wg.Wait()
		}
		runtime.GC()
		time.Sleep(configs.BlockInterval)
	}
}

func storagefiller(wg *sync.WaitGroup, n *Node, pkey types.AccountID) {
	defer func() {
		wg.Done()
		if err := recover(); err != nil {
			n.Logs.Pnc("error", utils.RecoverError(err))
		}
	}()

	var (
		err         error
		txhash      string
		fileHash    string
		minerinfo   chain.Cache_MinerInfo
		sendFillers = make([]string, 2)
		fillerMetas = make([]chain.FillerMetaInfo, 0)
	)

	// sign message
	msg := utils.GetRandomcode(16)
	kr, _ := cesskeyring.FromURI(n.Chain.GetMnemonicSeed(), cesskeyring.NetSubstrate{})
	sign, err := kr.Sign(kr.SigningContext([]byte(msg)))
	if err != nil {
		n.Logs.Spc("err", err)
		return
	}

	if !n.Chain.GetChainStatus() {
		return
	}

	minercache, err := n.Cache.Get(pkey[:])
	if err != nil {
		return
	}

	err = json.Unmarshal(minercache, &minerinfo)
	if err != nil {
		n.Cache.Delete(pkey[:])
		return
	}

	if blackMiners.IsExist(minerinfo.Peerid) {
		return
	}

	tcpConn, err := dialTcpServer(minerinfo.Ip)
	if err != nil {
		blackMiners.Add(minerinfo.Peerid)
		return
	}

	for i := 0; i < configs.Num_Filler_Reserved; i++ {
		var filler = <-C_Filler
		sendFillers[0] = filler.TagPath
		sendFillers[1] = filler.FillerPath
		err = NewClient(NewTcp(tcpConn), "", sendFillers).SendFile(n, "", FileType_filler, n.Chain.GetPublicKey(), []byte(msg), sign[:])
		if err != nil {
			blackMiners.Add(minerinfo.Peerid)
			n.Logs.Spc("err", fmt.Errorf("[C%v] %v", minerinfo.Peerid, err))
			break
		}
		fileHash = filepath.Base(sendFillers[0])
		fillerMetas = append(fillerMetas, combineFillerMeta(fileHash, pkey[:]))
	}

	// submit filler meta
	for len(fillerMetas) > 0 {
		txhash, err = n.Chain.SubmitFillerMeta(types.NewAccountID(pkey[:]), fillerMetas)
		if txhash == "" {
			n.Logs.FillerMeta("err", err)
			time.Sleep(configs.BlockInterval)
			continue
		}
		n.Logs.FillerMeta("info", fmt.Errorf("[C%v] %v", minerinfo.Peerid, txhash))
		break
	}

	for i := 0; i < len(fillerMetas); i++ {
		fpath := filepath.Join(n.FillerDir, fmt.Sprintf("%s", string(fillerMetas[i].Hash[:])))
		os.Remove(fpath)
		os.Remove(fpath + configs.TagFileExt)
	}
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
		return nil, errors.New("network conversion failed")
	}
	return conTcp, nil
}
