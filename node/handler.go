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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/pbc"
	"github.com/CESSProject/cess-scheduler/pkg/utils"
	cesskeyring "github.com/CESSProject/go-keyring"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

func NewServer(conn NetConn, dir string) Server {
	return &ConMgr{
		conn:       conn,
		dir:        dir,
		waitNotify: make(chan bool, 1),
	}
}

func NewClient(conn NetConn, dir string, files []string) Client {
	return &ConMgr{
		conn:       conn,
		dir:        dir,
		sendFiles:  files,
		waitNotify: make(chan bool, 1),
	}
}

func (c *ConMgr) Start(node *Node) {
	wg := &sync.WaitGroup{}
	c.conn.HandlerLoop(wg, true)
	err := c.handler(node)
	if err != nil {
		node.Logs.Common("error", err)
	}
	wg.Wait()
	node.Logs.Common("info", fmt.Errorf("Close a conn: %v", c.conn.GetRemoteAddr()))
}

func (c *ConMgr) SendFile(node *Node, fid string, filetype uint8, pkey, signmsg, sign []byte) error {
	wg := &sync.WaitGroup{}
	c.conn.HandlerLoop(wg, false)
	go func() {
		err := c.handler(node)
		if err != nil {
			node.Logs.Spc("err", err)
		}
	}()
	err := c.sendFile(node, fid, filetype, pkey, signmsg, sign)
	wg.Wait()
	return err
}

func (c *ConMgr) handler(node *Node) error {
	var (
		err          error
		fs           *os.File
		timeOutTimer *time.Ticker
	)

	defer func() {
		c.conn.Close()
		close(c.waitNotify)
		if fs != nil {
			fs.Close()
		}
		if timeOutTimer != nil {
			timeOutTimer.Stop()
		}
		if err := recover(); err != nil {
			node.Logs.Pnc("error", utils.RecoverError(err))
		}
	}()

	for !c.conn.IsClose() {
		if timeOutTimer != nil {
			select {
			case <-timeOutTimer.C:
				return errors.New("Get msg timeout")
			default:
			}
		}

		m, ok := c.conn.GetMsg()
		if !ok {
			return fmt.Errorf("Getmsg failed")
		}

		if m == nil {
			if timeOutTimer == nil {
				timeOutTimer = time.NewTicker(configs.TCP_Time_WaitMsg)
			}
			continue
		} else {
			if timeOutTimer != nil {
				timeOutTimer.Reset(configs.TCP_Time_WaitMsg)
			}
		}

		switch m.MsgType {
		case MsgHead:
			switch cap(m.Bytes) {
			case configs.TCP_ReadBuffer:
				readBufPool.Put(m.Bytes)
			default:
			}
			// Verify signature
			ok, err = VerifySign(m.Pubkey, m.SignMsg, m.Sign)
			if err != nil || !ok {
				c.conn.SendMsg(buildNotifyMsg("", Status_Err))
				return errors.New("Signature error")
			}
			//Judge whether the file has been uploaded
			fileState, err := GetFileState(node.Chain, m.FileHash)
			if err != nil {
				c.conn.SendMsg(buildNotifyMsg("", Status_Err))
				return errors.New("Get file state error")
			}

			// file state
			if fileState == chain.FILE_STATE_ACTIVE {
				c.conn.SendMsg(buildNotifyMsg("", Status_Err))
				return errors.New("File uploaded")
			}

			c.fileName = m.FileName

			// create file
			fs, err = os.OpenFile(filepath.Join(c.dir, m.FileName), os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
			if err != nil {
				c.conn.SendMsg(buildNotifyMsg("", Status_Err))
				return err
			}
			// notify suc
			c.conn.SendMsg(buildNotifyMsg(c.fileName, Status_Ok))

		case MsgFileSt:
			switch cap(m.Bytes) {
			case configs.TCP_ReadBuffer:
				readBufPool.Put(m.Bytes)
			default:
			}
			val, _ := node.Cache.Get([]byte(m.FileHash))

			c.conn.SendMsg(buildFileStMsg(m.FileHash, val))
			c.conn.SendMsg(buildNotifyMsg("", Status_Ok))
			time.Sleep(time.Second * 3)
			return nil
		case MsgFile:
			// If fs=nil, it means that the file has not been created.
			// You need to request MsgHead message first
			if fs == nil {
				c.conn.SendMsg(buildCloseMsg(Status_Err))
				return errors.New("File not open")
			}
			_, err = fs.Write(m.Bytes[:m.FileSize])
			if err != nil {
				c.conn.SendMsg(buildCloseMsg(Status_Err))
				return errors.New("File write failed")
			}
			switch cap(m.Bytes) {
			case configs.TCP_ReadBuffer:
				readBufPool.Put(m.Bytes)
			default:
			}

		case MsgEnd:
			switch cap(m.Bytes) {
			case configs.TCP_ReadBuffer:
				readBufPool.Put(m.Bytes)
			default:
			}
			info, err := fs.Stat()
			if err != nil {
				c.conn.SendMsg(buildNotifyMsg("", Status_Err))
				return errors.New("Invalid file")
			}

			if info.Size() != int64(m.FileSize) {
				c.conn.SendMsg(buildNotifyMsg("", Status_Err))
				return fmt.Errorf("file.size %v rece size %v \n", info.Size(), m.FileSize)
			}
			c.conn.SendMsg(buildNotifyMsg(c.fileName, Status_Ok))

			// close fs
			fs.Close()
			fs = nil

			// Last file Mark
			if m.LastMark {
				allChunksPath, _ := filepath.Glob(c.dir + "/" + m.FileHash + "*")
				filesize := binary.BigEndian.Uint64(m.SignMsg)
				// backup file to miner
				go node.FileBackupManagement(m.FileHash, int64(filesize), allChunksPath)
			}
		case MsgNotify:
			c.waitNotify <- m.Bytes[0] == byte(Status_Ok)
			switch cap(m.Bytes) {
			case configs.TCP_ReadBuffer:
				readBufPool.Put(m.Bytes)
			default:
			}

		case MsgClose:
			switch cap(m.Bytes) {
			case configs.TCP_ReadBuffer:
				readBufPool.Put(m.Bytes)
			default:
			}
			return errors.New("Close message")

		default:
			switch cap(m.Bytes) {
			case configs.TCP_ReadBuffer:
				readBufPool.Put(m.Bytes)
			default:
			}
			return errors.New("Invalid msgType")
		}
	}
	return err
}

func (c *ConMgr) sendFile(n *Node, fid string, filetype uint8, pkey, signmsg, sign []byte) error {
	defer func() {
		c.conn.Close()
	}()

	var (
		err          error
		lastmatrk    bool
		remoteAddr   string
		sharingtime  float64
		averagespeed float64
		tRecord      time.Time
	)

	remoteAddr = strings.Split(c.conn.GetRemoteAddr(), ":")[0]

	for i := 0; i < len(c.sendFiles); i++ {
		if (i + 1) == len(c.sendFiles) {
			lastmatrk = true
		}
		fileHash := utils.GetFileNameWithoutSuffix(c.sendFiles[i])
		if !strings.Contains(c.sendFiles[i], ".tag") {
			tRecord = time.Now()
			n.Logs.Speed(fmt.Errorf("Start transfer filler [%v] to [%v]", fileHash, remoteAddr))
		}
		err = c.sendSingleFile(c.sendFiles[i], fileHash, filetype, lastmatrk, pkey, signmsg, sign)
		if err != nil {
			if !strings.Contains(c.sendFiles[i], ".tag") {
				n.Logs.Speed(fmt.Errorf("Transfer Failed filler [%v] to [%v]", fileHash, remoteAddr))
			}
			break
		}
		if !strings.Contains(c.sendFiles[i], ".tag") {
			n.Logs.Speed(fmt.Errorf("Transfer completed filler [%v] to [%v]", fileHash, remoteAddr))
			sharingtime = time.Since(tRecord).Seconds()
			averagespeed = float64(configs.FillerSize) / sharingtime
			n.Logs.Speed(fmt.Errorf("[%v] Total time: %.2f seconds, average speed: %.2f bytes/s", fileHash, sharingtime, averagespeed))
		}
	}
	c.conn.SendMsg(buildCloseMsg(Status_Ok))
	time.Sleep(time.Second)
	return err
}

func (c *ConMgr) sendSingleFile(filePath string, fid string, filetype uint8, lastmark bool, pkey, signmsg, sign []byte) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}

	defer file.Close()

	fileInfo, _ := file.Stat()

	c.conn.SendMsg(buildHeadMsg(fileInfo.Name(), fid, filetype, lastmark, pkey, signmsg, sign))

	timerHead := time.NewTimer(configs.TCP_Time_WaitNotification)
	defer timerHead.Stop()
	select {
	case ok := <-c.waitNotify:
		if !ok {
			return fmt.Errorf("HeadMsg notification failed")
		}
	case <-timerHead.C:
		return fmt.Errorf("Timeout waiting for HeadMsg notification")
	}

	readBuf := sendBufPool.Get().([]byte)
	defer func() {
		sendBufPool.Put(readBuf)
	}()

	for !c.conn.IsClose() {
		n, err := file.Read(readBuf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}
		c.conn.SendMsg(buildFileMsg(fileInfo.Name(), filetype, n, readBuf[:n]))
	}

	c.conn.SendMsg(buildEndMsg(filetype, fileInfo.Name(), fid, uint64(fileInfo.Size()), lastmark))

	var t time.Duration
	waitTime := fileInfo.Size() / configs.TCP_Transmission_Slowest
	if time.Second*time.Duration(waitTime) < configs.TCP_Time_WaitNotification {
		t = configs.TCP_Time_WaitNotification
	} else {
		t = time.Second * time.Duration(waitTime)
	}
	timerFile := time.NewTimer(t)
	defer timerFile.Stop()
	select {
	case ok := <-c.waitNotify:
		if !ok {
			return fmt.Errorf("EndMsg notification failed")
		}
	case <-timerFile.C:
		return fmt.Errorf("Timeout waiting for EndMsg notification")
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
	var fileSt = FileStoreInfo{
		FileId:      fid,
		FileSize:    fsize,
		FileState:   chain.FILE_STATE_PENDING,
		IsUpload:    true,
		IsCheck:     true,
		IsShard:     true,
		IsScheduler: true,
	}
	n.Logs.Upfile("info", fmt.Errorf("[%v] Start the file backup management", fid))

	if len(chunks) <= 0 {
		fileSt.IsScheduler = false
		b, _ := json.Marshal(&fileSt)
		n.Cache.Put([]byte(fid), b)
		n.Logs.Upfile("err", fmt.Errorf("[%v] Not found", fid))
		return
	}

	b, _ := json.Marshal(&fileSt)
	n.Cache.Put([]byte(fid), b)

	var chunksInfo = make([]chain.BlockInfo, len(chunks))
	fileSt.Miners = make(map[int]string, len(chunks))
	for i := 0; i < len(chunks); {
		chunksInfo[i], err = n.backupFile(fid, chunks[i])
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		if chunksInfo[i].MinerId == types.U64(0) || chunksInfo[i].BlockSize == types.U64(0) {
			continue
		}
		fileSt.Miners[i] = string(chunksInfo[i].MinerAcc[:])
		b, _ := json.Marshal(&fileSt)
		n.Cache.Put([]byte(fid), b)
		n.Logs.Upfile("info", fmt.Errorf("[%v] backup suc", chunks[i]))
		i++
	}

	// Submit the file meta information to the chain
	var tryCount uint8
	for {
		txhash, err = n.Chain.SubmitFileMeta(fid, uint64(fsize), chunksInfo)
		if err != nil {
			tryCount++
			if tryCount > 3 {
				fileSt.FileState = chain.ERR_Failed
				b, _ = json.Marshal(&fileSt)
				n.Cache.Put([]byte(fid), b)
				return
			}
			time.Sleep(configs.BlockInterval)

			//Judge whether the file has been uploaded
			fileState, _ := GetFileState(n.Chain, fid)

			// file state
			if fileState == chain.FILE_STATE_ACTIVE {
				break
			}
			n.Logs.Upfile("err", fmt.Errorf("[%v] Submit filemeta fail: %v", fid, err))
			continue
		}
		break
	}
	fileSt.FileState = chain.FILE_STATE_ACTIVE
	b, _ = json.Marshal(&fileSt)
	n.Cache.Put([]byte(fid), b)
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
	blocksize, _ := CalcFileBlockSizeAndScanSize(fstat.Size())
	blocknum := fstat.Size() / blocksize

	fileTagPath := filepath.Join(n.TagDir, fname+".tag")
	_, err = os.Stat(fileTagPath)
	if err != nil {
		// calculate file tag info
		var commitResponse pbc.SigGenResponse

		matrix, num := pbc.SplitV2(fpath, configs.SIZE_1MiB)
		commitResponseCh := pbc.PbcKey.SigGen(matrix, num)

		select {
		case commitResponse = <-commitResponseCh:
		}

		if commitResponse.StatueMsg.StatusCode != pbc.Success {
			n.Logs.Upfile("error", fmt.Errorf("[%v] Failed to calculate the file tag", fname))
			return rtnValue, errors.New("failed")
		}

		var tag TagInfo
		tag.T = commitResponse.T
		tag.Phi = commitResponse.Phi
		tag.SigRootHash = commitResponse.SigRootHash

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

		if blackMiners.IsExist(minerinfo.Peerid) {
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
		dialer := net.Dialer{Timeout: configs.Tcp_Dial_Timeout}
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
		srv := NewClient(NewTcp(conTcp), "", []string{fpath, fileTagPath})
		err = srv.SendFile(n, fid, FileType_file, n.Chain.GetPublicKey(), []byte(msg), sign[:])
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
