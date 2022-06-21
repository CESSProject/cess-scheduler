package rpc

import (
	"cess-scheduler/configs"
	"cess-scheduler/internal/chain"
	. "cess-scheduler/internal/logger"
	proof "cess-scheduler/internal/proof/apiv1"
	"cess-scheduler/tools"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"

	. "cess-scheduler/internal/rpc/protobuf"

	keyring "github.com/CESSProject/go-keyring"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"google.golang.org/protobuf/proto"
	"storj.io/common/base58"
)

type WService struct {
}

type authinfo struct {
	publicKey  string
	fileId     string
	fileName   string
	updateTime int64
	blockTotal uint32
}

type authmap struct {
	lock   *sync.Mutex
	users  map[string]string
	tokens map[string]authinfo
}

var am *authmap

func (this *authmap) Delete(key string) {
	this.lock.Lock()
	defer this.lock.Unlock()

	v, ok := this.tokens[key]
	if ok {
		delete(this.users, v.publicKey)
		delete(this.tokens, key)
	}
}

func (this *authmap) DeleteExpired() {
	this.lock.Lock()
	defer this.lock.Unlock()

	for k, v := range this.tokens {
		if time.Since(time.Unix(v.updateTime, 0)).Minutes() > 10 {
			delete(this.users, v.publicKey)
			delete(this.tokens, k)
		}
	}
}

func (this *authmap) UpdateTimeIfExists(key string) string {
	this.lock.Lock()
	defer this.lock.Unlock()

	v, ok := this.users[key]
	if ok {
		info := this.tokens[v]
		info.updateTime = time.Now().Unix()
		this.tokens[v] = info
		return v
	}
	return ""
}

func (this *authmap) UpdateTime(key string) (uint32, string, string, string, error) {
	this.lock.Lock()
	defer this.lock.Unlock()

	v, ok := this.tokens[key]
	if !ok {
		return 0, "", "", "", errors.New("Authentication failed")
	}
	v.updateTime = time.Now().Unix()
	this.tokens[key] = v
	return v.blockTotal, v.fileId, v.publicKey, v.fileName, nil
}

func init() {
	am = new(authmap)
	am.lock = new(sync.Mutex)
	am.users = make(map[string]string, 10)
	am.tokens = make(map[string]authinfo, 10)
}

// Start tcp service.
// If an error occurs, it will exit immediately.
func Rpc_Main() {
	go task_Management()
	srv := NewServer()
	err := srv.Register(configs.RpcService_Scheduler, WService{})
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
	err = http.ListenAndServe(":"+configs.C.ServicePort, srv.WebsocketHandler([]string{"*"}))
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
}

func task_Management() {
	var (
		channel_1 = make(chan bool, 1)
	)
	go task_ClearAuthMap(channel_1)
	for {
		select {
		case <-channel_1:
			go task_ClearAuthMap(channel_1)
		}
	}
}

func task_ClearAuthMap(ch chan bool) {
	defer func() {
		if err := recover(); err != nil {
			Gpnc.Sugar().Infof("%v", tools.RecoverError(err))
		}
		ch <- true
	}()

	for {
		time.Sleep(time.Minute * time.Duration(tools.RandomInRange(2, 5)))
		am.DeleteExpired()
	}
}

// AuthAction is used to generate credentials.
// The return code is 200 for success, non-200 for failure.
// The returned Msg indicates the result reason.
func (WService) AuthAction(body []byte) (proto.Message, error) {
	defer func() {
		if err := recover(); err != nil {
			Gpnc.Sugar().Infof("%v", tools.RecoverError(err))
		}
	}()

	if len(am.users) >= 2000 {
		return &RespBody{Code: 503, Msg: "Server is busy"}, nil
	}

	var b AuthReq
	err := proto.Unmarshal(body, &b)
	if err != nil {
		return &RespBody{Code: 400, Msg: "Bad Requset"}, nil
	}

	if len(b.Msg) == 0 || len(b.Sign) < 64 {
		return &RespBody{Code: 400, Msg: "Invalid Sign"}, nil
	}

	token := am.UpdateTimeIfExists(string(b.PublicKey))
	if token != "" {
		return &RespBody{Code: 200, Msg: "success", Data: []byte(token)}, nil
	}

	// Verify signature
	ss58, err := tools.EncodeToSS58(b.PublicKey)
	if err != nil {
		return &RespBody{Code: 400, Msg: "Invalid PublicKey"}, nil
	}

	verkr, _ := keyring.FromURI(ss58, keyring.NetSubstrate{})

	var sign [64]byte
	for i := 0; i < 64; i++ {
		sign[i] = b.Sign[i]
	}
	ok := verkr.Verify(verkr.SigningContext(b.Msg), sign)
	if !ok {
		return &RespBody{Code: 403, Msg: "Authentication failed"}, nil
	}

	// Verify space
	if b.FileSize == 0 {
		return &RespBody{Code: 400, Msg: "Invalid File Size"}, nil
	}

	//Judge whether the space is enough
	count := 0
	code := configs.Code_404
	userSpace := chain.UserSpaceInfo{}
	for code != configs.Code_200 {
		userSpace, code, err = chain.GetUserSpaceByPuk(types.NewAccountID(b.PublicKey))
		if count > 3 && code != configs.Code_200 {
			Uld.Sugar().Infof("[%v] GetUserSpaceByPuk err: %v", b.FileId, err)
			return &RespBody{Code: int32(code), Msg: err.Error()}, nil
		}
		if code != configs.Code_200 {
			time.Sleep(time.Second * 3)
		} else {
			if new(big.Int).SetUint64(b.FileSize).CmpAbs(new(big.Int).SetBytes(userSpace.RemainingSpace.Bytes())) == -1 {
				return &RespBody{Code: 403, Msg: "Not enough space"}, nil
			}
		}
		count++
	}

	//Judge whether the file has been uploaded
	count = 0
	code = configs.Code_404
	fmeta := chain.FileMetaInfo{}
	for code != configs.Code_200 {
		fmeta, code, err = chain.GetFileMetaInfoOnChain(b.FileId)
		if count > 3 && code != configs.Code_200 {
			Uld.Sugar().Infof("[%v] GetFileMetaInfoOnChain err: %v", b.FileId, err)
			return &RespBody{Code: int32(code), Msg: err.Error()}, nil
		}
		if code != configs.Code_200 {
			time.Sleep(time.Second * 3)
		} else {
			if string(fmeta.FileState) == "active" {
				return &RespBody{Code: 201, Msg: "success"}, nil
			}
		}
		count++
	}

	var info authinfo
	info.publicKey = string(b.PublicKey)
	info.fileId = b.FileId
	info.fileName = b.FileName
	info.updateTime = time.Now().Unix()
	info.blockTotal = b.BlockTotal
	token = tools.GetRandomcode(12)
	am.lock.Lock()
	defer am.lock.Unlock()
	am.users[string(b.PublicKey)] = token
	am.tokens[token] = info
	return &RespBody{Code: 200, Msg: "success", Data: []byte(token)}, nil
}

// WritefileAction is used to handle client requests to upload files.
// The return code is 200 for success, non-200 for failure.
// The returned Msg indicates the result reason.
func (WService) WritefileAction(body []byte) (proto.Message, error) {
	defer func() {
		if err := recover(); err != nil {
			Gpnc.Sugar().Infof("%v", tools.RecoverError(err))
		}
	}()

	if len(am.users) >= 2000 {
		return &RespBody{Code: 503, Msg: "Server is busy"}, nil
	}

	var b FileUploadReq
	err := proto.Unmarshal(body, &b)
	if err != nil {
		return &RespBody{Code: 400, Msg: "Bad Requset"}, nil
	}

	if b.BlockIndex == 0 {
		return &RespBody{Code: 400, Msg: "Invalid parameter"}, nil
	}

	blockTotal, fid, pubkey, fname, err := am.UpdateTime(string(b.Auth))
	if err != nil {
		return &RespBody{Code: 403, Msg: err.Error()}, nil
	}

	fileAbsPath := filepath.Join(configs.FileCacheDir, fid)

	if b.BlockIndex == 1 {
		Uld.Sugar().Infof("+++> Upload file [%v] ", fid)
		_, err = os.Create(fileAbsPath)
		if err != nil {
			Uld.Sugar().Infof("[%v] Create file err: %v", fid, err)
			return &RespBody{Code: 500, Msg: err.Error()}, nil
		}
	}

	f, err := os.OpenFile(fileAbsPath, os.O_RDWR|os.O_APPEND, os.ModePerm)
	if err != nil {
		Uld.Sugar().Infof("[%v] OpenFile err: %v", fid, err)
		return &RespBody{Code: 500, Msg: err.Error()}, nil
	}

	_, err = f.Write(b.FileData)
	if err != nil {
		f.Close()
		os.Remove(fileAbsPath)
		Uld.Sugar().Infof("[%v] f.Write err: %v", fid, err)
		return &RespBody{Code: 500, Msg: err.Error()}, nil
	}

	err = f.Sync()
	if err != nil {
		f.Close()
		os.Remove(fileAbsPath)
		Uld.Sugar().Infof("[%v] f.Sync err: %v", fid, err)
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	}

	f.Close()

	if b.BlockIndex == blockTotal {
		Uld.Sugar().Infof("[%v] All %v chunks saved successfully", fid, blockTotal)
		am.Delete(string(b.Auth))
		filehash, err := calcFileHashByChunks(fileAbsPath, configs.BYTE_SIZE_1GB)
		if err != nil {
			Uld.Sugar().Infof("[%v] calcFileHashByChunks err: %v", fid, err)
			return &RespBody{Code: 500, Msg: err.Error()}, nil
		}
		if filehash != fid[4:] {
			Uld.Sugar().Infof("[%v] Invalid file hash", fid)
			return &RespBody{Code: 400, Msg: "Invalid file hash"}, nil
		}
		go storeFiles(fid, fileAbsPath, fname, pubkey)
		return &RespBody{Code: 200, Msg: "success"}, nil
	}
	Uld.Sugar().Infof("[%v] The %v chunk saved successfully", fid, b.BlockIndex)
	return &RespBody{Code: 200, Msg: "success"}, nil
}

func calcFileHashByChunks(fpath string, chunksize int64) (string, error) {
	if chunksize <= 0 {
		return "", errors.New("Invalid chunk size")
	}
	fstat, err := os.Stat(fpath)
	if err != nil {
		return "", err
	}
	chunkNum := fstat.Size() / chunksize
	if fstat.Size()%chunksize != 0 {
		chunkNum++
	}
	var n int
	var chunkhash, allhash, filehash string
	var buf = make([]byte, chunksize)
	f, err := os.OpenFile(fpath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return "", err
	}
	defer f.Close()
	for i := int64(0); i < chunkNum; i++ {
		f.Seek(i*chunksize, 0)
		n, err = f.Read(buf)
		if err != nil && err != io.EOF {
			return "", err
		}
		chunkhash, err = tools.CalcHash(buf[:n])
		if err != nil {
			return "", err
		}
		allhash += chunkhash
	}
	filehash, err = tools.CalcHash([]byte(allhash))
	if err != nil {
		return "", err
	}
	return filehash, nil
}

func storeFiles(fid, fpath, name, pubkey string) {
	var (
		channel_map = make(chan uint8, 1)
	)
	defer func() {
		if err := recover(); err != nil {
			Gpnc.Sugar().Infof("%v", tools.RecoverError(err))
		}
	}()
	Uld.Sugar().Infof("[%v] Start the file backup management process", fid)

	go backupFile(channel_map, fid, fpath, name, pubkey)

	for {
		select {
		case result := <-channel_map:
			if result == 1 {
				go backupFile(channel_map, fid, fpath, name, pubkey)
			}
			if result == 2 {
				Uld.Sugar().Infof("[%v] File save successfully", fid)
				return
			}
			if result == 3 {
				Uld.Sugar().Infof("[%v] File save failed", fid)
				return
			}
		}
	}
}

// ReadfileAction is used to handle client requests to download files.
// The return code is 0 for success, non-0 for failure.
// The returned Msg indicates the result reason.
// func (WService) ReadfileAction(body []byte) (proto.Message, error) {
// 	var (
// 		err error
// 		b   FileDownloadReq
// 	)
// 	defer func() {
// 		if err := recover(); err != nil {
// 			Gpnc.Sugar().Infof("%v", tools.RecoverError(err))
// 		}
// 	}()
// 	err = proto.Unmarshal(body, &b)
// 	if err != nil {
// 		return &RespBody{Code: 400, Msg: "Bad Request"}, nil
// 	}

// 	if b.FileId == "" || b.BlockIndex == 0 {
// 		return &RespBody{Code: 400, Msg: "Invalid parameter"}, nil
// 	}

// 	Dld.Sugar().Infof("---> Download [%v] %v", b.FileId, b.BlockIndex)

// 	if b.BlockIndex == 1 {
// 		uspace, code, err := chain.GetUserSpaceOnChain(b.WalletAddress)
// 		if err != nil {
// 			Dld.Sugar().Infof("[%v] GetUserSpaceOnChain err: %v", b.FileId, err)
// 			return &RespBody{Code: int32(code), Msg: err.Error()}, nil
// 		}

// 		fmeta, code, err := chain.GetFileMetaInfoOnChain(b.FileId)
// 		if err != nil {
// 			Dld.Sugar().Infof("[%v] GetFileMetaInfoOnChain err: %v", b.FileId, err)
// 			return &RespBody{Code: int32(code), Msg: err.Error()}, nil
// 		}

// 		if uspace.PurchasedSpace.CmpAbs(new(big.Int).SetBytes(uspace.UsedSpace.Bytes())) < 0 {
// 			Dld.Sugar().Infof("[%v] Not enough space", b.FileId)
// 			return &RespBody{Code: 403, Msg: "Not enough space"}, nil
// 		}

// 		if string(fmeta.FileState) != "active" {
// 			Dld.Sugar().Infof("[%v] Please download later", b.FileId)
// 			return &RespBody{Code: 403, Msg: "Please download later"}, nil
// 		}
// 	}

// 	path := filepath.Join(configs.FileCacheDir, b.FileId)
// 	_, err = os.Stat(path)
// 	if err != nil {
// 		os.MkdirAll(path, os.ModeDir)
// 	}
// 	filefullname := filepath.Join(path, b.FileId+".u")
// 	_, err = os.Stat(filefullname)
// 	if err != nil {
// 		// file not exist, query dupl file
// 		fmeta, code, err := chain.GetFileMetaInfoOnChain(b.FileId)
// 		if err != nil {
// 			Dld.Sugar().Infof("[%v] GetFileMetaInfoOnChain err: %v", b.FileId, err)
// 			return &RespBody{Code: int32(code), Msg: err.Error(), Data: nil}, nil
// 		}
// 		for i := 0; i < len(fmeta.FileDupl); i++ {
// 			duplname := filepath.Join(path, string(fmeta.FileDupl[i].DuplId))
// 			_, err = os.Stat(duplname)
// 			if err == nil {
// 				buf, err := ioutil.ReadFile(duplname)
// 				if err != nil {
// 					Dld.Sugar().Infof("[%v] [%v] ReadFile-1 err: %v", b.FileId, duplname, err)
// 					os.Remove(duplname)
// 					continue
// 				}

// 				//aes decryption
// 				ivkey := fmeta.FileDupl[i].RandKey[:16]
// 				bkey := base58.Decode(string(fmeta.FileDupl[i].RandKey))
// 				decrypted, err := encryption.AesCtrDecrypt(buf, []byte(bkey), ivkey)
// 				if err != nil {
// 					Dld.Sugar().Infof("[%v] [%v] AesCtrDecrypt-1 err: ", b.FileId, duplname, err)
// 					os.Remove(duplname)
// 					continue
// 				}
// 				fu, err := os.OpenFile(filefullname, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
// 				if err != nil {
// 					Dld.Sugar().Infof("[%v] [%v] OpenFile-1 err: %v", b.FileId, filefullname, err)
// 					continue
// 				}
// 				fu.Write(decrypted)
// 				err = fu.Sync()
// 				if err != nil {
// 					Dld.Sugar().Infof("[%v] [%v] fu.Sync err: %v", b.FileId, filefullname, err)
// 					fu.Close()
// 					os.Remove(filefullname)
// 					continue
// 				}
// 				fu.Close()
// 				break
// 			}
// 		}
// 	}

// 	fstat, err := os.Stat(filefullname)
// 	if err == nil {
// 		fuser, err := os.OpenFile(filefullname, os.O_RDONLY, os.ModePerm)
// 		if err != nil {
// 			Dld.Sugar().Infof("[%v] [%v] OpenFile-2 err: %v", b.FileId, filefullname, err)
// 			return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
// 		}
// 		defer fuser.Close()
// 		blockTotal := fstat.Size() / configs.RpcFileBuffer
// 		if fstat.Size()%configs.RpcFileBuffer != 0 {
// 			blockTotal += 1
// 		}
// 		var tmp = make([]byte, configs.RpcFileBuffer)
// 		var blockSize int32
// 		var n int

// 		fuser.Seek(int64((b.BlockIndex-1)*configs.RpcFileBuffer), 0)
// 		n, _ = fuser.Read(tmp)
// 		blockSize = int32(n)

// 		respb := &FileDownloadInfo{
// 			FileId:     b.FileId,
// 			BlockTotal: int32(blockTotal),
// 			BlockSize:  blockSize,
// 			BlockIndex: b.BlockIndex,
// 			Data:       tmp[:n],
// 		}
// 		protob, err := proto.Marshal(respb)
// 		if err != nil {
// 			Dld.Sugar().Infof("[%v] [%v] Marshal err: ", b.FileId, filefullname, err)
// 			return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
// 		}
// 		Dld.Sugar().Infof("[%v] [%v] download successful", b.FileId)
// 		return &RespBody{Code: 200, Msg: "success", Data: protob}, nil
// 	}

// 	// download dupl
// 	fmeta, code, err := chain.GetFileMetaInfoOnChain(b.FileId)
// 	if err != nil {
// 		Dld.Sugar().Infof("[%v] GetFileMetaInfoOnChain err: %v", b.FileId, err)
// 		return &RespBody{Code: int32(code), Msg: err.Error(), Data: nil}, nil
// 	}

// 	var client *Client
// 	var index int = -1
// 	for i := 0; i < len(fmeta.FileDupl); i++ {
// 		dstip := "ws://" + string(base58.Decode(string(fmeta.FileDupl[i].MinerIp)))
// 		ctx, _ := context.WithTimeout(context.Background(), 6*time.Second)
// 		client, err = DialWebsocket(ctx, dstip, "")
// 		if err != nil {
// 			continue
// 		}
// 		index = i
// 		break
// 	}

// 	if client != nil && index != -1 {
// 		err = ReadFile2(client, path, string(fmeta.FileDupl[index].DuplId), b.WalletAddress)
// 		if err != nil {
// 			Dld.Sugar().Infof("[%v] ReadFile2 err: %v", b.FileId, err)
// 			return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
// 		}
// 	} else {
// 		Dld.Sugar().Infof("[%v] All miners connection failed", b.FileId)
// 		return &RespBody{Code: 500, Msg: "No miners available", Data: nil}, nil
// 	}

// 	// file not exist, query dupl file
// 	for i := 0; i < len(fmeta.FileDupl); i++ {
// 		duplname := filepath.Join(path, string(fmeta.FileDupl[i].DuplId))
// 		_, err = os.Stat(duplname)
// 		if err == nil {
// 			buf, err := ioutil.ReadFile(duplname)
// 			if err != nil {
// 				Dld.Sugar().Infof("[%v] [%v] ReadFile-3 err: ", b.FileId, duplname, err)
// 				os.Remove(duplname)
// 				continue
// 			}
// 			//aes decryption
// 			ivkey := fmeta.FileDupl[i].RandKey[:16]
// 			bkey := base58.Decode(string(fmeta.FileDupl[i].RandKey))
// 			decrypted, err := encryption.AesCtrDecrypt(buf, bkey, ivkey)
// 			if err != nil {
// 				Dld.Sugar().Infof("[%v] [%v] AesCtrDecrypt-2 err: %v", b.FileId, duplname, err)
// 				os.Remove(duplname)
// 				continue
// 			}
// 			fu, err := os.OpenFile(filefullname, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
// 			if err != nil {
// 				Dld.Sugar().Infof("[%v] [%v] OpenFile-3 err: %v", b.FileId, filefullname, err)
// 				continue
// 			}
// 			fu.Write(decrypted)
// 			err = fu.Sync()
// 			if err != nil {
// 				Dld.Sugar().Infof("[%v] [%v] fu.Sync err: %v", b.FileId, filefullname, err)
// 				fu.Close()
// 				os.Remove(filefullname)
// 				continue
// 			}
// 			fu.Close()
// 			break
// 		}
// 	}

// 	fstat, err = os.Stat(filefullname)
// 	if err == nil {
// 		fuser, err := os.OpenFile(filefullname, os.O_RDONLY, os.ModePerm)
// 		if err != nil {
// 			Dld.Sugar().Infof("[%v] [%v] ReadFile-4 err: ", b.FileId, filefullname, err)
// 			return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
// 		}
// 		defer fuser.Close()
// 		blockTotal := fstat.Size() / configs.RpcFileBuffer
// 		if fstat.Size()%configs.RpcFileBuffer != 0 {
// 			blockTotal += 1
// 		}
// 		var tmp = make([]byte, configs.RpcFileBuffer)
// 		var blockSize int32
// 		var n int
// 		fuser.Seek(int64(b.BlockIndex*configs.RpcFileBuffer), 0)
// 		n, _ = fuser.Read(tmp)
// 		blockSize = int32(n)
// 		respb := &FileDownloadInfo{
// 			FileId:     b.FileId,
// 			BlockTotal: int32(blockTotal),
// 			BlockSize:  blockSize,
// 			BlockIndex: b.BlockIndex,
// 			Data:       tmp[:n],
// 		}
// 		protob, err := proto.Marshal(respb)
// 		if err != nil {
// 			Dld.Sugar().Infof("[%v] [%v] Marshal-2 err: %v", b.FileId, filefullname, err)
// 			return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
// 		}
// 		Dld.Sugar().Infof("[%v] Download successful", b.FileId)
// 		return &RespBody{Code: 200, Msg: "success", Data: protob}, nil
// 	}
// 	Dld.Sugar().Infof("[%v] Download failed", b.FileId)
// 	return &RespBody{Code: 500, Msg: "fail", Data: nil}, nil
// }

type RespSpacetagInfo struct {
	FileId  string         `json:"fileId"`
	Content string         `json:"content"`
	Rule    []byte         `json:"rule"`
	T       proof.FileTagT `json:"file_tag_t"`
	Sigmas  [][]byte       `json:"sigmas"`
}

var spaceFileMap = make(map[string]int64, 10)

// SpacefileAction is used to handle miner requests to download space files.
// The return code is 200 for success, non-200 for failure.
// The returned Msg indicates the result reason.
func (WService) SpacefileAction(body []byte) (proto.Message, error) {
	defer func() {
		if err := recover(); err != nil {
			Gpnc.Sugar().Infof("%v", tools.RecoverError(err))
		}
	}()
	var b SpaceFileReq
	err := proto.Unmarshal(body, &b)
	if err != nil {
		return &RespBody{Code: 400, Msg: "Bad Request"}, nil
	}

	//Prohibit frequent requests
	v, ok := spaceFileMap[string(b.Publickey)]
	if ok {
		if time.Since(time.Unix(v, 0)).Seconds() < 10 {
			return &RespBody{Code: 403, Msg: "Requests too frequently"}, nil
		}
	}
	spaceFileMap[string(b.Publickey)] = time.Now().Unix()

	addr, err := tools.EncodeToCESSAddr(b.Publickey)
	if err != nil {
		return &RespBody{Code: 400, Msg: "Bad publickey"}, nil
	}

	//Log valid requests
	if b.Fileid == "" {
		Spc.Sugar().Infof("[%v] Get space file", addr)
	} else {
		Spc.Sugar().Infof("[%v] Uplink space file meta", addr)
	}

	//Verify miner identity
	ss58addr, err := tools.EncodeToSS58(b.Publickey)
	if err != nil {
		return &RespBody{Code: 400, Msg: "Bad publickey"}, nil
	}
	verkr, _ := keyring.FromURI(ss58addr, keyring.NetSubstrate{})
	var sign [64]byte
	for i := 0; i < 64; i++ {
		sign[i] = b.Sign[i]
	}
	ok = verkr.Verify(verkr.SigningContext(b.Msg), sign)
	if !ok {
		return &RespBody{Code: 403, Msg: "Authentication failed"}, nil
	}

	filebasedir := filepath.Join(configs.SpaceCacheDir, addr)
	_, err = os.Stat(filebasedir)
	if err != nil {
		err = os.MkdirAll(filebasedir, os.ModeDir)
		if err != nil {
			Spc.Sugar().Infof("[%v] mkdir [%v] err: %v", addr, filebasedir, err)
			return &RespBody{Code: 500, Msg: err.Error()}, nil
		}
	}

	filename := fmt.Sprintf("%d", time.Now().Unix())
	filefullpath := filepath.Join(filebasedir, filename)

	if b.Fileid != "" {
		txhash, err := upChainSpaceFileMeta(addr, b.Fileid, filefullpath, b.Publickey)
		if err != nil {
			return &RespBody{Code: 500, Msg: err.Error()}, nil
		}
		return &RespBody{Code: 200, Msg: txhash}, nil
	}

	baseStr := tools.RandStr(configs.LengthOfALine)
	reg := make([]uint8, 1023)
	for i := 0; i < len(reg); i++ {
		reg[i] = uint8(tools.RandomInRange(1, 64))
	}

	//Generate space file
	err = genSpaceFile(filefullpath, baseStr, reg)
	if err != nil {
		Spc.Sugar().Infof("[%v] [%v] Stat err: %v", addr, filefullpath, err)
		os.Remove(filefullpath)
		return &RespBody{Code: 500, Msg: err.Error()}, nil
	}

	// calculate file tag info
	var PoDR2commit proof.PoDR2Commit
	var commitResponse proof.PoDR2CommitResponse
	PoDR2commit.FilePath = filefullpath
	PoDR2commit.BlockSize = configs.BlockSize

	gWait := make(chan bool)
	go func(ch chan bool) {
		runtime.LockOSThread()
		defer func() {
			if err := recover(); err != nil {
				ch <- true
				Gpnc.Sugar().Infof("%v", tools.RecoverError(err))
			}
		}()
		commitResponseCh, err := PoDR2commit.PoDR2ProofCommit(proof.Key_Ssk, string(proof.Key_SharedParams), int64(configs.ScanBlockSize))
		if err != nil {
			ch <- false
			return
		}
		aft := time.After(time.Second * 5)
		select {
		case commitResponse = <-commitResponseCh:
		case <-aft:
			ch <- false
			return
		}
		if commitResponse.StatueMsg.StatusCode != proof.Success {
			ch <- false
		} else {
			ch <- true
		}
	}(gWait)

	if !<-gWait {
		Spc.Sugar().Infof("[%v] PoDR2ProofCommit false", addr)
		return &RespBody{Code: 500, Msg: "unexpected system error"}, nil
	}

	var resp RespSpacetagInfo
	resp.FileId = filename
	resp.Content = baseStr
	resp.Rule = reg
	resp.T = commitResponse.T
	resp.Sigmas = commitResponse.Sigmas
	resp_b, err := json.Marshal(resp)
	if err != nil {
		Spc.Sugar().Infof("[%v] Marshal err: %v", addr, err)
		return &RespBody{Code: 500, Msg: err.Error()}, nil
	}

	Spc.Sugar().Infof("[%v] Generat space file [%v]", addr, filename)
	return &RespBody{Code: 200, Msg: "success", Data: resp_b}, nil
}

//
func WriteData(dst string, service, method string, body []byte) ([]byte, error) {
	dstip := "ws://" + string(base58.Decode(dst))
	dstip = strings.Replace(dstip, " ", "", -1)
	req := &ReqMsg{
		Service: service,
		Method:  method,
		Body:    body,
	}
	ctx1, _ := context.WithTimeout(context.Background(), 6*time.Second)
	client, err := DialWebsocket(ctx1, dstip, "")
	if err != nil {
		return nil, errors.Wrap(err, "DialWebsocket:")
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	resp, err := client.Call(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "Call err:")
	}

	var b RespBody
	err = proto.Unmarshal(resp.Body, &b)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal:")
	}
	if b.Code == 200 {
		return b.Data, nil
	}
	errstr := fmt.Sprintf("%d", b.Code)
	return nil, errors.New("return code:" + errstr)
}

//
func WriteData2(cli *Client, service, method string, body []byte) ([]byte, error) {
	req := &ReqMsg{
		Service: service,
		Method:  method,
		Body:    body,
	}
	ctx, _ := context.WithTimeout(context.Background(), 90*time.Second)
	resp, err := cli.Call(ctx, req)
	if err != nil {
		return nil, errors.Wrap(err, "Call err:")
	}

	var b RespBody
	err = proto.Unmarshal(resp.Body, &b)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal:")
	}
	if b.Code == 200 {
		return b.Data, nil
	}
	errstr := fmt.Sprintf("%d", b.Code)
	return nil, errors.New("return code:" + errstr)
}

//
func ReadFile(dst string, path, fid, walletaddr string) error {
	dstip := "ws://" + string(base58.Decode(dst))
	dstip = strings.Replace(dstip, " ", "", -1)
	reqbody := FileDownloadReq{
		FileId:        fid,
		WalletAddress: walletaddr,
		BlockIndex:    0,
	}
	bo, err := proto.Marshal(&reqbody)
	if err != nil {
		return err
	}
	req := &ReqMsg{
		Service: configs.RpcService_Miner,
		Method:  configs.RpcMethod_Miner_ReadFile,
		Body:    bo,
	}
	var client *Client
	var count = 0
	for {
		client, err = DialWebsocket(context.Background(), dstip, "")
		if err != nil {
			count++
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 5)))
		} else {
			break
		}
		if count > 10 {
			Err.Sugar().Errorf("DialWebsocket failed more than 10 times:%v", err)
			return err
		}
	}
	defer client.Close()
	ctx, _ := context.WithTimeout(context.Background(), 90*time.Second)
	resp, err := client.Call(ctx, req)
	if err != nil {
		return err
	}

	var b RespBody
	var b_data FileDownloadInfo
	err = proto.Unmarshal(resp.Body, &b)
	if err != nil {
		return err
	}
	if b.Code == 200 {
		err = proto.Unmarshal(b.Data, &b_data)
		if err != nil {
			return err
		}
		if b_data.BlockTotal <= 1 {
			f, err := os.OpenFile(filepath.Join(path, fid), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
			if err != nil {
				return err
			}
			f.Write(b_data.Data)
			f.Close()
			return nil
		} else {
			if b_data.BlockIndex == 0 {
				f, err := os.OpenFile(filepath.Join(path, fid+"-0"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
				if err != nil {
					return err
				}
				f.Write(b_data.Data)
				f.Close()
			}
		}
		for i := int32(1); i < b_data.BlockTotal; i++ {
			reqbody := FileDownloadReq{
				FileId:        fid,
				WalletAddress: walletaddr,
				BlockIndex:    i,
			}
			body_loop, err := proto.Marshal(&reqbody)
			if err != nil {
				if i > 1 {
					i--
				}
				continue
			}
			req := &ReqMsg{
				Service: configs.RpcService_Miner,
				Method:  configs.RpcMethod_Miner_ReadFile,
				Body:    body_loop,
			}
			ctx2, cancel2 := context.WithTimeout(context.Background(), 90*time.Second)
			resp_loop, err := client.Call(ctx2, req)
			defer cancel2()
			if err != nil {
				if i > 1 {
					i--
				}
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
				continue
			}

			var rtn_body RespBody
			var bdata_loop FileDownloadInfo
			err = proto.Unmarshal(resp_loop.Body, &rtn_body)
			if err != nil {
				return err
			}
			if rtn_body.Code == 200 {
				err = proto.Unmarshal(rtn_body.Data, &bdata_loop)
				if err != nil {
					return err
				}
				f_loop, err := os.OpenFile(filepath.Join(path, fid+"-"+fmt.Sprintf("%d", i)), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
				if err != nil {
					return err
				}
				f_loop.Write(bdata_loop.Data)
				f_loop.Close()
			}
			if i+1 == b_data.BlockTotal {
				completefile := filepath.Join(path, fid)
				cf, err := os.OpenFile(completefile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC|os.O_APPEND, os.ModePerm)
				if err != nil {
					return err
				}
				defer cf.Close()
				for j := 0; j < int(b_data.BlockTotal); j++ {
					path := filepath.Join(path, fid+"-"+fmt.Sprintf("%d", j))
					f, err := os.Open(path)
					if err != nil {
						return err
					}
					defer f.Close()
					temp, err := ioutil.ReadAll(f)
					if err != nil {
						return err
					}
					cf.Write(temp)
				}
				return nil
			}
		}
	}
	return errors.New("receiving file failed, please try again...... ")
}

func ReadFile2(cli *Client, path, fid, walletaddr string) error {
	reqbody := FileDownloadReq{
		FileId:        fid,
		WalletAddress: walletaddr,
		BlockIndex:    0,
	}
	bo, err := proto.Marshal(&reqbody)
	if err != nil {
		return err
	}
	req := &ReqMsg{
		Service: configs.RpcService_Miner,
		Method:  configs.RpcMethod_Miner_ReadFile,
		Body:    bo,
	}

	ctx, _ := context.WithTimeout(context.Background(), 90*time.Second)
	resp, err := cli.Call(ctx, req)
	if err != nil {
		return err
	}

	var b RespBody
	var b_data FileDownloadInfo
	err = proto.Unmarshal(resp.Body, &b)
	if err != nil {
		return err
	}
	if b.Code == 200 {
		err = proto.Unmarshal(b.Data, &b_data)
		if err != nil {
			return err
		}
		if b_data.BlockTotal <= 1 {
			f, err := os.OpenFile(filepath.Join(path, fid), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
			if err != nil {
				return err
			}
			f.Write(b_data.Data)
			f.Close()
			return nil
		}

		f, err := os.OpenFile(filepath.Join(path, fid), os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
		if err != nil {
			return err
		}
		f.Write(b_data.Data)

		for i := int32(1); i < b_data.BlockTotal; i++ {
			reqbody := FileDownloadReq{
				FileId:        fid,
				WalletAddress: walletaddr,
				BlockIndex:    i,
			}
			body_loop, _ := proto.Marshal(&reqbody)
			req := &ReqMsg{
				Service: configs.RpcService_Miner,
				Method:  configs.RpcMethod_Miner_ReadFile,
				Body:    body_loop,
			}
			ctx2, _ := context.WithTimeout(context.Background(), 90*time.Second)
			resp_loop, err := cli.Call(ctx2, req)
			if err != nil {
				f.Close()
				os.Remove(filepath.Join(path, fid))
				return err
			}

			var rtn_body RespBody
			var bdata_loop FileDownloadInfo
			err = proto.Unmarshal(resp_loop.Body, &rtn_body)
			if err != nil {
				f.Close()
				os.Remove(filepath.Join(path, fid))
				return err
			}
			if rtn_body.Code == 200 {
				err = proto.Unmarshal(rtn_body.Data, &bdata_loop)
				if err != nil {
					f.Close()
					os.Remove(filepath.Join(path, fid))
					return err
				}
				f.Write(bdata_loop.Data)
			} else {
				f.Close()
				os.Remove(filepath.Join(path, fid))
				return err
			}
			if i+1 == b_data.BlockTotal {
				// completefile := filepath.Join(path, fid)
				// cf, err := os.OpenFile(completefile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC|os.O_APPEND, os.ModePerm)
				// if err != nil {
				// 	return err
				// }
				// defer cf.Close()
				// for j := 0; j < int(b_data.BlockTotal); j++ {
				// 	path := filepath.Join(path, fid+"-"+fmt.Sprintf("%d", j))
				// 	f, err := os.Open(path)
				// 	if err != nil {
				// 		return err
				// 	}
				// 	defer f.Close()
				// 	temp, err := ioutil.ReadAll(f)
				// 	if err != nil {
				// 		return err
				// 	}
				// 	cf.Write(temp)
				// }
				f.Close()
				return nil
			}
		}
	}
	return errors.New("receiving file failed, please try again...... ")
}

//
func CalcFileBlockSizeAndScanSize(fsize int64) (int64, int64) {
	var (
		blockSize     int64
		scanBlockSize int64
	)
	if fsize < configs.ByteSize_1Kb {
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

// processingfile is used to process all copies of the file and the corresponding tag information
func backupFile(ch chan uint8, fid, fpath, name, userkey string) {
	var (
		err            error
		allMinerPubkey []types.Bytes
	)
	defer func() {
		if err := recover(); err != nil {
			Gpnc.Sugar().Infof("%v", tools.RecoverError(err))
		}
		result := <-ch
		if result == 0 {
			ch <- 1
		}
	}()

	Uld.Sugar().Infof("[%v] Start to store files", fid)

	for len(allMinerPubkey) == 0 {
		allMinerPubkey, _, err = chain.GetAllMinerDataOnChain()
		if err != nil {
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
		}
	}

	Uld.Sugar().Infof("[%v] %v miners found", fid, len(allMinerPubkey))
	// for i := 0; i < len(mDatas); i++ {
	// 	Uld.Sugar().Infof("   %v: %v", i, string(mDatas[i].Ip))
	// }

	fstat, err := os.Stat(fpath)
	if err != nil {
		ch <- 3
		Uld.Sugar().Infof("[%v] The file was deleted and cannot be stored: %v", fid, err)
		return
	}

	f, err := os.OpenFile(fpath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		ch <- 3
		Uld.Sugar().Infof("[%v] OpenFile err: %v", fid, err)
		return
	}

	blockTotal := fstat.Size() / configs.RpcFileBuffer
	if fstat.Size()%configs.RpcFileBuffer != 0 {
		blockTotal += 1
	}
	var client *Client
	var filedIndex = make(map[int]struct{}, 0)
	var mip = ""
	var index int
	var n int
	var minerInfo chain.MinerInfo

	puk, _ := chain.GetPublicKeyByPrk(configs.C.CtrlPrk)

	var bo = PutFileToBucket{}
	bo.BlockTotal = uint32(blockTotal)
	bo.FileId = fid
	bo.Publickey = puk

	for j := int64(0); j < blockTotal; j++ {
		var buf = make([]byte, configs.RpcFileBuffer)
		f.Seek(j*configs.RpcFileBuffer, 0)
		n, _ = f.Read(buf)

		bo.BlockIndex = uint32(j)
		bo.BlockData = buf[:n]

		bob, _ := proto.Marshal(&bo)
		if err != nil {
			ch <- 3
			Uld.Sugar().Infof("[%v] Marshal err: %v", fid, err)
			return
		}
		var failcount uint8

		for {
			if mip == "" {
				if len(filedIndex) >= len(allMinerPubkey) {
					for k, _ := range filedIndex {
						delete(filedIndex, k)
					}
					Uld.Sugar().Infof("[%v] All miners cannot store, refresh the miner list", fid)
					allMinerPubkey, _, err = chain.GetAllMinerDataOnChain()
					if err != nil {
						time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
					}
				}

				index = tools.RandomInRange(0, len(allMinerPubkey))
				if _, ok := filedIndex[index]; ok {
					continue
				}

				minerInfo, _, err = chain.GetMinerInfo(allMinerPubkey[index])
				if err != nil {
					filedIndex[index] = struct{}{}
					Uld.Sugar().Infof("[%v] GetMinerInfo err: %v", fid, err)
					continue
				}

				// if minerInfo.Power.CmpAbs(new(big.Int).SetBytes(minerInfo.Space.Bytes())) < 0 {
				// 	filedIndex[index] = struct{}{}
				// 	Uld.Sugar().Infof("[%v] [%v] [%v] [%v] Abnormal size of space", fid, uint64(minerInfo.PeerId), minerInfo.Power, minerInfo.Space)
				// 	continue
				// }

				var temp = new(big.Int)
				temp.Sub(new(big.Int).SetBytes(minerInfo.Power.Bytes()), new(big.Int).SetBytes(minerInfo.Space.Bytes()))
				if temp.CmpAbs(new(big.Int).SetInt64(fstat.Size())) <= 0 {
					filedIndex[index] = struct{}{}
					Uld.Sugar().Infof("[%v] [%v] Not enough space", fid, fstat.Size())
					continue
				}

				dstip := "ws://" + string(base58.Decode(string(minerInfo.Ip)))
				ctx, _ := context.WithTimeout(context.Background(), 6*time.Second)
				client, err = DialWebsocket(ctx, dstip, "")
				if err != nil {
					filedIndex[index] = struct{}{}
					continue
				}
				Uld.Sugar().Infof("[%v] [%v] connection suc", fid, mip)
				_, err = WriteData2(client, configs.RpcService_Miner, configs.RpcMethod_Miner_WriteFile, bob)
				if err == nil {
					mip = string(minerInfo.Ip)
					Uld.Sugar().Infof("[%v] [%v] transfer suc", fid, j)
					break
				}
				filedIndex[index] = struct{}{}
			} else {
				_, err = WriteData2(client, configs.RpcService_Miner, configs.RpcMethod_Miner_WriteFile, bob)
				if err != nil {
					failcount++
					if failcount >= 5 {
						ch <- 1
						Uld.Sugar().Infof("[%v] [%v] transfer failed: %v", fid, j)
						return
					}
					time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
					continue
				}
				break
			}
		}
	}
	f.Close()

	bs, sbs := CalcFileBlockSizeAndScanSize(fstat.Size())
	blocknum := fstat.Size() / bs
	if n == 0 {
		n = 1
	}

	// calculate file tag info
	var PoDR2commit proof.PoDR2Commit
	var commitResponse proof.PoDR2CommitResponse
	PoDR2commit.FilePath = fpath
	PoDR2commit.BlockSize = bs
	commitResponseCh, err := PoDR2commit.PoDR2ProofCommit(proof.Key_Ssk, string(proof.Key_SharedParams), sbs)
	if err != nil {
		ch <- 1
		Uld.Sugar().Infof("[%v] [%v] PoDR2ProofCommit err: %v", fid, sbs, err)
		return
	}
	select {
	case commitResponse = <-commitResponseCh:
	}
	if commitResponse.StatueMsg.StatusCode != proof.Success {
		ch <- 1
		Uld.Sugar().Infof("[%v] [%v] PoDR2ProofCommit failed", fid, sbs)
		return
	}
	var resp PutTagToBucket
	resp.FileId = fid
	resp.Name = commitResponse.T.Name
	resp.N = commitResponse.T.N
	resp.U = commitResponse.T.U
	resp.Signature = commitResponse.T.Signature
	resp.Sigmas = commitResponse.Sigmas
	resp_proto, err := proto.Marshal(&resp)
	if err != nil {
		ch <- 1
		Uld.Sugar().Infof("[%v] Marshal resp err: %v", fid, err)
		return
	}

	_, err = WriteData2(client, configs.RpcService_Miner, configs.RpcMethod_Miner_WriteFileTag, resp_proto)
	if err != nil {
		ch <- 1
		Uld.Sugar().Infof("[%v] [%v] WriteData2 tag err: %v", fid, mip, err)
		return
	}

	// Upload the file meta information to the chain
	for i := 0; i < 3; i++ {
		ok, err := chain.PutMetaInfoToChain(configs.C.CtrlPrk, fid, fstat.Size(), blocknum, sbs, bs, allMinerPubkey[index], minerInfo.Ip, []byte(userkey))
		if !ok || err != nil {
			if i == 2 {
				Uld.Sugar().Infof("[%v] File meta info on the chain failed: %v", fid, err)
				ch <- 1
				break
			}
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
			continue
		}
		Uld.Sugar().Infof("[%v] File meta info on the chain successfully", fid)
		ch <- 2
		break
	}
}

func genSpaceFile(fpath, content string, rule []byte) error {
	if len(content) != 4096 || len(rule) != 1023 {
		return errors.New("content err")
	}

	f, err := os.OpenFile(fpath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	for i := 0; i < len(rule); i++ {
		f.WriteString(content[rule[i]:])
		f.WriteString(content[:rule[i]])
		f.WriteString("\n")
		f.WriteString(content[rule[i]*rule[i]:])
		f.WriteString(content[:rule[i]*rule[i]])
		f.WriteString("\n")
		if i+1 == len(rule) {
			f.WriteString(content)
			f.WriteString("\n")
			f.WriteString(content[3884:])
		}
	}
	return f.Sync()
}

func upChainSpaceFileMeta(addr, fileid, fpath string, pubkey []byte) (string, error) {
	// up-chain meta info
	var metainfo = make([]chain.SpaceFileInfo, 1)
	metainfo[0].FileId = []byte(fileid)
	fstat, err := os.Stat(fpath)
	if err != nil {
		Spc.Sugar().Infof("[%v] os.Stat [%v] err: %v", addr, fpath, err)
		return "", err
	}

	hash, err := tools.CalcFileHash(fpath)
	if err != nil {
		Spc.Sugar().Infof("[%v] CalcFileHash [%v] err: %v", addr, fpath, err)
		return "", err
	}

	metainfo[0].FileHash = []byte(hash)
	metainfo[0].FileSize = types.U64(uint64(fstat.Size()))
	metainfo[0].Acc = types.NewAccountID(pubkey)

	blocknum := uint64(math.Ceil(float64(fstat.Size() / configs.BlockSize)))
	if blocknum == 0 {
		blocknum = 1
	}
	metainfo[0].BlockNum = types.U32(blocknum)
	metainfo[0].BlockSize = types.U32(uint32(configs.BlockSize))
	metainfo[0].ScanSize = types.U32(uint32(configs.ScanBlockSize))

	var txhash = ""
	var code int
	for i := 0; i < 3; i++ {
		txhash, code, _ = chain.PutSpaceTagInfoToChain(
			configs.C.CtrlPrk,
			pubkey,
			metainfo,
		)
		if txhash != "" || code == configs.Code_200 || code == configs.Code_600 {
			break
		}
		time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
	}
	if txhash != "" {
		Spc.Sugar().Infof("[%v] Uplink file meta failed", addr)
		return "", errors.New("Uplink file meta failed")
	}
	os.Remove(fpath)
	Spc.Sugar().Infof("[%v] Uplink file meta [%v]", addr, txhash)
	return txhash, nil
}
