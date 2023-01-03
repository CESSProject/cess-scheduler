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
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
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
	Chain   chain.IChain
	Logs    logger.ILog
	Cach    db.ICache
	FileDir string
}

type MsgFile struct {
	Token    string `json:"token"`
	RootHash string `json:"roothash"`
	FileHash string `json:"filehash"`
	FileSize int64  `json:"filesize"`
	Lastfile bool   `json:"lastfile"`
	Data     []byte `json:"data"`
}

type MsgConfirm struct {
	Token     string `json:"token"`
	RootHash  string `json:"roothash"`
	SliceHash string `json:"slicehash"`
	ShardId   string `json:"shardId"`
}

var sendFileBufPool = &sync.Pool{
	New: func() interface{} {
		return make([]byte, configs.SIZE_1MiB)
	},
}

// FileRouter Handle
func (f *FileRouter) Handle(ctx context.CancelFunc, request IRequest) {
	fmt.Println("Call FileRouter Handle and msgId=", request.GetMsgID())

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

	fmt.Println("Msg: ", msg.FileHash, msg.FileSize, msg.Lastfile, msg.RootHash)
	fmt.Println("Msg len(data): ", len(msg.Data))
	if !Tokens.Update(msg.Token) {
		request.GetConnection().SendMsg(Msg_Forbidden, nil)
		return
	}

	fpath := filepath.Join(f.FileDir, msg.FileHash)
	fmt.Println("fpath: ", fpath)
	finfo, err := os.Stat(fpath)
	if err == nil {
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
	}
	fmt.Println("Open file: ", fpath)
	fs, err := os.OpenFile(fpath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		fmt.Println("OpenFile  error")
		ctx()
		return
	}

	fs.Write(msg.Data)
	err = fs.Sync()
	if err != nil {
		fs.Close()
		fmt.Println("Sync  error")
		ctx()
		return
	}
	fmt.Println("Write file: ", fpath)
	if msg.Lastfile {
		finfo, err = fs.Stat()
		fmt.Println("finfo.Size(): ", finfo.Size())
		if finfo.Size() >= msg.FileSize {
			request.GetConnection().SendMsg(Msg_OK_FILE, nil)
			//
			appendBuf := make([]byte, configs.SIZE_SLICE-msg.FileSize)
			fs.Write(appendBuf)
			fs.Sync()
			fs.Close()
			go dataBackupMgt(msg.RootHash, f.FileDir, msg.FileSize, f.Chain, f.Logs, f.Cach)
			return
		}
	}
	err = request.GetConnection().SendMsg(Msg_OK, nil)
	if err != nil {
		fmt.Println(err)
	}
	fs.Close()
}

// file backup management
func dataBackupMgt(fid, fdir string, lastsize int64, c chain.IChain, logs logger.ILog, cach db.ICache) {
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
		logs.Upfile("err", err)
		fmt.Println("Get file meta err")
		return
	}

	// file state
	if string(fileMeta.State) == chain.FILE_STATE_ACTIVE {
		return
	}
	acc, _ := c.GetCessAccount()
	var fileSt = StorageProgress{
		FileId:      fid,
		FileSize:    int64(fileMeta.Size),
		FileState:   chain.FILE_STATE_PENDING,
		Scheduler:   acc,
		IsUpload:    true,
		IsCheck:     true,
		IsShard:     true,
		IsScheduler: true,
	}
	logs.Upfile("info", fmt.Errorf("[%v] Start the file backup management", fid))

	b, _ := json.Marshal(&fileSt)
	cach.Put([]byte(fid), b)

	var chunks = make([]string, len(fileMeta.Backups[0].Slice_info))
	for i := 0; i < len(fileMeta.Backups[0].Slice_info); i++ {
		_, err = os.Stat(filepath.Join(fdir, string(fileMeta.Backups[0].Slice_info[i].Slice_hash[:])))
		if err != nil {
			logs.Upfile("err", err)
			fmt.Println("os.Stat err: ", filepath.Join(fdir, string(fileMeta.Backups[0].Slice_info[i].Slice_hash[:])))
			fileSt.IsScheduler = false
			b, _ := json.Marshal(&fileSt)
			cach.Put([]byte(fid), b)
			return
		}
		chunks[i] = filepath.Join(fdir, string(fileMeta.Backups[0].Slice_info[i].Slice_hash[:]))
	}

	if len(chunks) <= 0 {
		logs.Upfile("err", fmt.Errorf("[%v] Not slice found", fid))
		fmt.Println("len(chunks)==0")
		return
	}

	var sliceSum [configs.BackupNum][]chain.SliceSummary
	for i := uint8(0); i < configs.BackupNum; i++ {
		sliceSum[i] = make([]chain.SliceSummary, 0)
	}
	fileSt.Backups = make([]map[int]string, configs.BackupNum)
	for i := uint8(0); i < configs.BackupNum; i++ {
		fileSt.Backups[i] = make(map[int]string)
	}

	var lastfile = false
	var fsize int64
	for bck := uint8(0); bck < configs.BackupNum; bck++ {
		for i := 0; i < len(chunks); {
			if (i + 1) == len(chunks) {
				lastfile = true
				fsize = lastsize
			} else {
				lastfile = false
				fsize = configs.SIZE_SLICE
			}
			val, err := backupFile(fid, chunks[i], i, fsize, lastfile, c, logs, cach)
			if err != nil {
				time.Sleep(time.Second)
				continue
			}
			sliceSum[bck] = append(sliceSum[bck], val)
			fileSt.Backups[bck][i], _ = utils.EncodePublicKeyAsCessAccount(val.Miner_acc[:])
			b, _ := json.Marshal(&fileSt)
			cach.Put([]byte(fid), b)
			logs.Upfile("info", fmt.Errorf("[%v] backup suc", chunks[i]))
			fmt.Println("backup suc: ", chunks[i])
			i++
		}
	}

	// Submit the file meta information to the chain
	var tryCount uint8
	for {
		txhash, err = c.PackDeal(fid, sliceSum)
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
func backupFile(fid, fpath string, index int, fsize int64, lastfile bool, c chain.IChain, logs logger.ILog, cach db.ICache) (chain.SliceSummary, error) {
	var (
		err            error
		rtnValue       chain.SliceSummary
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

		if BlackMiners.IsExist(minerinfo.Ip) {
			continue
		}

		conTcp, err := dialTcpServer(minerinfo.Ip)
		if err != nil {
			BlackMiners.Add(minerinfo.Ip)
			logs.Upfile("err", fmt.Errorf("dial %v err: %v", minerinfo.Ip, err))
			continue
		}

		token, err := AuthReq(conTcp, c.GetMnemonicSeed())
		if err != nil {
			conTcp.Close()
			logs.Upfile("err", fmt.Errorf("dial %v err: %v", minerinfo.Ip, err))
			continue
		}

		err = FileReq(conTcp, token, fid, fpath, lastfile, fsize)
		if err != nil {
			conTcp.Close()
			continue
		}

		rtnValue, err = ConfirmReq(conTcp, token, fid, filepath.Base(fpath), index)
		if err != nil {
			conTcp.Close()
			continue
		}

		conTcp.Close()

		return rtnValue, nil
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

func FileReq(conn net.Conn, token, fid string, fpath string, lastfile bool, fsize int64) error {
	var (
		err     error
		num     int
		tempBuf []byte
		msgHead IMessage
		fs      *os.File
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

	fs, err = os.Open(fpath)
	if err != nil {
		return err
	}

	message.FileHash = filepath.Base(fpath)
	message.FileSize = fsize
	message.Lastfile = lastfile

	for {
		num, err = fs.Read(readBuf)
		if err != nil && err != io.EOF {
			return err
		}
		if num == 0 {
			break
		}
		num += num
		if lastfile && num >= int(fsize) {
			bound := cap(readBuf) + int(fsize) - num
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

		if msgHead.GetMsgID() != Msg_OK {
			return fmt.Errorf("send file error")
		}

		if lastfile && num >= int(fsize) {
			return nil
		}
	}

	return err
}

func ConfirmReq(conn net.Conn, token, fid, slicehash string, index int) (chain.SliceSummary, error) {
	var (
		err     error
		tempBuf []byte
		msgHead IMessage
		message = MsgConfirm{
			Token:     token,
			RootHash:  fid,
			SliceHash: slicehash,
		}
		dp       = NewDataPack()
		headData = make([]byte, dp.GetHeadLen())
	)

	if index < 10 {
		message.ShardId = fmt.Sprintf("%v.00%d", fid, uint8(index))
	} else if index < 100 {
		message.ShardId = fmt.Sprintf("%v.0%d", fid, uint8(index))
	} else {
		message.ShardId = fmt.Sprintf("%v.%d", fid, uint8(index))
	}

	tempBuf, err = json.Marshal(&message)
	if err != nil {
		return chain.SliceSummary{}, err
	}

	//send  message
	tempBuf, _ = dp.Pack(NewMsgPackage(Msg_Confirm, tempBuf))
	_, err = conn.Write(tempBuf)
	if err != nil {
		return chain.SliceSummary{}, err
	}

	//read head
	_, err = io.ReadFull(conn, headData)
	if err != nil {
		return chain.SliceSummary{}, err
	}

	msgHead, err = dp.Unpack(headData)
	if err != nil {
		return chain.SliceSummary{}, err
	}

	if msgHead.GetMsgID() != Msg_OK || msgHead.GetDataLen() <= 0 {
		return chain.SliceSummary{}, fmt.Errorf("confirm req error")
	}

	var buf = make([]byte, msgHead.GetDataLen())

	//read data
	_, err = io.ReadFull(conn, buf)
	if err != nil {
		return chain.SliceSummary{}, err
	}
	var rtnValue chain.SliceSummary
	err = json.Unmarshal(buf, &rtnValue)
	if err != nil {
		return chain.SliceSummary{}, err
	}
	return rtnValue, nil
}
