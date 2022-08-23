package rpc

import (
	"cess-scheduler/configs"
	"cess-scheduler/internal/chain"
	"cess-scheduler/internal/fileHandling"
	. "cess-scheduler/internal/logger"
	"cess-scheduler/internal/pattern"
	apiv1 "cess-scheduler/internal/proof/apiv1"
	"cess-scheduler/tools"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/pkg/errors"

	. "cess-scheduler/internal/rpc/protobuf"

	keyring "github.com/CESSProject/go-keyring"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"google.golang.org/protobuf/proto"
	"storj.io/common/base58"
)

// rpc service and method
const (
	RpcService_Scheduler         = "wservice"
	RpcService_Miner             = "mservice"
	RpcMethod_Miner_WriteFile    = "writefile"
	RpcMethod_Miner_ReadFile     = "readfile"
	RpcMethod_Miner_WriteFileTag = "writefiletag"
	RpcMethod_Miner_ReadFileTag  = "readfiletag"
	RpcFileBuffer                = 1024 * 1024 //1MB
	RpcSpaceBuffer               = 512 * 1024  //512KB
)

const mutexLocked = 1 << iota

type WService struct {
}

type calcTagLock struct {
	flag bool
	lock *sync.Mutex
}

type RespSpaceInfo struct {
	FileId string `json:"fileId"`
	Token  string `json:"token"`
	T      apiv1.FileTagT
	Sigmas [][]byte `json:"sigmas"`
}

type connmap struct {
	lock  *sync.Mutex
	conns map[string]int64
}

var co *connmap
var ctl *calcTagLock

var cacheSt bool = false
var globalTransport *http.Transport

// init
func init() {
	globalTransport = &http.Transport{
		DisableKeepAlives: true,
	}

	co = new(connmap)
	co.lock = new(sync.Mutex)
	co.conns = make(map[string]int64, 10)

	ctl = new(calcTagLock)
	ctl.lock = new(sync.Mutex)
	ctl.flag = false

}

// Start tcp service.
// If an error occurs, it will exit immediately.
func Rpc_Main() {
	// go task_Management()
	srv := NewServer()
	err := srv.Register(RpcService_Scheduler, WService{})
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
	log.Println("Start and listen on port ", configs.C.ServicePort, "...")
	err = http.ListenAndServe(":"+configs.C.ServicePort, srv.WebsocketHandler([]string{"*"}))
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
}

func (this *connmap) UpdateTime(key string) {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.conns[key] = time.Now().Unix()
}

func (this *connmap) DeleteExpired() {
	this.lock.Lock()
	defer this.lock.Unlock()

	for k, v := range this.conns {
		if time.Since(time.Unix(v, 0)).Minutes() > 1 {
			delete(this.conns, k)
		}
	}
}

func (this *connmap) GetConnsNum() int {
	this.lock.Lock()
	defer this.lock.Unlock()
	return len(this.conns)
}

func (this *connmap) IsExist(pubkey string) bool {
	this.lock.Lock()
	defer this.lock.Unlock()
	for k, v := range this.conns {
		addr, _ := tools.EncodeToCESSAddr([]byte(k))
		fmt.Println("k:", addr, "  v:", v)
	}

	_, ok := this.conns[pubkey]
	addr, _ := tools.EncodeToCESSAddr([]byte(pubkey))
	fmt.Println("is: ", ok, "   ", addr)
	return ok
}

func (this *calcTagLock) TryLock() bool {
	return atomic.CompareAndSwapInt32((*int32)(unsafe.Pointer(this.lock)), 0, mutexLocked)
}

func (this *calcTagLock) Lock() {
	this.lock.Lock()
}

func (this *calcTagLock) FreeLock() {
	this.lock.Unlock()
}

// func task_Management() {
// 	var (
// 		channel_1 = make(chan bool, 1)
// 		//channel_2 = make(chan bool, 1)
// 		channel_3 = make(chan bool, 1)
// 		channel_4 = make(chan bool, 1)
// 		channel_5 = make(chan bool, 1)
// 		channel_6 = make(chan bool, 1)
// 	)
// 	go task_SyncMinersInfo(channel_1)
// 	//go task_RecoveryFiles(channel_2)
// 	go task_ValidateProof(channel_3)
// 	go task_ClearAuthMap(channel_4)
// 	go task_SubmitFillerMeta(channel_5)
// 	go task_GenerateFiller(channel_6)
// 	for {
// 		select {
// 		case <-channel_1:
// 			go task_SyncMinersInfo(channel_1)
// 		// case <-channel_2:
// 		// 	go task_RecoveryFiles(channel_2)
// 		case <-channel_3:
// 			go task_ValidateProof(channel_3)
// 		case <-channel_4:
// 			go task_ClearAuthMap(channel_4)
// 		case <-channel_5:
// 			go task_SubmitFillerMeta(channel_5)
// 		case <-channel_6:
// 			go task_GenerateFiller(channel_6)
// 		}
// 	}
// }

// func task_ClearAuthMap(ch chan bool) {
// 	defer func() {
// 		if err := recover(); err != nil {
// 			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
// 		}
// 		ch <- true
// 	}()
// 	var count uint8
// 	for {
// 		count++
// 		if count >= 3 {
// 			count = 0
// 			Com.Info("Connected miners:")
// 			sm.lock.Lock()
// 			for _, v := range sm.miners {
// 				addr, _ := tools.EncodeToCESSAddr([]byte(sm.tokens[v].publicKey))
// 				Com.Sugar().Infof("  %v", addr)
// 			}
// 			sm.lock.Unlock()
// 			Com.Info("Black miners:")
// 			sm.lock.Lock()
// 			for k, v := range sm.blacklist {
// 				addr, _ := tools.EncodeToCESSAddr([]byte(k))
// 				Com.Sugar().Infof("  %v : %v", addr, time.Since(time.Unix(v, 0)).Seconds())
// 			}
// 			sm.lock.Unlock()
// 		}
// 		time.Sleep(time.Minute)
// 		am.DeleteExpired()
// 		sm.DeleteExpired()
// 		co.DeleteExpired()
// 	}
// }

// AuthAction is used to generate credentials.
// The return code is 200 for success, non-200 for failure.
// The returned Msg indicates the result reason.
func (WService) AuthAction(body []byte) (proto.Message, error) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	var b AuthReq
	err := proto.Unmarshal(body, &b)
	if err != nil {
		return &RespBody{Code: 400, Msg: "Bad Requset"}, nil
	}

	if pattern.IsPass(string(b.PublicKey)) {
		return &RespBody{Code: 403, Msg: "Forbidden"}, nil
	}

	if len(b.Msg) == 0 || len(b.Sign) < 64 {
		return &RespBody{Code: 400, Msg: "Invalid Sign"}, nil
	}

	token := pattern.UpdateAuth(string(b.PublicKey))
	if pattern.UpdateAuth(string(b.PublicKey)) != "" {
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
	userSpace, err := chain.GetSpacePackageInfo(types.NewAccountID(b.PublicKey))
	if err != nil {
		Uld.Sugar().Infof("[%v] GetUserSpaceByPuk err: %v", b.FileId, err)
		return &RespBody{Code: 500, Msg: err.Error()}, nil
	}

	if new(big.Int).SetUint64(b.FileSize).CmpAbs(new(big.Int).SetBytes(userSpace.Remaining_space.Bytes())) == 1 {
		return &RespBody{Code: 403, Msg: "Not enough space"}, nil
	}

	//Judge whether the file has been uploaded
	count := 0
	code := configs.Code_404
	fmeta := chain.FileMetaInfo{}
	for {
		if count > 3 {
			Uld.Sugar().Infof("[%v] GetFileMetaInfoOnChain err: %v", b.FileId, err)
			return &RespBody{Code: int32(code), Msg: err.Error()}, nil
		}
		fmeta, err = chain.GetFileMetaInfo(b.FileId)
		if err != nil {
			count++
			time.Sleep(time.Second * 3)
			continue
		}

		if string(fmeta.FileState) == "active" {
			return &RespBody{Code: 201, Msg: "success"}, nil
		}
		break
	}

	var info pattern.Authinfo
	info.PublicKey = string(b.PublicKey)
	info.FileId = b.FileId
	info.FileName = b.FileName
	info.UpdateTime = time.Now().Unix()
	info.BlockTotal = b.BlockTotal
	token = tools.GetRandomcode(12)
	for pattern.IsToken(token) {
		token = tools.GetRandomcode(12)
	}
	pattern.AddAuth(string(b.PublicKey), token, info)
	return &RespBody{Code: 200, Msg: "success", Data: []byte(token)}, nil
}

// WritefileAction is used to handle client requests to upload files.
// The return code is 200 for success, non-200 for failure.
// The returned Msg indicates the result reason.
func (WService) WritefileAction(body []byte) (proto.Message, error) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	var b FileUploadReq
	err := proto.Unmarshal(body, &b)
	if err != nil {
		return &RespBody{Code: 400, Msg: "Bad Requset"}, nil
	}

	if b.BlockIndex == 0 || len(b.FileData) == 0 {
		return &RespBody{Code: 400, Msg: "Invalid parameter"}, nil
	}

	blockTotal, fid, pubkey, fname, err := pattern.GetAndUpdateAuth(string(b.Auth))
	if err != nil {
		return &RespBody{Code: 403, Msg: err.Error()}, nil
	}

	fileAbsPath := filepath.Join(configs.FileCacheDir, fid)

	if b.BlockIndex == 1 {
		Uld.Sugar().Infof("++> Upload file [%v] ", fid)
		_, err = os.Create(fileAbsPath)
		if err != nil {
			Uld.Sugar().Errorf("[%v] Create: %v", fid, err)
			return &RespBody{Code: 500, Msg: err.Error()}, nil
		}
	}

	f, err := os.OpenFile(fileAbsPath, os.O_RDWR|os.O_APPEND, os.ModePerm)
	if err != nil {
		Uld.Sugar().Errorf("[%v] OpenFile: %v", fid, err)
		return &RespBody{Code: 500, Msg: err.Error()}, nil
	}

	_, err = f.Write(b.FileData)
	if err != nil {
		f.Close()
		os.Remove(fileAbsPath)
		Uld.Sugar().Errorf("[%v] Write: %v", fid, err)
		return &RespBody{Code: 500, Msg: err.Error()}, nil
	}

	err = f.Sync()
	if err != nil {
		f.Close()
		os.Remove(fileAbsPath)
		Uld.Sugar().Errorf("[%v] Sync: %v", fid, err)
		return &RespBody{Code: 500, Msg: err.Error()}, nil
	}

	f.Close()

	if b.BlockIndex == blockTotal {
		Uld.Sugar().Infof("[%v] Receive all %v blocks", fid, blockTotal)
		pattern.DeleteAuth(string(b.Auth))
		filehash, err := calcFileHashByChunks(fileAbsPath, configs.SIZE_1GB)
		if err != nil {
			Uld.Sugar().Errorf("[%v] CalcFileHash: %v", fid, err)
			return &RespBody{Code: 500, Msg: err.Error()}, nil
		}
		if filehash != fid[4:] {
			Uld.Sugar().Errorf("[%v] Invalid file hash", fid)
			return &RespBody{Code: 400, Msg: "Invalid file hash"}, nil
		}
		go storeFiles(fid, fileAbsPath, fname, pubkey)
		return &RespBody{Code: 200, Msg: "success"}, nil
	}
	Uld.Sugar().Infof("[%v] %vnd block received", fid, b.BlockIndex)
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
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()
	Uld.Sugar().Infof("[%v] Start the file backup management process", fid)

	fstat, err := os.Stat(fpath)
	if err != nil {
		Uld.Sugar().Errorf("[%v] Stat: %v", fid, err)
		return
	}

	// redundant coding
	chunkspath, datachunks, rduchunks, err := fileHandling.ReedSolomon(fpath, fstat.Size())
	if err != nil {
		Uld.Sugar().Errorf("[%v] ReedSolomon: %v", fid, err)
		return
	}

	Uld.Sugar().Infof("[%v] D: %v  R: %v", fid, datachunks, rduchunks)

	var chunksInfo = make([]chain.ChunkInfo, datachunks+rduchunks)
	var channel_chunks = make(map[int]chan chain.ChunkInfo, datachunks+rduchunks)

	for i := 0; i < len(chunkspath); i++ {
		channel_chunks[i] = make(chan chain.ChunkInfo, 1)
		go backupFile(channel_chunks[i], chunkspath[i], pubkey, i)
	}

	for {
		for k, v := range channel_chunks {
			if len(v) == 1 {
				result := <-v
				if !result.IsEmpty() {
					Uld.Sugar().Infof("[%v.%v] Chunk storage successfully", fid, k)
					chunksInfo[k] = result
					delete(channel_chunks, k)
				} else {
					Uld.Sugar().Warnf("[%v.%v] Try storage again", fid, k)
					go backupFile(channel_chunks[k], chunkspath[k], pubkey, k)
				}
			}
		}
		if len(channel_chunks) == 0 {
			break
		}
	}

	var txhash string
	// Upload the file meta information to the chain
	for {
		txhash, err = chain.PutMetaInfoToChain(configs.C.CtrlPrk, fid, uint64(fstat.Size()), []byte(pubkey), chunksInfo)
		if txhash == "" {
			Uld.Sugar().Errorf("[%v] FileMeta On-chain fail: %v", fid, err)
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 30)))
			continue
		}
		break
	}
	Uld.Sugar().Infof("[%v] FileMeta On-chain [%v]", fid, txhash)
	return
}

// SpacefileAction is used to handle miner requests to download space files.
// The return code is 200 for success, non-200 for failure.
// The returned Msg indicates the result reason.
func (WService) StateAction(body []byte) (proto.Message, error) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()
	l := co.GetConnsNum()
	bb := make([]byte, 4)
	bb[0] = uint8(l >> 24)
	bb[1] = uint8(l >> 26)
	bb[2] = uint8(l >> 8)
	bb[3] = uint8(l)
	return &RespBody{Code: 200, Msg: "success", Data: bb}, nil
}

//
func WriteData(dst string, service, method string, t time.Duration, body []byte) ([]byte, error) {
	dstip := "ws://" + string(base58.Decode(dst))
	dstip = strings.Replace(dstip, " ", "", -1)
	req := &ReqMsg{
		Service: service,
		Method:  method,
		Body:    body,
	}
	ctx1, _ := context.WithTimeout(context.Background(), 5*time.Second)
	client, err := DialWebsocket(ctx1, dstip, "")
	if err != nil {
		return nil, errors.Wrap(err, "DialWebsocket:")
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), t)
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
		Service: RpcService_Miner,
		Method:  RpcMethod_Miner_ReadFile,
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
			Com.Sugar().Errorf("DialWebsocket failed more than 10 times:%v", err)
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
				Service: RpcService_Miner,
				Method:  RpcMethod_Miner_ReadFile,
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
		Service: RpcService_Miner,
		Method:  RpcMethod_Miner_ReadFile,
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
				Service: RpcService_Miner,
				Method:  RpcMethod_Miner_ReadFile,
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
	if fsize < configs.SIZE_1KB {
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
func backupFile(ch chan chain.ChunkInfo, fpath, userkey string, chunkindex int) {
	var (
		err            error
		allMinerPubkey []types.AccountID
	)
	defer func() {
		if len(ch) == 0 {
			ch <- chain.ChunkInfo{}
		}
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()
	fname := filepath.Base(fpath)
	Uld.Sugar().Infof("[%v] Ready to store the chunk", fname)

	for len(allMinerPubkey) == 0 {
		allMinerPubkey, err = chain.GetAllMinerDataOnChain()
		if err != nil {
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
		}
	}

	Uld.Sugar().Infof("[%v] %v miners found", fname, len(allMinerPubkey))

	fstat, err := os.Stat(fpath)
	if err != nil {
		Uld.Sugar().Errorf("[%v] The chunk not found: %v", fname, err)
		return
	}

	f, err := os.OpenFile(fpath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		Uld.Sugar().Errorf("[%v] OpenFile: %v", fname, err)
		return
	}

	blockTotal := fstat.Size() / RpcFileBuffer
	if fstat.Size()%RpcFileBuffer != 0 {
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
	bo.FileId = fname
	bo.Publickey = puk

	for j := int64(0); j < blockTotal; j++ {
		var buf = make([]byte, RpcFileBuffer)
		f.Seek(j*RpcFileBuffer, 0)
		n, _ = f.Read(buf)

		bo.BlockIndex = uint32(j)
		bo.BlockData = buf[:n]

		bob, _ := proto.Marshal(&bo)
		if err != nil {
			Uld.Sugar().Errorf("[%v] Marshal: %v", fname, err)
			return
		}
		var failcount uint8

		for {
			if mip == "" {
				if len(filedIndex) >= len(allMinerPubkey) {
					for k, _ := range filedIndex {
						delete(filedIndex, k)
					}
					Uld.Sugar().Errorf("[%v] All miners cannot store and refresh miner list", fname)
					allMinerPubkey, err = chain.GetAllMinerDataOnChain()
					if err != nil {
						time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
					}
				}

				index = tools.RandomInRange(0, len(allMinerPubkey))
				if _, ok := filedIndex[index]; ok {
					continue
				}

				minerInfo, err = chain.GetMinerInfo(allMinerPubkey[index])
				if err != nil {
					filedIndex[index] = struct{}{}
					Uld.Sugar().Errorf("[%v] GetMinerInfo: %v", fname, err)
					continue
				}

				var temp = new(big.Int)
				temp.Sub(new(big.Int).SetBytes(minerInfo.Power.Bytes()), new(big.Int).SetBytes(minerInfo.Space.Bytes()))
				if temp.CmpAbs(new(big.Int).SetInt64(fstat.Size())) <= 0 {
					filedIndex[index] = struct{}{}
					Uld.Sugar().Errorf("[%v] [%v] Not enough space", fname, fstat.Size())
					continue
				}

				dstip := "ws://" + string(base58.Decode(string(minerInfo.Ip)))
				ctx, _ := context.WithTimeout(context.Background(), 6*time.Second)
				client, err = DialWebsocket(ctx, dstip, "")
				if err != nil {
					filedIndex[index] = struct{}{}
					continue
				}
				Uld.Sugar().Infof("[%v] connected [%v]", fname, string(minerInfo.Ip))
				_, err = WriteData2(client, RpcService_Miner, RpcMethod_Miner_WriteFile, bob)
				if err == nil {
					mip = string(minerInfo.Ip)
					Uld.Sugar().Infof("[%v] transferred [%v-%v]", fname, bo.BlockTotal, bo.BlockIndex)
					break
				}
				filedIndex[index] = struct{}{}
			} else {
				_, err = WriteData2(client, RpcService_Miner, RpcMethod_Miner_WriteFile, bob)
				if err != nil {
					failcount++
					if failcount >= 5 {
						Uld.Sugar().Errorf("[%v] transfer failed [%v-%v]", fname, bo.BlockTotal, bo.BlockIndex)
						return
					}
					time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
					continue
				}
				Uld.Sugar().Infof("[%v] transferred [%v-%v]", fname, bo.BlockTotal, bo.BlockIndex)
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
	Uld.Sugar().Infof("[%v] Calculate tag information", fname)
	// calculate file tag info
	var PoDR2commit apiv1.PoDR2Commit
	var commitResponse apiv1.PoDR2CommitResponse
	PoDR2commit.FilePath = fpath
	PoDR2commit.BlockSize = bs
	commitResponseCh, err := PoDR2commit.PoDR2ProofCommit(apiv1.Key_Ssk, string(apiv1.Key_SharedParams), sbs)
	if err != nil {
		Uld.Sugar().Errorf("[%v] [%v] PoDR2ProofCommit: %v", fname, sbs, err)
		return
	}
	select {
	case commitResponse = <-commitResponseCh:
	}
	if commitResponse.StatueMsg.StatusCode != apiv1.Success {
		Uld.Sugar().Errorf("[%v] [%v] PoDR2ProofCommit failed", fname, sbs)
		return
	}
	var resp PutTagToBucket
	resp.FileId = fname
	resp.Name = commitResponse.T.Name
	resp.N = commitResponse.T.N
	resp.U = commitResponse.T.U
	resp.Signature = commitResponse.T.Signature
	resp.Sigmas = commitResponse.Sigmas
	resp_proto, err := proto.Marshal(&resp)
	if err != nil {
		Uld.Sugar().Errorf("[%v] Marshal: %v", fname, err)
		return
	}

	_, err = WriteData2(client, RpcService_Miner, RpcMethod_Miner_WriteFileTag, resp_proto)
	if err != nil {
		Uld.Sugar().Errorf("[%v] Failed to transfer tag: %v", fname, err)
		return
	}

	Uld.Sugar().Infof("[%v] Transfer tag completed", fname)

	var chunk chain.ChunkInfo
	chunk.ChunkId = types.NewBytes([]byte(fname))
	chunk.ChunkSize = types.U64(fstat.Size())
	chunk.MinerAcc = allMinerPubkey[index]
	chunk.MinerIp = types.NewBytes([]byte(mip))
	chunk.MinerId = minerInfo.PeerId
	chunk.BlockNum = types.U32(blocknum)
	ch <- chunk
}

func combineFillerMeta(addr, fileid, fpath string, pubkey []byte) (chain.SpaceFileInfo, error) {
	var metainfo chain.SpaceFileInfo
	metainfo.FileId = []byte(fileid)
	metainfo.Index = 0
	fstat, err := os.Stat(fpath)
	if err != nil {
		Flr.Sugar().Errorf("[%v] Stat: %v", addr, err)
		return metainfo, err
	}

	hash, err := tools.CalcFileHash(fpath)
	if err != nil {
		Flr.Sugar().Errorf("[%v] CalcFileHash: %v", addr, err)
		return metainfo, err
	}

	metainfo.FileHash = []byte(hash)
	metainfo.FileSize = 8388608
	metainfo.Acc = types.NewAccountID(pubkey)

	blocknum := uint64(math.Ceil(float64(fstat.Size() / configs.BlockSize)))
	if blocknum == 0 {
		blocknum = 1
	}
	metainfo.BlockNum = types.U32(blocknum)
	metainfo.BlockSize = types.U32(uint32(configs.BlockSize))
	metainfo.ScanSize = types.U32(uint32(configs.ScanBlockSize))

	return metainfo, nil
}
