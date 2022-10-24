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
	"log"
	"math"
	"net"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/pbc"
	"github.com/CESSProject/cess-scheduler/pkg/utils"
	cesskeyring "github.com/CESSProject/go-keyring"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

var TCP_ConnLength atomic.Uint32

func init() {
	TCP_ConnLength.Store(0)
}

func (n *Node) NewServer(conn NetConn, dir string) Server {
	n.Conn = &ConMgr{
		conn: conn,
		dir:  dir,
		stop: make(chan struct{}),
	}
	return n
}

func (n *Node) Start() {
	n.Conn.conn.HandlerLoop()

	err := n.handler()
	if err != nil {
		log.Println(err)
		return
	}

}

func (n *Node) handler() error {
	var fs *os.File
	var fillerFs *os.File
	var err error
	var filler Filler
	var minerAcc []byte
	num := TCP_ConnLength.Load()
	num += 1
	TCP_ConnLength.Store(num)
	defer func() {
		if fs != nil {
			_ = fs.Close()
		}
		num := TCP_ConnLength.Load()
		if num > 0 {
			num -= 1
		}
		TCP_ConnLength.Store(num)
	}()

	for !n.Conn.conn.IsClose() {
		m, ok := n.Conn.conn.GetMsg()
		if !ok {
			return fmt.Errorf("close by connect")
		}
		if m == nil {
			continue
		}

		switch m.MsgType {
		case MsgHead:
			// Verify signature
			ok, err := VerifySign(m.Pubkey, m.SignMsg, m.Sign)
			if err != nil {
				fmt.Println("VerifySign err: ", err)
				n.Conn.conn.SendMsg(NewNotifyMsg(n.Conn.fileName, Status_Err))
				return err
			}
			if !ok {
				fmt.Println("Verify Sign failed: ", err)
				n.Conn.conn.SendMsg(NewNotifyMsg(n.Conn.fileName, Status_Err))
				return err
			}

			//Judge whether the file has been uploaded
			fileState, err := GetFileState(n.Chain, m.FileHash)
			if err != nil {
				fmt.Println("GetFileState err: ", err)
				n.Conn.conn.SendMsg(NewNotifyMsg(n.Conn.fileName, Status_Err))
				return err
			}

			if fileState == chain.FILE_STATE_ACTIVE {
				fmt.Println("Repeated upload: ", m.FileHash)
				n.Conn.conn.SendMsg(NewNotifyMsg(n.Conn.fileName, Status_Err))
				return err
			}

			//TODO: Judge whether the space is enough

			n.Conn.fileName = m.FileName
			fmt.Println("recv head fileName is", n.Conn.fileName)

			fs, err = os.OpenFile(filepath.Join(n.Conn.dir, n.Conn.fileName), os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.ModePerm)
			if err != nil {
				fmt.Println("os.Create err =", err)
				n.Conn.conn.SendMsg(NewNotifyMsg(n.Conn.fileName, Status_Err))
				return err
			}
			fmt.Println("send head is ok")

			n.Conn.conn.SendMsg(NewNotifyMsg(n.Conn.fileName, Status_Ok))

		case MsgFillerHead:
			fmt.Println("Recv a filler head req")
			// Verify signature
			ok, err := VerifySign(m.Pubkey, m.SignMsg, m.Sign)
			if err != nil {
				fmt.Println("VerifySign err: ", err)
				n.Conn.conn.SendMsg(NewNotifyMsg(n.Conn.fileName, Status_Err))
				return err
			}
			if !ok {
				fmt.Println("Verify Sign failed: ", err)
				n.Conn.conn.SendMsg(NewNotifyMsg(n.Conn.fileName, Status_Err))
				return err
			}
			minerAcc = m.Pubkey
			// select a filler
			timer := time.NewTimer(time.Second * 3)
			select {
			case filler = <-C_Filler:
			case <-timer.C:
			}

			n.Conn.conn.SendMsg(NewNotifyFillerMsg(filler.Hash, Status_Ok))
			fmt.Println("Return a filler: ", filler.Hash)
		case MsgFiller:
			fmt.Println("Recv a filler req: ", m.FileName)
			// send filler tag
			fillerFs, err = os.OpenFile(filepath.Join(n.FillerDir, m.FileName), os.O_RDONLY, os.ModePerm)
			if err != nil {
				fmt.Println("OpenFile err: ", filler.TagPath, err)
				n.Conn.conn.SendMsg(NewNotifyMsg("", Status_Err))
				return err
			}
			fillerInfo, _ := fillerFs.Stat()
			for !n.Conn.conn.IsClose() {
				readBuf := BytesPool.Get().([]byte)
				num, err := fillerFs.Read(readBuf)
				if err != nil && err != io.EOF {
					return err
				}
				if num == 0 {
					break
				}
				n.Conn.conn.SendMsg(NewFillerMsg(m.FileName, readBuf[:num]))
			}
			time.Sleep(time.Millisecond * 20)
			n.Conn.conn.SendMsg(NewFillerEndMsg(m.FileName, uint64(fillerInfo.Size())))
			time.Sleep(time.Millisecond * 20)
			n.Conn.conn.SendMsg(NewNotifyMsg(m.FileName, Status_Ok))
			fmt.Println("Return a filler: ", m.FileName)
			if m.FileName == filler.Hash {
				fmt.Println("Finish a filler: ", filler.Hash)
				fillerMetaEle := combineFillerMeta(filler.Hash, minerAcc)
				C_FillerMeta <- fillerMetaEle
				os.Remove(filler.FillerPath)
				os.Remove(filler.TagPath)
			}
		case MsgFile:
			if fs == nil {
				fmt.Println(n.Conn.fileName, "file is not open!")
				n.Conn.conn.SendMsg(NewCloseMsg(n.Conn.fileName, Status_Err))
				return errors.New("file is not open")
			}
			_, err = fs.Write(m.Bytes)
			if err != nil {
				fmt.Println("file.Write err =", err)
				n.Conn.conn.SendMsg(NewCloseMsg(n.Conn.fileName, Status_Err))
				return err
			}
		case MsgEnd:
			info, _ := fs.Stat()
			if info.Size() != int64(m.FileSize) {
				err = fmt.Errorf("file.size %v rece size %v \n", info.Size(), m.FileSize)
				n.Conn.conn.SendMsg(NewCloseMsg(n.Conn.fileName, Status_Err))
				return err
			}

			fmt.Printf("save file %v is success \n", info.Name())
			n.Conn.conn.SendMsg(NewNotifyMsg(n.Conn.fileName, Status_Ok))

			fmt.Printf("close file %v is success \n", n.Conn.fileName)
			_ = fs.Close()
			fs = nil

			fmt.Println("Lastmark: ", m.LastMark)

			if m.LastMark {
				fmt.Println("The last file transfered")
				fmt.Println("m.hash:", m.FileHash)
				hash := m.FileHash
				fmt.Println("m.hash:", m.FileHash)
				fmt.Println("hash:", hash)
				allChunksPath, _ := filepath.Glob(n.Conn.dir + "/" + m.FileHash + "*")
				var filesize uint64
				filesize = binary.BigEndian.Uint64(m.SignMsg)
				fmt.Println(filesize)
				fmt.Println(allChunksPath)
				// backup file to miner
				go n.FileBackupManagement(hash, int64(filesize), allChunksPath)
			}
		case MsgNotify:
			n.Conn.waitNotify <- m.Bytes[0] == byte(Status_Ok)
			if !(m.Bytes[0] == byte(Status_Ok)) {
				return errors.New("failed")
			}
		case MsgClose:
			fmt.Printf("recv close msg ....\n")
			if m.Bytes[0] != byte(Status_Ok) {
				return fmt.Errorf("server an error occurred")
			}
			return nil
		}
	}
	return err
}

func (n *Node) NewClient(conn NetConn, dir string, files []string) Client {
	n.Conn = &ConMgr{
		conn:       conn,
		dir:        dir,
		sendFiles:  files,
		waitNotify: make(chan bool, 1),
		stop:       make(chan struct{}),
	}
	return n
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
	c.conn.SendMsg(NewCloseMsg(c.fileName, Status_Ok))
	timer := time.NewTimer(5 * time.Second)
	select {
	case <-timer.C:
		return err
	}
}

func (c *ConMgr) sendSingleFile(filePath string, fid string, lastmark bool, pkey, signmsg, sign []byte) error {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("open file err %v \n", err)
		return err
	}

	defer func() {
		if file != nil {
			_ = file.Close()
		}
	}()
	fileInfo, _ := file.Stat()

	log.Println("Ready to write file: ", filePath)
	m := NewHeadMsg(fileInfo.Name(), fid, lastmark, pkey, signmsg, sign)
	c.conn.SendMsg(m)

	timer := time.NewTimer(5 * time.Second)
	select {
	case ok := <-c.waitNotify:
		if !ok {
			return fmt.Errorf("send err")
		}
	case <-timer.C:
		return fmt.Errorf("wait server msg timeout")
	}

	for !c.conn.IsClose() {
		readBuf := BytesPool.Get().([]byte)

		n, err := file.Read(readBuf)
		if err != nil && err != io.EOF {
			return err
		}

		if n == 0 {
			break
		}

		c.conn.SendMsg(NewFileMsg(c.fileName, readBuf[:n]))
	}

	c.conn.SendMsg(NewEndMsg(c.fileName, uint64(fileInfo.Size()), lastmark))
	waitTime := fileInfo.Size() / 1024 / 10
	if waitTime < 5 {
		waitTime = 5
	}

	timer = time.NewTimer(time.Second * time.Duration(waitTime))
	select {
	case ok := <-c.waitNotify:
		if !ok {
			return fmt.Errorf("send err")
		}
	case <-timer.C:
		return fmt.Errorf("wait server msg timeout")
	}

	log.Println("Send " + filePath + " file success...")
	return nil
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
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
	fmt.Println("Start the file backup management...")
	fmt.Println("fid: ", fid)
	fmt.Println("fsize: ", fsize)
	fmt.Println("chunks: ", chunks)
	n.Logs.Log("upfile", "info", fmt.Errorf("[%v] Start the file backup management", fid))

	var chunksInfo = make([]chain.BlockInfo, len(chunks))

	for i := 0; i < len(chunks); {
		chunksInfo[i], err = n.backupFile(fid, chunks[i])
		time.Sleep(time.Second)
		if err != nil {
			fmt.Println("backup failed: ", chunks[i])
			continue
		}
		fmt.Println("backup suc: ", chunks[i])
		i++
	}

	// Submit the file meta information to the chain
	for {
		txhash, err = n.Chain.SubmitFileMeta(fid, uint64(fsize), chunksInfo)
		if txhash == "" {
			n.Logs.Log("upfile", "error", fmt.Errorf("[%v] Submit filemeta fail: %v", fid, err))
			time.Sleep(configs.BlockInterval)
			continue
		}
		break
	}
	n.Logs.Log("upfile", "info", fmt.Errorf("[%v] Submit filemeta [%v]", fid, txhash))
	return
}

// processingfile is used to process all copies of the file and the corresponding tag information
func (n *Node) backupFile(fid, fpath string) (chain.BlockInfo, error) {
	var (
		err            error
		rtnValue       chain.BlockInfo
		allMinerPubkey []types.AccountID
	)
	defer func() {
		if err := recover(); err != nil {
			n.Logs.Log("panic", "error", utils.RecoverError(err))
		}
	}()
	fname := filepath.Base(fpath)
	n.Logs.Log("upfile", "info", fmt.Errorf("[%v] Prepare to store to miner", fname))

	fstat, err := os.Stat(fpath)
	if err != nil {
		n.Logs.Log("upfile", "error", fmt.Errorf("[%v] %v", fname, err))
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
			n.Logs.Log("upfile", "error", fmt.Errorf("[%v] PoDR2ProofCommit: %v", fname, err))
			return rtnValue, err
		}
		select {
		case commitResponse = <-commitResponseCh:
		}
		if commitResponse.StatueMsg.StatusCode != pbc.Success {
			n.Logs.Log("upfile", "error", fmt.Errorf("[%v] Failed to calculate the file tag", fname))
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
			n.Logs.Log("upfile", "error", fmt.Errorf("[%v] %v", fname, err))
			return rtnValue, err
		}

		//Save tag information to file
		ftag, err := os.OpenFile(fileTagPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
		if err != nil {
			n.Logs.Log("upfile", "error", fmt.Errorf("[%v] %v", fname, err))
			return rtnValue, err
		}
		ftag.Write(tag_bytes)

		//flush to disk
		err = ftag.Sync()
		if err != nil {
			n.Logs.Log("upfile", "error", fmt.Errorf("[%v] %v", fname, err))
			ftag.Close()
			os.Remove(fileTagPath)
			return rtnValue, err
		}
		ftag.Close()
	}

	// Get the publickey of all miners in the chain
	for len(allMinerPubkey) == 0 {
		allMinerPubkey, err = n.Chain.GetAllStorageMiner()
		if err != nil {
			time.Sleep(time.Second * time.Duration(utils.RandomInRange(3, 10)))
		}
	}

	// Disrupt the order of miners
	utils.RandSlice(allMinerPubkey)

	n.Logs.Log("upfile", "info", fmt.Errorf("[%v] %v miners found", fname, len(allMinerPubkey)))

	var minerinfo chain.Cache_MinerInfo
	msg := utils.GetRandomcode(16)
	kr, _ := cesskeyring.FromURI(n.Chain.GetMnemonicSeed(), cesskeyring.NetSubstrate{})
	// sign message
	sign, err := kr.Sign(kr.SigningContext([]byte(msg)))
	if err != nil {
		n.Logs.Log("upfile", "error", fmt.Errorf("[%v] %v", fname, err))
		return rtnValue, err
	}
	for i := 0; i < len(allMinerPubkey); i++ {

		pkey, err := utils.DecodePublicKeyOfCessAccount("cXfyomKDABfehLkvARFE854wgDJFMbsxwAJEHezRb6mfcAi2y")
		//minercache, err := n.Cache.Get(allMinerPubkey[i][:])
		minercache, err := n.Cache.Get(pkey)
		if err != nil {
			continue
		}

		err = json.Unmarshal(minercache, &minerinfo)
		if err != nil {
			n.Cache.Delete(allMinerPubkey[i][:])
			continue
		}

		//dstURL := string(base58.Decode(minerinfo.Ip))
		dstURL := "43.128.134.24:15001"
		tcpAddr, err := net.ResolveTCPAddr("tcp", dstURL)
		if err != nil {
			continue
		}

		conTcp, err := net.DialTCP("tcp", nil, tcpAddr)
		if err != nil {
			continue
		}

		srv := n.NewClient(NewTcp(conTcp), "", []string{fpath, fileTagPath})
		err = srv.SendFile(fid, n.Chain.GetPublicKey(), []byte(msg), sign[:])
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		rtnValue.BlockId = types.NewBytes([]byte(fname))
		rtnValue.BlockSize = types.U64(fstat.Size())
		rtnValue.MinerAcc = types.NewAccountID(pkey) //allMinerPubkey[i]
		rtnValue.MinerIp = types.NewBytes([]byte(minerinfo.Ip))
		rtnValue.MinerId = types.U64(minerinfo.Peerid)
		rtnValue.BlockNum = types.U32(blocknum)
		break
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
