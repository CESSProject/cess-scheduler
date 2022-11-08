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
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/pbc"
	"github.com/CESSProject/cess-scheduler/pkg/utils"
	cesskeyring "github.com/CESSProject/go-keyring"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

func (n *Node) NewServer(conn NetConn) Server {
	node := New()
	node.Conn = &ConMgr{
		conn:       conn,
		dir:        n.FileDir,
		waitNotify: make(chan bool, 1),
		stop:       make(chan struct{}),
	}
	node.Cache = n.Cache
	node.Chain = n.Chain
	node.Confile = n.Confile
	node.Logs = n.Logs
	node.lock = n.lock
	node.conns = n.conns
	node.FileDir = n.FileDir
	node.FillerDir = n.FillerDir
	node.TagDir = n.TagDir
	return node
}

func (n *Node) NewClient(conn NetConn, dir string, files []string) Client {
	node := New()
	node.Conn = &ConMgr{
		conn:       conn,
		dir:        dir,
		sendFiles:  files,
		waitNotify: make(chan bool, 1),
		stop:       make(chan struct{}),
	}
	node.Cache = n.Cache
	node.Chain = n.Chain
	node.Confile = n.Confile
	node.Logs = n.Logs
	node.lock = n.lock
	node.conns = n.conns
	node.FileDir = n.FileDir
	node.FillerDir = n.FillerDir
	node.TagDir = n.TagDir
	return node
}

func (n *Node) Start() {
	defer func() {
		n.Logs.Common("info", fmt.Errorf("Close a conn: %v", n.Conn.conn.GetRemoteAddr()))
		n = nil
	}()
	n.Conn.conn.HandlerLoop()
	err := n.handler()
	if err != nil {
		n.Logs.Common("error", err)
	}
	time.Sleep(time.Second)
	runtime.Goexit()
}

func (n *Node) handler() error {
	var (
		err        error
		fs         *os.File
		fillerFs   *os.File
		tStart     time.Time
		tTimer     *time.Ticker
		filler     Filler
		minerAcc   []byte
		remoteAddr string
	)
	n.AddConns()
	//log.Printf("Recv a conn:%v", n.Conn.conn.GetRemoteAddr())
	defer func() {
		err := recover()
		if err != nil {
			n.Logs.Pnc("error", utils.RecoverError(err))
		}
		n.ClearConns()
		n.Conn.conn.Close()
		close(n.Conn.stop)
		close(n.Conn.waitNotify)
		if fs != nil {
			fs.Close()
		}
		if fillerFs != nil {
			fillerFs.Close()
		}
		if tTimer != nil {
			tTimer.Stop()
		}
	}()

	for !n.Conn.conn.IsClose() {
		if tTimer != nil {
			select {
			case <-tTimer.C:
				runtime.Goexit()
			default:
			}
		}
		m, ok := n.Conn.conn.GetMsg()
		if !ok {
			runtime.Goexit()
		}
		if m == nil {
			continue
		}

		switch m.MsgType {
		case MsgHead:
			msg := msgPool.Get().(*Message)
			defer msgPool.Put(msg)
			// Verify signature
			ok, err = VerifySign(m.Pubkey, m.SignMsg, m.Sign)
			if err != nil || !ok {
				msg.buildNotifyMsg("", Status_Err)
				n.Conn.conn.SendMsg(msg)
				//n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
				runtime.Goexit()
			}
			//Judge whether the file has been uploaded
			fileState, err := GetFileState(n.Chain, m.FileHash)
			if err != nil {
				msg.buildNotifyMsg("", Status_Err)
				n.Conn.conn.SendMsg(msg)
				runtime.Goexit()
			}

			// file state
			if fileState == chain.FILE_STATE_ACTIVE {
				msg.buildNotifyMsg("", Status_Err)
				n.Conn.conn.SendMsg(msg)
				//n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
				runtime.Goexit()
			}

			//TODO: Judge whether the space is enough

			n.Conn.fileName = m.FileName

			// create file
			fs, err = os.OpenFile(filepath.Join(n.Conn.dir, m.FileName), os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
			if err != nil {
				msg.buildNotifyMsg("", Status_Err)
				n.Conn.conn.SendMsg(msg)
				//n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
				runtime.Goexit()
			}
			// Notify the other party
			msg.buildNotifyMsg(n.Conn.fileName, Status_Ok)
			n.Conn.conn.SendMsg(msg)
			//n.Conn.conn.SendMsg(NewNotifyMsg(n.Conn.fileName, Status_Ok))

		case MsgFillerHead:
			// Verify signature
			ok, err := VerifySign(m.Pubkey, m.SignMsg, m.Sign)
			if err != nil || !ok {
				msg := msgPool.Get().(*Message)
				defer msgPool.Put(msg)
				msg.buildNotifyMsg("", Status_Err)
				n.Conn.conn.SendMsg(msg)
				runtime.Goexit()
			}

			minerAcc = m.Pubkey

			// select a filler
			timer := time.NewTimer(time.Second * 3)
			select {
			case filler = <-C_Filler:
				n.Logs.GenFiller("info", fmt.Errorf("Consumed a filler: %v", filler.Hash))
			case <-timer.C:
				msg := msgPool.Get().(*Message)
				defer msgPool.Put(msg)
				msg.buildNotifyMsg(n.Conn.fileName, Status_Ok)
				n.Conn.conn.SendMsg(msg)
				//n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
			}
			// Notify the other party
			n.Conn.conn.SendMsg(NewNotifyFillerMsg(filler.Hash, Status_Ok))
			tTimer = time.NewTicker(configs.TCP_ShortMessage_WaitingTime)
		case MsgFiller:
			tTimer = time.NewTicker(configs.TCP_FillerMessage_WaitingTime)
			readBuf := bytesPool.Get().(*[]byte)
			defer bytesPool.Put(readBuf)
			msg := msgPool.Get().(*Message)
			defer msgPool.Put(msg)

			remoteAddr = n.Conn.conn.GetRemoteAddr()
			if !strings.Contains(m.FileName, "tag") {
				tStart = time.Now()
				n.Logs.Speed(fmt.Errorf("Start transfer filler [%v] to [%v]", m.FileName, remoteAddr))
			}
			// open filler
			fillerFs, err = os.OpenFile(filepath.Join(n.FillerDir, m.FileName), os.O_RDONLY, os.ModePerm)
			if err != nil {
				msg.buildNotifyMsg(n.Conn.fileName, Status_Ok)
				n.Conn.conn.SendMsg(msg)
				runtime.Goexit()
			}

			fillerInfo, _ := fillerFs.Stat()

			// send filler
			for !n.Conn.conn.IsClose() {
				num, err := fillerFs.Read(*readBuf)
				if err != nil && err != io.EOF {
					// notify the other party
					bytesPool.Put(readBuf)
					msg.buildNotifyMsg(n.Conn.fileName, Status_Ok)
					n.Conn.conn.SendMsg(msg)
					//n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
					runtime.Goexit()
				}
				if num == 0 {
					break
				}
				msg.buildFillerMsg(m.FileName, (*readBuf)[:num])
				n.Conn.conn.SendMsg(msg)
				time.Sleep(configs.TCP_Connection_Interval)
			}

			// send end msg
			msg.buildFillerEndMsg(m.FileName, uint64(fillerInfo.Size()))
			n.Conn.conn.SendMsg(msg)
			time.Sleep(configs.TCP_Connection_Interval)

			// notify the other party
			msg.buildNotifyMsg(m.FileName, Status_Ok)
			n.Conn.conn.SendMsg(msg)
			//n.Conn.conn.SendMsg(NewNotifyMsg(m.FileName, Status_Ok))

		case MsgFile:
			msg := msgPool.Get().(*Message)
			defer msgPool.Put(msg)
			// If fs=nil, it means that the file has not been created.
			// You need to request MsgHead message first
			if fs == nil {
				msg.buildCloseMsg("", Status_Err)
				n.Conn.conn.SendMsg(msg)
				runtime.Goexit()
			}
			_, err = fs.Write(m.Bytes)
			if err != nil {
				msg.buildCloseMsg("", Status_Err)
				n.Conn.conn.SendMsg(msg)
				runtime.Goexit()
			}

		case MsgEnd:
			msg := msgPool.Get().(*Message)
			defer msgPool.Put(msg)
			info, _ := fs.Stat()
			if info.Size() != int64(m.FileSize) {
				err = fmt.Errorf("file.size %v rece size %v \n", info.Size(), m.FileSize)
				msg.buildCloseMsg(n.Conn.fileName, Status_Err)
				n.Conn.conn.SendMsg(msg)
				runtime.Goexit()
			}
			msg.buildNotifyMsg(n.Conn.fileName, Status_Ok)
			n.Conn.conn.SendMsg(msg)
			//n.Conn.conn.SendMsg(NewNotifyMsg(n.Conn.fileName, Status_Ok))

			// close fs
			_ = fs.Close()
			fs = nil

			// Last file Mark
			if m.LastMark {
				allChunksPath, _ := filepath.Glob(n.Conn.dir + "/" + m.FileHash + "*")
				filesize := binary.BigEndian.Uint64(m.SignMsg)
				// backup file to miner
				go n.FileBackupManagement(m.FileHash, int64(filesize), allChunksPath)
			}
		case MsgNotify:
			n.Conn.waitNotify <- m.Bytes[0] == byte(Status_Ok)
			if !(m.Bytes[0] == byte(Status_Ok)) {
				runtime.Goexit()
			}

			if len(m.Bytes) > 1 {
				// If the transmission is completed, the information will be recorded
				// Uplink and delete local cache later
				if filler.Hash == string(m.Bytes[1:]) {
					//log.Printf("Transfer completed filler [%v] to [%v]", filler.Hash, remoteAddr)
					n.Logs.Speed(fmt.Errorf("Transfer completed filler [%v] to [%v]", filler.Hash, remoteAddr))
					sharingtime := time.Since(tStart).Seconds()
					averagespeed := float64(configs.FillerSize) / sharingtime
					//log.Printf("Sharing time: %.2f seconds, average speed: %.2f byte/s", sharingtime, averagespeed)
					n.Logs.Speed(fmt.Errorf("[%v] Total time: %.2f seconds, average speed: %.2f bytes/s", filler.Hash, sharingtime, averagespeed))
					fillerMetaEle := combineFillerMeta(filler.Hash, minerAcc)
					C_FillerMeta <- fillerMetaEle
					os.Remove(filler.FillerPath)
					os.Remove(filler.TagPath)
					runtime.Goexit()
				}
			}
		case MsgClose:
			runtime.Goexit()
		}
	}
	return err
}

func (n *Node) SendFile(fid string, pkey, signmsg, sign []byte) error {
	var err error
	n.Conn.conn.HandlerLoop()
	go func() {
		_ = n.handler()
	}()
	err = n.Conn.sendFile(fid, pkey, signmsg, sign)
	return err
}

func (c *ConMgr) sendFile(fid string, pkey, signmsg, sign []byte) error {
	defer func() {
		_ = c.conn.Close()
	}()

	var err error
	var lastmatrk bool
	for i := 0; i < len(c.sendFiles); i++ {
		if (i + 1) == len(c.sendFiles) {
			lastmatrk = true
		}
		err = c.sendSingleFile(c.sendFiles[i], fid, lastmatrk, pkey, signmsg, sign)
		if err != nil {
			return err
		}
		if lastmatrk {
			for _, v := range c.sendFiles {
				os.Remove(v)
			}
		}
	}
	msg := msgPool.Get().(*Message)
	defer msgPool.Put(msg)
	msg.buildCloseMsg(c.fileName, Status_Ok)
	c.conn.SendMsg(msg)
	timer := time.NewTimer(configs.TCP_ShortMessage_WaitingTime)
	select {
	case <-timer.C:
	}
	return nil
}

func (c *ConMgr) sendSingleFile(filePath string, fid string, lastmark bool, pkey, signmsg, sign []byte) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}

	defer func() {
		if file != nil {
			_ = file.Close()
		}
	}()

	fileInfo, _ := file.Stat()

	m := NewHeadMsg(fileInfo.Name(), fid, lastmark, pkey, signmsg, sign)
	c.conn.SendMsg(m)

	timer := time.NewTimer(configs.TCP_ShortMessage_WaitingTime)
	select {
	case ok := <-c.waitNotify:
		if !ok {
			return fmt.Errorf("send err")
		}
	case <-timer.C:
		return fmt.Errorf("wait server msg timeout")
	}

	readBuf := bytesPool.Get().(*[]byte)
	defer bytesPool.Put(readBuf)
	for !c.conn.IsClose() {

		n, err := file.Read(*readBuf)
		if err != nil && err != io.EOF {
			return err
		}

		if n == 0 {
			break
		}
		c.conn.SendMsg(NewFileMsg(c.fileName, (*readBuf)[:n]))
	}

	c.conn.SendMsg(NewEndMsg(c.fileName, uint64(fileInfo.Size()), lastmark))

	waitTime := fileInfo.Size() / configs.TCP_Transmission_Slowest
	if time.Second*time.Duration(waitTime) < configs.TCP_ShortMessage_WaitingTime {
		timer = time.NewTimer(configs.TCP_ShortMessage_WaitingTime)
	} else {
		timer = time.NewTimer(time.Second * time.Duration(waitTime))
	}

	select {
	case ok := <-c.waitNotify:
		if !ok {
			return fmt.Errorf("send err")
		}
	case <-timer.C:
		return fmt.Errorf("wait server msg timeout")
	}

	return nil
}

// file backup management
func (n *Node) FileBackupManagement(fid string, fsize int64, chunks []string) {
	defer func() {
		if err := recover(); err != nil {
			n.Logs.Pnc("error", utils.RecoverError(err))
		}
	}()
	var (
		err    error
		txhash string
	)

	n.Logs.Upfile("info", fmt.Errorf("[%v] Start the file backup management", fid))

	var chunksInfo = make([]chain.BlockInfo, len(chunks))

	for i := 0; i < len(chunks); {
		chunksInfo[i], err = n.backupFile(fid, chunks[i])
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		n.Logs.Upfile("info", fmt.Errorf("[%v] backup suc", chunks[i]))
		i++
	}

	// Submit the file meta information to the chain
	for {
		txhash, err = n.Chain.SubmitFileMeta(fid, uint64(fsize), chunksInfo)
		if txhash == "" {
			n.Logs.Upfile("error", fmt.Errorf("[%v] Submit filemeta fail: %v", fid, err))
			time.Sleep(configs.BlockInterval)
			continue
		}
		break
	}
	n.Logs.Upfile("info", fmt.Errorf("[%v] Submit filemeta [%v]", fid, txhash))
	return
}

// processingfile is used to process all copies of the file and the corresponding tag information
func (n *Node) backupFile(fid, fpath string) (chain.BlockInfo, error) {
	var (
		err            error
		msg            string
		rtnValue       chain.BlockInfo
		minerinfo      chain.Cache_MinerInfo
		allMinerPubkey []types.AccountID
	)
	defer func() {
		if err := recover(); err != nil {
			n.Logs.Log("panic", "error", utils.RecoverError(err))
		}
	}()
	fname := filepath.Base(fpath)
	n.Logs.Upfile("info", fmt.Errorf("[%v] Prepare to store to miner", fname))

	fstat, err := os.Stat(fpath)
	if err != nil {
		n.Logs.Upfile("error", fmt.Errorf("[%v] %v", fname, err))
		return rtnValue, err
	}
	blocksize, scansize := CalcFileBlockSizeAndScanSize(fstat.Size())
	blocknum := fstat.Size() / blocksize

	fileTagPath := filepath.Join(n.TagDir, fname+".tag")
	_, err = os.Stat(fileTagPath)
	if err != nil {
		// calculate file tag info
		var PoDR2commit pbc.PoDR2Commit
		var commitResponse pbc.PoDR2CommitResponse
		PoDR2commit.FilePath = fpath
		PoDR2commit.BlockSize = blocksize
		commitResponseCh, err := PoDR2commit.PoDR2ProofCommit(pbc.Key_Ssk, string(pbc.Key_SharedParams), scansize)
		if err != nil {
			n.Logs.Upfile("error", fmt.Errorf("[%v] PoDR2ProofCommit: %v", fname, err))
			return rtnValue, err
		}
		select {
		case commitResponse = <-commitResponseCh:
		}
		if commitResponse.StatueMsg.StatusCode != pbc.Success {
			n.Logs.Upfile("error", fmt.Errorf("[%v] Failed to calculate the file tag", fname))
			return rtnValue, errors.New("failed")
		}
		var tag TagInfo
		tag.T.Name = commitResponse.T.Name
		tag.T.N = commitResponse.T.N
		tag.T.U = commitResponse.T.U
		tag.T.Signature = commitResponse.T.Signature
		tag.Sigmas = commitResponse.Sigmas
		tag_bytes, err := json.Marshal(&tag)
		if err != nil {
			n.Logs.Upfile("error", fmt.Errorf("[%v] %v", fname, err))
			return rtnValue, err
		}

		//Save tag information to file
		ftag, err := os.OpenFile(fileTagPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
		if err != nil {
			n.Logs.Upfile("error", fmt.Errorf("[%v] %v", fname, err))
			return rtnValue, err
		}
		ftag.Write(tag_bytes)

		//flush to disk
		err = ftag.Sync()
		if err != nil {
			n.Logs.Upfile("error", fmt.Errorf("[%v] %v", fname, err))
			ftag.Close()
			os.Remove(fileTagPath)
			return rtnValue, err
		}
		ftag.Close()
	}

	// Get the publickey of all miners in the chain
	for {
		allMinerPubkey, err = n.Chain.GetAllStorageMiner()
		if err != nil {
			time.Sleep(configs.BlockInterval)
			continue
		}
		break
	}

	// Disrupt the order of miners
	utils.RandSlice(allMinerPubkey)

	msg = utils.GetRandomcode(16)
	kr, _ := cesskeyring.FromURI(n.Chain.GetMnemonicSeed(), cesskeyring.NetSubstrate{})
	// sign message
	sign, err := kr.Sign(kr.SigningContext([]byte(msg)))
	if err != nil {
		n.Logs.Upfile("error", fmt.Errorf("[%v] %v", fname, err))
		return rtnValue, err
	}

	for i := 0; i < len(allMinerPubkey); i++ {
		minercache, err := n.Cache.Get(allMinerPubkey[i][:])
		if err != nil {
			n.Logs.Upfile("error", fmt.Errorf("[%v] %v", fname, err))
			continue
		}

		err = json.Unmarshal(minercache, &minerinfo)
		if err != nil {
			n.Cache.Delete(allMinerPubkey[i][:])
			n.Logs.Upfile("error", fmt.Errorf("[%v] %v", fname, err))
			continue
		}

		if minerinfo.Free < uint64(fstat.Size()) {
			continue
		}

		tcpAddr, err := net.ResolveTCPAddr("tcp", minerinfo.Ip)
		if err != nil {
			n.Logs.Upfile("error", fmt.Errorf("[%v] %v", fname, err))
			continue
		}
		dialer := net.Dialer{Timeout: configs.TCP_ShortMessage_WaitingTime}
		netCon, err := dialer.Dial("tcp", tcpAddr.String())
		if err != nil {
			n.Logs.Upfile("error", fmt.Errorf("[%v] %v", fname, err))
			continue
		}
		conTcp, ok := netCon.(*net.TCPConn)
		if !ok {
			n.Logs.Upfile("error", fmt.Errorf("[%v] %v", fname, tcpAddr.String()))
			continue
		}
		srv := n.NewClient(NewTcp(conTcp), "", []string{fpath, fileTagPath})
		err = srv.SendFile(fid, n.Chain.GetPublicKey(), []byte(msg), sign[:])
		if err != nil {
			n.Logs.Upfile("error", fmt.Errorf("[%v] %v", fname, err))
			continue
		}
		var blockId chain.FileBlockId
		if len(fname) < len(blockId) {
			fname += ".000"
		}
		if len(fname) != len(blockId) {
			continue
		}
		for i := 0; i < len(blockId); i++ {
			blockId[i] = types.U8(fname[i])
		}
		rtnValue.BlockId = blockId

		var ipType chain.Ipv4Type
		ipType.Index = types.U8(0)
		ip_port := strings.Split(minerinfo.Ip, ":")
		port, _ := strconv.Atoi(ip_port[1])
		ipType.Port = types.U16(port)
		if utils.IsIPv4(ip_port[0]) {
			ips := strings.Split(ip_port[0], ".")
			for i := 0; i < len(ipType.Value); i++ {
				temp, _ := strconv.Atoi(ips[i])
				ipType.Value[i] = types.U8(temp)
			}
		}
		rtnValue.BlockSize = types.U64(fstat.Size())
		rtnValue.MinerAcc = allMinerPubkey[i]
		rtnValue.MinerIp = ipType
		rtnValue.MinerId = types.U64(minerinfo.Peerid)
		rtnValue.BlockNum = types.U32(blocknum)
		break
	}
	if rtnValue.MinerId == 0 {
		return rtnValue, errors.New("failed")
	}
	return rtnValue, err
}

func CalcFileBlockSizeAndScanSize(fsize int64) (int64, int64) {
	var (
		blockSize     int64
		scanBlockSize int64
	)
	if fsize < configs.SIZE_1KiB {
		return fsize, fsize
	}
	if fsize > math.MaxUint32 {
		blockSize = math.MaxUint32
		scanBlockSize = blockSize / 8
		return blockSize, scanBlockSize
	}
	blockSize = fsize / 16
	scanBlockSize = blockSize / 8
	return blockSize, scanBlockSize
}

func combineFillerMeta(fileHash string, pubkey []byte) chain.FillerMetaInfo {
	var metainfo chain.FillerMetaInfo
	var hash [64]types.U8
	for i := 0; i < 64; i++ {
		hash[i] = types.U8(fileHash[i])
	}
	metainfo.Hash = hash
	metainfo.Size = configs.FillerSize
	metainfo.Acc = types.NewAccountID(pubkey)

	blocknum := uint64(math.Ceil(float64(configs.FillerSize / configs.BlockSize)))
	if blocknum == 0 {
		blocknum = 1
	}
	metainfo.BlockNum = types.U32(blocknum)
	metainfo.BlockSize = types.U32(uint32(configs.BlockSize))
	return metainfo
}
