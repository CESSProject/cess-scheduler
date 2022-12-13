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

package serve

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/CESSProject/cess-scheduler/configs"
	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/db"
	"github.com/CESSProject/cess-scheduler/pkg/logger"
	"github.com/CESSProject/cess-scheduler/pkg/utils"
	cesskeyring "github.com/CESSProject/go-keyring"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

// FileRouter
type FileRouter struct {
	BaseRouter
	Chain   chain.Chainer
	Logs    logger.Logger
	Cach    db.Cacher
	FileDir string
}

type MsgFile struct {
	Token    string `json:"token"`
	RootHash string `json:"roothash"`
	FileHash string `json:"filehash"`
	FileSize int64  `json:"filesize"`
	LastSize int64  `json:"lastsize"`
	LastFile bool   `json:"lastfile"`
	Data     []byte `json:"data"`
}

var sendFileBufPool = &sync.Pool{
	New: func() interface{} {
		return make([]byte, configs.SIZE_1MiB)
	},
}

// FileRouter Handle
func (f *FileRouter) Handle(ctx context.CancelFunc, request IRequest) {
	fmt.Println("Call FileRouter Handle")
	fmt.Println("recv from client : msgId=", request.GetMsgID())

	if request.GetMsgID() != Msg_File {
		fmt.Println("MsgId error")
		ctx()
		return
	}

	var msg MsgFile
	err := json.Unmarshal(request.GetData(), &msg)
	if err != nil {
		fmt.Println("Msg format error")
		ctx()
		return
	}

	if !Tokens.Update(msg.Token) {
		request.GetConnection().SendMsg(Msg_Forbidden, nil)
		return
	}

	fpath := filepath.Join(f.FileDir, msg.FileHash)
	finfo, err := os.Stat(fpath)
	if err != nil {
		request.GetConnection().SendBuffMsg(Msg_ServerErr, nil)
		return
	}

	if finfo.Size() == configs.SIZE_SLICE {
		hash, _ := utils.CalcPathSHA256(fpath)
		if hash == msg.FileHash {
			request.GetConnection().SendBuffMsg(Msg_OK_FILE, nil)
			return
		}
	} else if finfo.Size() > configs.SIZE_SLICE {
		request.GetConnection().SendMsg(Msg_ClientErr, nil)
		return
	}

	fs, err := os.OpenFile(msg.FileHash, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		fmt.Println("OpenFile  error")
		ctx()
		return
	}
	defer fs.Close()

	fs.Write(msg.Data)
	err = fs.Sync()
	if err != nil {
		fmt.Println("Sync  error")
		ctx()
		return
	}

	if msg.LastFile {
		finfo, err = fs.Stat()
		if finfo.Size() == msg.LastSize {
			request.GetConnection().SendMsg(Msg_OK_FILE, nil)
			//
			appendBuf := make([]byte, configs.SIZE_SLICE-msg.LastSize)
			fs.Write(appendBuf)
			fs.Sync()
			go dataBackupMgt(msg.RootHash, f.FileDir, msg.LastSize, f.Chain, f.Logs, f.Cach)
			return
		}
	}
	err = request.GetConnection().SendMsg(Msg_OK, nil)
	if err != nil {
		fmt.Println(err)
	}
}

// file backup management
func dataBackupMgt(fid, fdir string, lastsize int64, c chain.Chainer, logs logger.Logger, cach db.Cacher) {
	defer func() {
		if err := recover(); err != nil {
			logs.Pnc("error", utils.RecoverError(err))
		}
	}()
	var (
		err    error
		txhash string
	)

	//Judge whether the file has been uploaded
	fileMeta, err := c.GetFileMetaInfo(fid)
	if err != nil {
		return
	}

	// file state
	if string(fileMeta.State) == chain.FILE_STATE_ACTIVE {
		return
	}

	var fileSt = StorageProgress{
		FileId:      fid,
		FileSize:    int64(configs.SIZE_SLICE * len(fileMeta.Blockups[0].Slice_info)),
		FileState:   chain.FILE_STATE_PENDING,
		IsUpload:    true,
		IsCheck:     true,
		IsShard:     true,
		IsScheduler: true,
	}
	logs.Upfile("info", fmt.Errorf("[%v] Start the file backup management", fid))

	b, _ := json.Marshal(&fileSt)
	cach.Put([]byte(fid), b)

	var chunks = make([]string, len(fileMeta.Blockups[0].Slice_info))
	for i := 0; i < len(fileMeta.Blockups[0].Slice_info); i++ {
		_, err = os.Stat(filepath.Join(fdir, string(fileMeta.Blockups[0].Slice_info[i].Slice_hash[:])))
		if err != nil {
			fileSt.IsScheduler = false
			b, _ := json.Marshal(&fileSt)
			cach.Put([]byte(fid), b)
			return
		}
		chunks[i] = filepath.Join(fdir, string(fileMeta.Blockups[0].Slice_info[i].Slice_hash[:]))
	}

	if len(chunks) <= 0 {
		logs.Upfile("err", fmt.Errorf("[%v] Not slice found", fid))
		return
	}

	var backups = make([]chain.Backup, configs.BackupNum)
	for i := uint8(0); i < configs.BackupNum; i++ {
		backups[i].Slice_info = make([]chain.SliceInfo, len(chunks))
	}
	fileSt.Backups = make([]map[int]string, configs.BackupNum)
	for i := uint8(0); i < configs.BackupNum; i++ {
		fileSt.Backups[i] = make(map[int]string)
	}

	var lastfile = false
	for bck := uint8(0); bck < configs.BackupNum; bck++ {
		backups[bck].Backup_index = types.U8(bck)
		backups[bck].State = types.Bool(true)

		for i := 0; i < len(chunks); {
			lastfile = false
			if (i + 1) == len(chunks) {
				lastfile = true
			}
			backups[bck].Slice_info[i], err = backupFile(fid, chunks[i], i, lastsize, lastfile, c, logs, cach)
			if err != nil {
				time.Sleep(time.Second)
				continue
			}
			fileSt.Backups[bck][i], _ = utils.EncodePublicKeyAsCessAccount(backups[bck].Slice_info[i].Miner_acc[:])
			b, _ := json.Marshal(&fileSt)
			cach.Put([]byte(fid), b)
			logs.Upfile("info", fmt.Errorf("[%v] backup suc", chunks[i]))
			i++
		}
	}
	// Submit the file meta information to the chain
	var tryCount uint8
	for {
		txhash, err = c.SubmitFileMeta(fid, uint64(len(chunks)*configs.SIZE_SLICE), backups)
		if err != nil {
			tryCount++
			if tryCount > 3 {
				fileSt.FileState = chain.ERR_Failed
				b, _ = json.Marshal(&fileSt)
				cach.Put([]byte(fid), b)
				return
			}
			time.Sleep(configs.BlockInterval)

			//Judge whether the file has been uploaded
			fileState, _ := GetFileState(c, fid)

			// file state
			if fileState == chain.FILE_STATE_ACTIVE {
				break
			}
			logs.Upfile("err", fmt.Errorf("[%v] Submit filemeta fail: %v", fid, err))
			continue
		}
		break
	}
	fileSt.FileState = chain.FILE_STATE_ACTIVE
	b, _ = json.Marshal(&fileSt)
	cach.Put([]byte(fid), b)
	logs.Upfile("info", fmt.Errorf("[%v] Submit filemeta [%v]", fid, txhash))
	return
}

// processingfile is used to process all copies of the file and the corresponding tag information
func backupFile(fid, fpath string, index int, lastsize int64, lastfile bool, c chain.Chainer, logs logger.Logger, cach db.Cacher) (chain.SliceInfo, error) {
	var (
		err            error
		rtnValue       chain.SliceInfo
		minerinfo      chain.Cache_MinerInfo
		allMinerPubkey []types.AccountID
	)
	defer func() {
		if err := recover(); err != nil {
			logs.Log("panic", "error", utils.RecoverError(err))
		}
	}()
	fname := filepath.Base(fpath)
	logs.Upfile("info", fmt.Errorf("[%v] Prepare to store to miner", fname))

	fstat, err := os.Stat(fpath)
	if err != nil {
		logs.Upfile("error", fmt.Errorf("[%v] %v", fname, err))
		return rtnValue, err
	}

	// Get the publickey of all miners in the chain
	for {
		allMinerPubkey, err = c.GetAllStorageMiner()
		if err != nil || len(allMinerPubkey) < int(configs.BackupNum) {
			logs.Upfile("err", fmt.Errorf("[%v] The number of miners < 3 or: %v", fname, err))
			time.Sleep(configs.BlockInterval)
			continue
		}
		break
	}

	// Disrupt the order of miners
	utils.RandSlice(allMinerPubkey)

	for i := 0; i < len(allMinerPubkey); i++ {
		minercache, err := cach.Get(allMinerPubkey[i][:])
		if err != nil {
			logs.Upfile("error", fmt.Errorf("[%v] %v", fname, err))
			continue
		}

		err = json.Unmarshal(minercache, &minerinfo)
		if err != nil {
			cach.Delete(allMinerPubkey[i][:])
			logs.Upfile("error", fmt.Errorf("[%v] %v", fname, err))
			continue
		}

		if minerinfo.Free < uint64(fstat.Size()) {
			continue
		}

		conTcp, err := dialTcpServer(minerinfo.Ip)
		if err != nil {
			BlackMiners.Add(minerinfo.Peerid)
			logs.Upfile("err", fmt.Errorf("dial %v err: %v", minerinfo.Ip, err))
			continue
		}

		token, err := AuthReq(conTcp, c.GetMnemonicSeed())
		if err != nil {
			logs.Upfile("err", fmt.Errorf("dial %v err: %v", minerinfo.Ip, err))
			continue
		}

		err = FileReq(conTcp, token, fid, fpath, lastfile, lastsize)
		if err != nil {
			continue
		}

		var blockId chain.SliceId
		var slicehash chain.FileHash
		var blockId_temp string
		if index < 10 {
			blockId_temp = fmt.Sprintf("%v.00%d", fid, uint8(index))
		} else if index < 99 {
			blockId_temp = fmt.Sprintf("%v.0%d", fid, uint8(index))
		} else {
			blockId_temp = fmt.Sprintf("%v.%d", fid, uint8(index))
		}
		if len(blockId_temp) != len(blockId) {
			continue
		}
		for i := 0; i < len(blockId); i++ {
			blockId[i] = types.U8(blockId_temp[i])
		}
		for i := 0; i < len(slicehash); i++ {
			slicehash[i] = types.U8(fname[i])
		}
		rtnValue.Shard_id = blockId

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
		rtnValue.Shard_size = types.U64(fstat.Size())
		rtnValue.Miner_acc = allMinerPubkey[i]
		rtnValue.Miner_ip = ipType
		//rtnValue.MinerId = types.U64(minerinfo.Peerid)

		rtnValue.Slice_hash = slicehash
		break
	}
	if rtnValue.Shard_size == 0 {
		return rtnValue, errors.New("failed")
	}
	return rtnValue, err
}

func AuthReq(conn net.Conn, secret string) (string, error) {
	unsignedMsg := utils.GetRandomcode(configs.TokenLength)

	kr, _ := cesskeyring.FromURI(secret, cesskeyring.NetSubstrate{})
	// sign message
	sign, err := kr.Sign(kr.SigningContext([]byte(unsignedMsg)))
	if err != nil {
		return "", err
	}

	keyring, err := signature.KeyringPairFromSecret(secret, 0)
	if err != nil {
		return "", err
	}

	account, err := utils.EncodePublicKeyAsCessAccount(keyring.PublicKey)
	if err != nil {
		return "", err
	}

	var mesage = MsgAuth{
		Account: account,
		Msg:     unsignedMsg,
		Sign:    sign[:],
	}

	b, err := json.Marshal(&mesage)
	if err != nil {
		return "", err
	}

	dp := NewDataPack()
	//send auth message
	msg, _ := dp.Pack(NewMsgPackage(Msg_Auth, b))
	_, err = conn.Write(msg)
	if err != nil {
		return "", err
	}

	//read head
	headData := make([]byte, dp.GetHeadLen())
	_, err = io.ReadFull(conn, headData)
	if err != nil {
		return "", err
	}

	msgHead, err := dp.Unpack(headData)
	if err != nil {
		return "", err
	}

	if msgHead.GetDataLen() > 0 {
		//read data
		msg := msgHead.(*Message)
		msg.Data = make([]byte, msg.GetDataLen())

		_, err := io.ReadFull(conn, msg.Data)
		if err != nil {
			return "", err
		}
		return string(msg.Data), nil
	}
	return "", fmt.Errorf("Nil head")
}

func FileReq(conn net.Conn, token, fid string, file string, lastfile bool, lastsize int64) error {
	var (
		err     error
		num     int
		tempBuf []byte
		msgHead IMessage
		fs      *os.File
		finfo   os.FileInfo
		message = MsgFile{
			Token:    token,
			RootHash: fid,
			FileHash: "",
			Data:     nil,
		}
		dp       = NewDataPack()
		headData = make([]byte, dp.GetHeadLen())
	)

	readBuf := sendFileBufPool.Get().([]byte)
	defer func() {
		sendFileBufPool.Put(readBuf)
	}()

	fs, err = os.Open(file)
	if err != nil {
		return err
	}
	finfo, err = fs.Stat()
	if err != nil {
		return err
	}
	message.FileHash = filepath.Base(file)
	message.FileSize = finfo.Size()
	message.LastFile = lastfile

	for {
		num, err = fs.Read(readBuf)
		if err != nil && err != io.EOF {
			return err
		}
		if num == 0 {
			break
		}
		num += num
		if message.LastFile && num >= int(lastsize) {
			bound := cap(readBuf) + int(lastsize) - num
			message.Data = readBuf[:bound]
		} else {
			message.Data = readBuf[:num]
		}

		tempBuf, err = json.Marshal(&message)
		if err != nil {
			return err
		}

		//send auth message
		tempBuf, _ = dp.Pack(NewMsgPackage(Msg_File, tempBuf))
		_, err = conn.Write(tempBuf)
		if err != nil {
			return err
		}

		//read head
		_, err = io.ReadFull(conn, headData)
		if err != nil {
			return err
		}

		msgHead, err = dp.Unpack(headData)
		if err != nil {
			return err
		}

		if msgHead.GetMsgID() == Msg_OK_FILE {
			break
		}

		if msgHead.GetMsgID() == Msg_OK {
			if message.LastFile && num >= int(lastsize) {
				return nil
			}
		}

		if msgHead.GetMsgID() != Msg_OK {
			return fmt.Errorf("send file error")
		}
	}

	return err
}
