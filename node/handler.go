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

type FileStoreInfo struct {
	FileId      string         `json:"file_id"`
	FileState   string         `json:"file_state"`
	Scheduler   string         `json:"scheduler"`
	FileSize    int64          `json:"file_size"`
	IsUpload    bool           `json:"is_upload"`
	IsCheck     bool           `json:"is_check"`
	IsShard     bool           `json:"is_shard"`
	IsScheduler bool           `json:"is_scheduler"`
	Miners      map[int]string `json:"miners,omitempty"`
}

type RecordFileInfo struct {
	FileId   string   `json:"file_id"`
	FileSize uint64   `json:"file_size"`
	Chunks   []string `json:"chunks"`
}

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
		_ = c.handler(node)
	}()
	err := c.sendFile(node, fid, filetype, pkey, signmsg, sign)
	wg.Wait()
	return err
}

func (c *ConMgr) handler(node *Node) error {
	var (
		err        error
		remoteaddr string
		fs         *os.File
	)

	defer func() {
		c.conn.Close()
		close(c.waitNotify)
		if fs != nil {
			fs.Close()
		}
		if err := recover(); err != nil {
			node.Logs.Pnc("error", utils.RecoverError(err))
		}
	}()

	remoteaddr = c.conn.GetRemoteAddr()
	for !c.conn.IsClose() {
		m, ok := c.conn.GetMsg()
		if !ok {
			return fmt.Errorf("Get msg failed and handler exit")
		}

		if m == nil {
			continue
		}

		switch m.MsgType {
		case MsgHead:
			node.Logs.Upfile("info", fmt.Errorf("[%v] Recv msg type: %v", remoteaddr, MsgHead))
			switch cap(m.Bytes) {
			case configs.TCP_ReadBuffer:
				readBufPool.Put(m.Bytes)
			default:
			}
			// Verify signature
			ok, err = VerifySign(m.Pubkey, m.SignMsg, m.Sign)
			if err != nil || !ok {
				node.Logs.Upfile("err", fmt.Errorf("[%v] VerifySign err", remoteaddr))
				c.conn.SendMsg(buildNotifyMsg("", Status_Err, ""))
				return errors.New("Signature error")
			}

			// //Judge whether the file has been uploaded
			// fileState, err := GetFileState(node.Chain, m.FileHash)
			// if err != nil {
			// 	node.Logs.Upfile("info", fmt.Errorf("[%v] GetFileState err", remoteaddr))
			// 	c.conn.SendMsg(buildNotifyMsg("", Status_Err, ""))
			// 	return errors.New("Get file state error")
			// }

			// // file state
			// if fileState == chain.FILE_STATE_ACTIVE {
			// 	c.conn.SendMsg(buildNotifyMsg("", Status_Err, ""))
			// 	return errors.New("File uploaded")
			// }

			if m.FileSize >= 0 {
				c.fileName = m.FileName
				fstat, err := os.Stat(filepath.Join(c.dir, c.fileName))
				if err == nil {
					if fstat.Size() == int64(m.FileSize) {
						node.Logs.Upfile("err", fmt.Errorf("[%v] File already exist [%s]", remoteaddr, c.fileName))
						c.conn.SendMsg(buildNotifyMsg(c.fileName, Status_Exists, ""))
						continue
					}
				}
			}

			// create file
			fs, err = os.OpenFile(filepath.Join(c.dir, m.FileName), os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
			if err != nil {
				node.Logs.Upfile("err", fmt.Errorf("[%v] OpenFile err: [%s]", remoteaddr, err))
				c.conn.SendMsg(buildNotifyMsg("", Status_Err, ""))
				return err
			}
			// notify suc
			c.conn.SendMsg(buildNotifyMsg(c.fileName, Status_Ok, ""))
			node.Logs.Upfile("info", fmt.Errorf("[%v] MsgHead return suc", remoteaddr))

		case MsgFileSt:
			node.Logs.Upfile("info", fmt.Errorf("[%v] Recv msg type: %v, filehash: %v", remoteaddr, MsgFileSt, m.FileHash))
			switch cap(m.Bytes) {
			case configs.TCP_ReadBuffer:
				readBufPool.Put(m.Bytes)
			default:
			}
			val, err := node.Cache.Get([]byte(m.FileHash))
			if err != nil {
				node.Logs.Upfile("err", fmt.Errorf("[%v] MsgFileSt return err: %v", remoteaddr, err))
				c.conn.SendMsg(buildFileStMsg(m.FileHash, nil))
			} else {
				node.Logs.Upfile("info", fmt.Errorf("[%v] MsgFileSt return suc", remoteaddr))
				c.conn.SendMsg(buildFileStMsg(m.FileHash, val))
			}
			time.Sleep(time.Second)

		case MsgFile:
			// If fs=nil, it means that the file has not been created.
			// You need to request MsgHead message first
			if fs == nil {
				node.Logs.Upfile("err", fmt.Errorf("[%v] MsgFile fs==nil ", remoteaddr))
				c.conn.SendMsg(buildCloseMsg(Status_Err))
				return errors.New("File not open")
			}

			if m.FileSize == 0 {
				node.Logs.Upfile("err", fmt.Errorf("[%v] MsgFile m.FileSize==0", remoteaddr))
				c.conn.SendMsg(buildCloseMsg(Status_Err))
				return errors.New("File write failed")
			}
			_, err = fs.Write(m.Bytes[:m.FileSize])
			if err != nil {
				node.Logs.Upfile("err", fmt.Errorf("[%v] MsgFile fs==nil ", remoteaddr))
				c.conn.SendMsg(buildCloseMsg(Status_Err))
				return errors.New("File write failed")
			}
			switch cap(m.Bytes) {
			case configs.TCP_ReadBuffer:
				readBufPool.Put(m.Bytes)
			default:
			}

		case MsgEnd:
			node.Logs.Upfile("info", fmt.Errorf("[%v] Recv msg type: %v", remoteaddr, MsgEnd))
			switch cap(m.Bytes) {
			case configs.TCP_ReadBuffer:
				readBufPool.Put(m.Bytes)
			default:
			}
			info, err := fs.Stat()
			if err != nil {
				node.Logs.Upfile("err", fmt.Errorf("[%v] MsgEnd fs.Stat err: %v", remoteaddr, err))
				c.conn.SendMsg(buildNotifyMsg("", Status_Err, ""))
				return errors.New("Invalid file")
			}

			if info.Size() != int64(m.FileSize) {
				node.Logs.Upfile("err", fmt.Errorf("[%v] info.Size(%d) != m.FileSize(%d)", remoteaddr, info.Size(), m.FileSize))
				c.conn.SendMsg(buildNotifyMsg("", Status_Err, ""))
				return fmt.Errorf("file.size %v rece size %v \n", info.Size(), m.FileSize)
			}
			c.conn.SendMsg(buildNotifyMsg(c.fileName, Status_Ok, ""))
			node.Logs.Upfile("info", fmt.Errorf("[%v] MsgEnd return suc", remoteaddr))

			// close fs
			fs.Close()
			fs = nil

			// Last file Mark
			if m.LastMark {
				allChunksPath, _ := filepath.Glob(c.dir + "/" + m.FileHash + "*")
				filesize := binary.BigEndian.Uint64(m.SignMsg)
				node.Logs.Upfile("info", fmt.Errorf("[%v] MsgEnd recv all chunks: %v", remoteaddr, allChunksPath))

				var fileSt = FileStoreInfo{
					FileId:      m.FileHash,
					FileSize:    int64(filesize),
					FileState:   chain.FILE_STATE_PENDING,
					Scheduler:   fmt.Sprintf("%s:%s", node.Confile.GetServiceAddr(), node.Confile.GetServicePort()),
					IsUpload:    true,
					IsCheck:     true,
					IsShard:     true,
					IsScheduler: true,
				}

				b, err := json.Marshal(&fileSt)
				if err != nil {
					node.Logs.Upfile("err", fmt.Errorf("[%v] Marshal err: %v", m.FileHash, err.Error()))
				} else {
					node.Cache.Put([]byte(m.FileHash), b)
				}

				var record RecordFileInfo
				record.FileId = m.FileHash
				record.FileSize = filesize
				record.Chunks = allChunksPath
				b, err = json.Marshal(&record)
				if err != nil {
					node.Logs.Upfile("err", fmt.Errorf("[%v] MsgEnd record chunks: %v, Marshal err: %v", remoteaddr, allChunksPath, err))
					go node.FileBackupManagement(m.FileHash, int64(filesize), allChunksPath)
				} else {
					f, err := os.Create(filepath.Join(node.TraceDir, m.FileHash))
					if err != nil {
						node.Logs.Upfile("err", fmt.Errorf("[%v] MsgEnd record chunks: %v, Create err: %v", remoteaddr, allChunksPath, err))
						go node.FileBackupManagement(m.FileHash, int64(filesize), allChunksPath)
					} else {
						f.Write(b)
						f.Sync()
						f.Close()
					}
				}
			}

		case MsgNotify:
			node.Logs.Upfile("info", fmt.Errorf("[%v] Recv msg type: %v", remoteaddr, MsgNotify))
			c.waitNotify <- m.Bytes[0] == byte(Status_Ok)
			switch cap(m.Bytes) {
			case configs.TCP_ReadBuffer:
				readBufPool.Put(m.Bytes)
			default:
			}

		case MsgClose:
			node.Logs.Upfile("info", fmt.Errorf("[%v] Recv msg type: %v", remoteaddr, MsgClose))
			switch cap(m.Bytes) {
			case configs.TCP_ReadBuffer:
				readBufPool.Put(m.Bytes)
			default:
			}
			return errors.New("Close message")

		case MsgVersion:
			node.Logs.Upfile("info", fmt.Errorf("[%v] Recv msg type: %v", remoteaddr, MsgVersion))
			switch cap(m.Bytes) {
			case configs.TCP_ReadBuffer:
				readBufPool.Put(m.Bytes)
			default:
			}
			c.conn.SendMsg(buildVersionMsg(configs.Version))

		default:
			node.Logs.Upfile("info", fmt.Errorf("[%v] Recv msg Invalid type: %v", remoteaddr, m.MsgType))
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

	n.Logs.Upfile("info", fmt.Errorf("[%v] Start the file backup management", fid))

	var fileSt = FileStoreInfo{
		FileId:      fid,
		FileSize:    fsize,
		FileState:   chain.FILE_STATE_PENDING,
		Scheduler:   fmt.Sprintf("%s:%s", n.Confile.GetServiceAddr(), n.Confile.GetServicePort()),
		IsUpload:    true,
		IsCheck:     true,
		IsShard:     true,
		IsScheduler: true,
	}

	if len(chunks) <= 0 {
		fileSt.IsScheduler = false
		b, _ := json.Marshal(&fileSt)
		n.Cache.Put([]byte(fid), b)
		n.Logs.Upfile("err", fmt.Errorf("[%v] Not found", fid))
		return
	}

	b, err := json.Marshal(&fileSt)
	if err != nil {
		n.Logs.Upfile("err", fmt.Errorf("[%v] Marshal err: %v", fid, err.Error()))
	} else {
		n.Cache.Put([]byte(fid), b)
	}

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
		fileSt.Miners[i], _ = utils.EncodePublicKeyAsCessAccount(chunksInfo[i].MinerAcc[:])
		b, _ := json.Marshal(&fileSt)
		n.Cache.Put([]byte(fid), b)
		n.Logs.Upfile("info", fmt.Errorf("[%v] backup suc", chunks[i]))
		i++
	}

	// Submit the file meta information to the chain
	for {
		txhash, err = n.Chain.SubmitFileMeta(fid, uint64(fsize), chunksInfo)
		if err != nil && txhash == "" {
			n.Logs.Upfile("error", fmt.Errorf("[%v] Submit filemeta fail: %v", fid, err))
			time.Sleep(configs.BlockInterval)
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

// file backup management
func (n *Node) TrackFileBackupManagement(fid string, fsize int64, chunks []string) error {
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

	var fileSt = FileStoreInfo{
		FileId:      fid,
		FileSize:    fsize,
		FileState:   chain.FILE_STATE_PENDING,
		Scheduler:   fmt.Sprintf("%s:%s", n.Confile.GetServiceAddr(), n.Confile.GetServicePort()),
		IsUpload:    true,
		IsCheck:     true,
		IsShard:     true,
		IsScheduler: true,
	}

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
		fileSt.Miners[i], _ = utils.EncodePublicKeyAsCessAccount(chunksInfo[i].MinerAcc[:])
		b, _ := json.Marshal(&fileSt)
		n.Cache.Put([]byte(fid), b)
		n.Logs.Upfile("info", fmt.Errorf("[%v] backup suc", chunks[i]))
		i++
	}

	// Submit the file meta information to the chain
	var tryCount uint8
	for {
		if tryCount > 5 {
			return fmt.Errorf("SubmitFileMeta failed")
		}
		txhash, err = n.Chain.SubmitFileMeta(fid, uint64(fsize), chunksInfo)
		if err != nil {
			if txhash != "" {
				fileSt.FileState = chain.FILE_STATE_TXFAILED
				break
			}
			tryCount++
			n.Logs.Upfile("error", fmt.Errorf("[%v] Submit filemeta fail: %v", fid, err))
			time.Sleep(configs.BlockInterval)
			continue
		}
		fileSt.FileState = chain.FILE_STATE_ACTIVE
		break
	}

	b, _ := json.Marshal(&fileSt)
	n.Cache.Put([]byte(fid), b)
	n.Logs.Upfile("info", fmt.Errorf("[%v] Submit filemeta txhash: [%v]", fid, txhash))
	return err
}

// processingfile is used to process all copies of the file and the corresponding tag information
func (n *Node) backupFile(fid, fpath string) (chain.BlockInfo, error) {
	var (
		err            error
		ok             bool
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

		ok, _ = n.Cache.Has([]byte(fmt.Sprintf("%s%d", BlackPrefix, minerinfo.Peerid)))
		if ok {
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
			n.Cache.Put([]byte(fmt.Sprintf("%s%d", BlackPrefix, minerinfo.Peerid)), []byte(fmt.Sprintf("%d", time.Now().Unix())))
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
