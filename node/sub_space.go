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
	"time"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/utils"
	cesskeyring "github.com/CESSProject/go-keyring"
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
				go n.storagefiller(concurrency)
			default:
				time.Sleep(time.Second)
			}
		}
		time.Sleep(configs.BlockInterval)
	}
}

func (n *Node) storagefiller(ch chan bool) {
	var (
		err          error
		msg          string
		sharingtime  float64
		averagespeed float64
		tRecord      time.Time
		filler       Filler
		minerinfo    chain.Cache_MinerInfo
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

	// iterate over all minerss
	for i := 0; i < len(allMinerPubkey); i++ {
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
		tcpConn, err := dialTcpServer(minerinfo.Ip)
		if err != nil {
			n.Logs.Spc("err", err)
			continue
		}

		if filler.Hash == "" {
			filler = <-C_Filler
		}

		tRecord = time.Now()
		n.Logs.Speed(fmt.Errorf("Start transfer filler [%v] to [C%v]", filler.Hash, minerinfo.Peerid))
		srv := NewClient(NewTcp(tcpConn), "", []string{filler.TagPath, filler.FillerPath})
		err = srv.SendFile(n, filler.Hash, FileType_filler, n.Chain.GetPublicKey(), []byte(msg), sign[:])
		if err != nil {
			n.Logs.Spc("err", fmt.Errorf("[C%v] %v", minerinfo.Peerid, err))
			continue
		}
		n.Logs.Speed(fmt.Errorf("Transfer completed filler [%v] to [C%v]", filler.Hash, minerinfo.Peerid))
		sharingtime = time.Since(tRecord).Seconds()
		averagespeed = float64(configs.FillerSize) / sharingtime
		n.Logs.Speed(fmt.Errorf("[%v] Total time: %.2f seconds, average speed: %.2f bytes/s", filler.Hash, sharingtime, averagespeed))
		fillerMetaEle := combineFillerMeta(filler.Hash, allMinerPubkey[i][:])
		C_FillerMeta <- fillerMetaEle
		filler.Hash = ""
		os.Remove(filler.FillerPath)
		os.Remove(filler.TagPath)
		time.Sleep(time.Second)
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
