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
	"runtime"
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
		err    error
		msg    string
		txhash string
		//sharingtime float64
		//averagespeed float64
		//tRecord   time.Time
		filler1     Filler
		filler2     Filler
		filler3     Filler
		filler4     Filler
		filler5     Filler
		minerinfo   chain.Cache_MinerInfo
		fillerMetas = make([]chain.FillerMetaInfo, 5)
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

		if filler1.Hash == "" {
			filler1 = <-C_Filler
		}
		if filler2.Hash == "" {
			filler2 = <-C_Filler
		}
		if filler3.Hash == "" {
			filler3 = <-C_Filler
		}
		if filler4.Hash == "" {
			filler4 = <-C_Filler
		}
		if filler5.Hash == "" {
			filler5 = <-C_Filler
		}

		//tRecord = time.Now()
		//n.Logs.Speed(fmt.Errorf("Start transfer filler [%v] to [C%v]", filler.Hash, minerinfo.Peerid))
		srv := NewClient(NewTcp(tcpConn), "",
			[]string{
				filler1.TagPath, filler1.FillerPath,
				filler2.TagPath, filler2.FillerPath,
				filler3.TagPath, filler3.FillerPath,
				filler4.TagPath, filler4.FillerPath,
				filler5.TagPath, filler5.FillerPath,
			})
		err = srv.SendFile(n, "", FileType_filler, n.Chain.GetPublicKey(), []byte(msg), sign[:])
		if err != nil {
			n.Logs.Spc("err", fmt.Errorf("[C%v] %v", minerinfo.Peerid, err))
			continue
		}

		//n.Logs.Speed(fmt.Errorf("Transfer completed filler [%v] to [C%v]", filler.Hash, minerinfo.Peerid))
		//sharingtime = time.Since(tRecord).Seconds()
		//averagespeed = float64(configs.FillerSize) / sharingtime
		//n.Logs.Speed(fmt.Errorf("[%v] Total time: %.2f seconds, average speed: %.2f bytes/s", filler.Hash, sharingtime, averagespeed))

		// C_FillerMeta <- combineFillerMeta(filler1.Hash, allMinerPubkey[i][:])
		// C_FillerMeta <- combineFillerMeta(filler2.Hash, allMinerPubkey[i][:])
		// C_FillerMeta <- combineFillerMeta(filler3.Hash, allMinerPubkey[i][:])
		// C_FillerMeta <- combineFillerMeta(filler4.Hash, allMinerPubkey[i][:])
		// C_FillerMeta <- combineFillerMeta(filler5.Hash, allMinerPubkey[i][:])

		fillerMetas[0] = combineFillerMeta(filler1.Hash, allMinerPubkey[i][:])
		fillerMetas[1] = combineFillerMeta(filler2.Hash, allMinerPubkey[i][:])
		fillerMetas[2] = combineFillerMeta(filler3.Hash, allMinerPubkey[i][:])
		fillerMetas[3] = combineFillerMeta(filler4.Hash, allMinerPubkey[i][:])
		fillerMetas[4] = combineFillerMeta(filler5.Hash, allMinerPubkey[i][:])

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

		filler1.Hash = ""
		filler2.Hash = ""
		filler3.Hash = ""
		filler4.Hash = ""
		filler5.Hash = ""

		os.Remove(filler1.FillerPath)
		os.Remove(filler1.TagPath)
		os.Remove(filler2.FillerPath)
		os.Remove(filler2.TagPath)
		os.Remove(filler3.FillerPath)
		os.Remove(filler3.TagPath)
		os.Remove(filler4.FillerPath)
		os.Remove(filler4.TagPath)
		os.Remove(filler5.FillerPath)
		os.Remove(filler5.TagPath)
		time.Sleep(time.Second)
		runtime.GC()
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
