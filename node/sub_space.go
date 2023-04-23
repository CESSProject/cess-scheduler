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
	"strconv"
	"time"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/utils"
	cesskeyring "github.com/CESSProject/go-keyring"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

// task_Space is used to fill the miner space
func (n *Node) task_Space(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Logs.Pnc("error", utils.RecoverError(err))
		}
	}()

	concurrency := make(chan bool, configs.MAX_TCP_CONNECTION)
	defer close(concurrency)

	for i := uint8(0); i < configs.MAX_TCP_CONNECTION; i++ {
		concurrency <- true
	}

	for {
		for n.Chain.GetChainStatus() {
			select {
			case <-concurrency:
				runtime.GC()
				time.Sleep(time.Second)
				runtime.GC()
				go storagefiller(concurrency, n)
			default:
				time.Sleep(time.Second)
			}
		}
		time.Sleep(configs.BlockInterval)
	}
}

func storagefiller(ch chan bool, n *Node) {
	var (
		err         error
		msg         string
		txhash      string
		count       uint8
		minerinfo   chain.Cache_MinerInfo
		sendFillers = make([]string, configs.Num_Filler_Reserved*2)
		fillerMetas = make([]chain.FillerMetaInfo, configs.Num_Filler_Reserved)
	)

	defer func() {
		ch <- true
		if err := recover(); err != nil {
			n.Logs.Pnc("error", utils.RecoverError(err))
		}
	}()
	// get all miner addresses
	allMinerPubkey, err := n.Chain.GetAllStorageMiner()
	if err != nil {
		n.Logs.Spc("err", err)
		return
	}

	// disrupt the order of miners
	utils.RandSlice(allMinerPubkey)

	// sign message
	msg = utils.GetRandomcode(16)
	kr, _ := cesskeyring.FromURI(n.Chain.GetMnemonicSeed(), cesskeyring.NetSubstrate{})
	sign, err := kr.Sign(kr.SigningContext([]byte(msg)))
	if err != nil {
		n.Logs.Spc("err", err)
		return
	}
	count = 0
	// iterate over all minerss
	for i := 0; i < len(allMinerPubkey); i++ {
		time.Sleep(time.Second)
		if !n.Chain.GetChainStatus() {
			return
		}

		minercache, err := n.Cache.Get(allMinerPubkey[i][:])
		if err != nil {
			n.Logs.Spc("err", err)
			continue
		}

		err = json.Unmarshal(minercache, &minerinfo)
		if err != nil {
			n.Cache.Delete(allMinerPubkey[i][:])
			n.Logs.Spc("err", err)
			continue
		}

		b, err := n.Cache.Get([]byte(fmt.Sprintf("%s%d", BlackPrefix, minerinfo.Peerid)))
		if err == nil {
			t, err := strconv.ParseInt(string(b), 10, 64)
			if err != nil {
				n.Cache.Delete([]byte(fmt.Sprintf("%s%d", BlackPrefix, minerinfo.Peerid)))
			} else {
				if time.Since(time.Unix(t, 0)).Hours() < 5 {
					continue
				}
				n.Cache.Delete([]byte(fmt.Sprintf("%s%d", BlackPrefix, minerinfo.Peerid)))
			}
		}

		tcpConn, err := dialTcpServer(minerinfo.Ip)
		if err != nil {
			n.Cache.Put([]byte(fmt.Sprintf("%s%d", BlackPrefix, minerinfo.Peerid)), []byte(fmt.Sprintf("%d", time.Now().Unix())))
			n.Logs.Spc("err", err)
			continue
		}

		for j := 0; j < (configs.Num_Filler_Reserved * 2); j += 2 {
			if sendFillers[j] == "" {
				var filler = <-C_Filler
				sendFillers[j] = filler.TagPath
				sendFillers[j+1] = filler.FillerPath
			}
		}

		err = NewClient(NewTcp(tcpConn), "", sendFillers).SendFile(n, "", FileType_filler, n.Chain.GetPublicKey(), []byte(msg), sign[:])
		if err != nil {
			n.Logs.Spc("err", fmt.Errorf("[C%v] %v", minerinfo.Peerid, err))
			continue
		}

		for j := 1; j < (configs.Num_Filler_Reserved * 2); j += 2 {
			var fileHas = filepath.Base(sendFillers[j])
			fillerMetas[(j-1)/2] = combineFillerMeta(fileHas, allMinerPubkey[i][:])
		}

		// submit filler meta
		txhash = ""
		for {
			txhash, err = n.Chain.SubmitFillerMeta(types.NewAccountID(allMinerPubkey[i][:]), fillerMetas)
			if txhash == "" {
				n.Logs.FillerMeta("error", err)
				time.Sleep(configs.BlockInterval)
				continue
			}
			n.Logs.FillerMeta("info", fmt.Errorf("[C%v] %v", minerinfo.Peerid, txhash))
			break
		}

		for j := 0; j < (configs.Num_Filler_Reserved * 2); j++ {
			os.Remove(sendFillers[j])
		}

		for j := 0; j < (configs.Num_Filler_Reserved * 2); j += 2 {
			sendFillers[j] = ""
			sendFillers[j+1] = ""
		}
		count++
		if count > 10 {
			break
		}
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
