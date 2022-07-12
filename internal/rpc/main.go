package rpc

import (
	"bytes"
	"cess-scheduler/configs"
	"cess-scheduler/internal/chain"
	"cess-scheduler/internal/fileHandling"
	. "cess-scheduler/internal/logger"
	proof "cess-scheduler/internal/proof/apiv1"
	"cess-scheduler/tools"
	"context"
	"encoding/binary"
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

type RespSpaceInfo struct {
	FileId string `json:"fileId"`
	Token  string `json:"token"`
	T      proof.FileTagT
	Sigmas [][]byte `json:"sigmas"`
}

type authinfo struct {
	publicKey  string
	fileId     string
	fileName   string
	updateTime int64
	blockTotal uint32
}

type authspaceinfo struct {
	publicKey  string
	fid        string
	updateTime int64
}

type authmap struct {
	lock   *sync.Mutex
	users  map[string]string
	tokens map[string]authinfo
}

type spacemap struct {
	lock   *sync.Mutex
	miners map[string]string
	tokens map[string]authspaceinfo
}

type connmap struct {
	lock  *sync.Mutex
	conns map[string]int64
}

var am *authmap
var sm *spacemap
var co *connmap
var chan_FillerMeta chan chain.SpaceFileInfo

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

func (this *spacemap) UpdateTimeIfExists(key, fname string) (string, int, error) {
	this.lock.Lock()
	defer this.lock.Unlock()

	v, ok := this.miners[key]
	if ok {
		info := this.tokens[v]
		if time.Since(time.Unix(info.updateTime, 0)).Seconds() < 10 {
			return v, configs.Code_403, errors.New("Requests too frequently")
		}
		info.fid = fname
		info.updateTime = time.Now().Unix()
		this.tokens[v] = info
		return v, configs.Code_200, nil
	}
	token := tools.RandStr(16)
	data := authspaceinfo{}
	data.publicKey = key
	data.fid = fname
	data.updateTime = time.Now().Unix()
	this.miners[key] = token
	this.tokens[token] = data
	return token, configs.Code_200, nil
}

func (this *spacemap) VerifyToken(token string) (string, string, error) {
	this.lock.Lock()
	defer this.lock.Unlock()
	v, ok := this.tokens[token]
	if !ok {
		return "", "", errors.New("Invalid token")
	}
	return v.publicKey, v.fid, nil
}

func (this *spacemap) Delete(key string) {
	this.lock.Lock()
	defer this.lock.Unlock()

	v, ok := this.tokens[key]
	if ok {
		delete(this.miners, v.publicKey)
		delete(this.tokens, key)
	}
}

func (this *spacemap) DeleteExpired() {
	this.lock.Lock()
	defer this.lock.Unlock()

	for k, v := range this.tokens {
		if time.Since(time.Unix(v.updateTime, 0)).Minutes() > 2 {
			delete(this.miners, v.publicKey)
			delete(this.tokens, k)
		}
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

func init() {
	am = new(authmap)
	am.lock = new(sync.Mutex)
	am.users = make(map[string]string, 10)
	am.tokens = make(map[string]authinfo, 10)
	sm = new(spacemap)
	sm.lock = new(sync.Mutex)
	sm.miners = make(map[string]string, 10)
	sm.tokens = make(map[string]authspaceinfo, 10)
	co = new(connmap)
	co.lock = new(sync.Mutex)
	co.conns = make(map[string]int64, 10)
	chan_FillerMeta = make(chan chain.SpaceFileInfo, 20)
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
		channel_2 = make(chan bool, 1)
	)
	go task_ClearAuthMap(channel_1)
	go task_SubmitFillerMeta(channel_2)
	for {
		select {
		case <-channel_1:
			go task_ClearAuthMap(channel_1)
		case <-channel_2:
			go task_SubmitFillerMeta(channel_2)
		}
	}
}

func task_ClearAuthMap(ch chan bool) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
		ch <- true
	}()

	for {
		time.Sleep(time.Minute * time.Duration(tools.RandomInRange(2, 5)))
		am.DeleteExpired()
		sm.DeleteExpired()
		co.DeleteExpired()
	}
}

// AuthAction is used to generate credentials.
// The return code is 200 for success, non-200 for failure.
// The returned Msg indicates the result reason.
func (WService) AuthAction(body []byte) (proto.Message, error) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
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
			if new(big.Int).SetUint64(b.FileSize).CmpAbs(new(big.Int).SetBytes(userSpace.RemainingSpace.Bytes())) == 1 {
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
		fmeta, code, err = chain.GetFileMetaInfo(b.FileId)
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
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
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

	if b.BlockIndex == 0 || len(b.FileData) == 0 {
		return &RespBody{Code: 400, Msg: "Invalid parameter"}, nil
	}

	blockTotal, fid, pubkey, fname, err := am.UpdateTime(string(b.Auth))
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
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	}

	f.Close()

	if b.BlockIndex == blockTotal {
		Uld.Sugar().Infof("[%v] Receive all %v blocks", fid, blockTotal)
		am.Delete(string(b.Auth))
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
	co.UpdateTime(pubkey)
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
		if len(channel_chunks) == 0 {
			break
		}
	}

	var txhash string
	// Upload the file meta information to the chain
	for txhash == "" {
		txhash, err = chain.PutMetaInfoToChain(configs.C.CtrlPrk, fid, uint64(fstat.Size()), []byte(pubkey), chunksInfo)
		if err != nil {
			Uld.Sugar().Errorf("[%v] FileMeta On-chain fail: %v", fid, err)
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 30)))
			continue
		}
		break
	}
	Uld.Sugar().Infof("[%v] FileMeta On-chain [%v]", fid, txhash)
	return
}

// SpaceAction is used to handle miner requests to space files.
// The return code is 200 for success, non-200 for failure.
// The returned Msg indicates the result reason.
func (WService) SpaceAction(body []byte) (proto.Message, error) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	if len(sm.miners) > 2000 {
		return &RespBody{Code: 503, Msg: "Server is busy"}, nil
	}

	var b SpaceReq
	err := proto.Unmarshal(body, &b)
	if err != nil {
		return &RespBody{Code: 400, Msg: "Bad Request"}, nil
	}

	if len(b.Publickey) == 0 || len(b.Msg) == 0 || len(b.Sign) == 0 {
		return &RespBody{Code: 400, Msg: "Invalid parameter"}, nil
	}

	addr, err := tools.EncodeToCESSAddr(b.Publickey)
	if err != nil {
		return &RespBody{Code: 400, Msg: "Bad publickey"}, nil
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
	ok := verkr.Verify(verkr.SigningContext(b.Msg), sign)
	if !ok {
		return &RespBody{Code: 403, Msg: "Authentication failed"}, nil
	}

	//Flr.Sugar().Infof("[%v] Filler tag", addr)

	filebasedir := filepath.Join(configs.SpaceCacheDir, addr)
	_, err = os.Stat(filebasedir)
	if err != nil {
		err = os.MkdirAll(filebasedir, os.ModeDir)
		if err != nil {
			Flr.Sugar().Infof("[%v] mkdir [%v] err: %v", addr, filebasedir, err)
			return &RespBody{Code: 500, Msg: err.Error()}, nil
		}
	}

	filename := fmt.Sprintf("%d", time.Now().Unix())
	filefullpath := filepath.Join(filebasedir, filename)

	//Prohibit frequent requests
	token, code, err := sm.UpdateTimeIfExists(string(b.Publickey), filename)
	if err != nil {
		Flr.Sugar().Errorf("[%v] UpdateTimeIfExists err: %v", addr, err)
		return &RespBody{Code: int32(code), Msg: err.Error()}, nil
	}

	//Generate space file
	err = genSpaceFile(filefullpath)
	if err != nil {
		Flr.Sugar().Errorf("[%v] genSpaceFile err: %v", addr, err)
		os.Remove(filefullpath)
		return &RespBody{Code: 500, Msg: err.Error()}, nil
	}

	fstat, _ := os.Stat(filefullpath)
	if fstat.Size() != 8386771 {
		Flr.Sugar().Errorf("[%v] file size err: %v", addr, err)
		os.Remove(filefullpath)
		return &RespBody{Code: 500, Msg: "file size err"}, nil
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
				Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
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
		Flr.Sugar().Errorf("[%v] PoDR2ProofCommit false", addr)
		return &RespBody{Code: 500, Msg: "unexpected system error"}, nil
	}

	var resp RespSpaceInfo
	resp.FileId = filename
	resp.Token = token
	resp.T = commitResponse.T
	resp.Sigmas = commitResponse.Sigmas
	resp_b, err := json.Marshal(resp)
	if err != nil {
		Flr.Sugar().Errorf("[%v] Marshal err: %v", addr, err)
		return &RespBody{Code: 500, Msg: err.Error()}, nil
	}
	Flr.Sugar().Infof("[%v] Generate filler: %v", addr, filename)
	return &RespBody{Code: 200, Msg: "success", Data: resp_b}, nil
}

// SpacefileAction is used to handle miner requests to download space files.
// The return code is 200 for success, non-200 for failure.
// The returned Msg indicates the result reason.
func (WService) SpacefileAction(body []byte) (proto.Message, error) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	if len(sm.miners) > 2000 {
		return &RespBody{Code: 503, Msg: "Server is busy"}, nil
	}

	var b SpaceFileReq
	err := proto.Unmarshal(body, &b)
	if err != nil {
		return &RespBody{Code: 400, Msg: "Bad Request"}, nil
	}

	if b.BlockIndex > 16 {
		return &RespBody{Code: 400, Msg: "Invalid blocknum"}, nil
	}

	if b.Token == "" {
		return &RespBody{Code: 400, Msg: "Empty token"}, nil
	}

	pubkey, fname, err := sm.VerifyToken(b.Token)
	if err != nil {
		return &RespBody{Code: 403, Msg: err.Error()}, nil
	}

	co.UpdateTime(pubkey)

	addr, err := tools.EncodeToCESSAddr([]byte(pubkey))
	if err != nil {
		return &RespBody{Code: 400, Msg: "Bad publickey"}, nil
	}

	filefullpath := filepath.Join(configs.SpaceCacheDir, addr, fname)
	if b.BlockIndex == 16 {
		sm.Delete(b.Token)
		Flr.Sugar().Infof("[%v] Transferred filler: %v", addr, fname)
		var data chain.SpaceFileInfo
		data, err = CombineFillerMeta(addr, fname, filefullpath, []byte(pubkey))
		if err != nil {
			return &RespBody{Code: 500, Msg: err.Error()}, nil
		}
		chan_FillerMeta <- data
		os.Remove(filefullpath)
		return &RespBody{Code: 200, Msg: "success"}, nil
	}

	f, err := os.OpenFile(filefullpath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return &RespBody{Code: 500, Msg: err.Error()}, nil
	}
	defer f.Close()
	var n = 0
	var buf = make([]byte, configs.RpcSpaceBuffer)
	f.Seek(int64(b.BlockIndex)*configs.RpcSpaceBuffer, 0)
	n, _ = f.Read(buf)
	co.conns[pubkey] = time.Now().Unix()
	return &RespBody{Code: 200, Msg: "success", Data: buf[:n]}, nil
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
	l := len(co.conns)
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, &l)
	return &RespBody{Code: 200, Msg: "success", Data: bytesBuffer.Bytes()}, nil
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
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()
	fname := filepath.Base(fpath)
	Uld.Sugar().Infof("[%v] Ready to store the chunk", fname)

	for len(allMinerPubkey) == 0 {
		allMinerPubkey, _, err = chain.GetAllMinerDataOnChain()
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
	bo.FileId = fname
	bo.Publickey = puk

	for j := int64(0); j < blockTotal; j++ {
		var buf = make([]byte, configs.RpcFileBuffer)
		f.Seek(j*configs.RpcFileBuffer, 0)
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
				_, err = WriteData2(client, configs.RpcService_Miner, configs.RpcMethod_Miner_WriteFile, bob)
				if err == nil {
					mip = string(minerInfo.Ip)
					Uld.Sugar().Infof("[%v] transferred [%v-%v]", fname, bo.BlockTotal, bo.BlockIndex)
					break
				}
				filedIndex[index] = struct{}{}
			} else {
				_, err = WriteData2(client, configs.RpcService_Miner, configs.RpcMethod_Miner_WriteFile, bob)
				if err != nil {
					failcount++
					if failcount >= 5 {
						Uld.Sugar().Errorf("[%v] [%v] transfer failed [%v-%v]", fname, bo.BlockTotal, bo.BlockIndex)
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
	var PoDR2commit proof.PoDR2Commit
	var commitResponse proof.PoDR2CommitResponse
	PoDR2commit.FilePath = fpath
	PoDR2commit.BlockSize = bs
	commitResponseCh, err := PoDR2commit.PoDR2ProofCommit(proof.Key_Ssk, string(proof.Key_SharedParams), sbs)
	if err != nil {
		Uld.Sugar().Errorf("[%v] [%v] PoDR2ProofCommit: %v", fname, sbs, err)
		return
	}
	select {
	case commitResponse = <-commitResponseCh:
	}
	if commitResponse.StatueMsg.StatusCode != proof.Success {
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

	_, err = WriteData2(client, configs.RpcService_Miner, configs.RpcMethod_Miner_WriteFileTag, resp_proto)
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

func genSpaceFile(fpath string) error {
	f, err := os.OpenFile(fpath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	for i := 0; i < 2047; i++ {
		f.WriteString(tools.RandStr(4096) + "\n")
	}
	f.WriteString(tools.RandStr(212))
	return f.Sync()
}

func task_SubmitFillerMeta(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	Tsfm.Info("-----> Start task_SubmitFillerMeta")

	var (
		err         error
		txhash      string
		fillermetas = make(map[types.AccountID][]chain.SpaceFileInfo, 10)
	)
	t_active := time.Now()
	t_inactive := time.Now()
	for {
		time.Sleep(time.Second)
		if len(chan_FillerMeta) > 0 {
			tmp := <-chan_FillerMeta
			_, ok := fillermetas[tmp.Acc]
			if !ok {
				fillermetas[tmp.Acc] = make([]chain.SpaceFileInfo, 0)
			}
			fillermetas[tmp.Acc] = append(fillermetas[tmp.Acc], tmp)
		}
		if time.Since(t_active).Minutes() > 1 {
			t_active = time.Now()
			for k, v := range fillermetas {
				if len(v) > 8 {
					txhash, err = chain.PutSpaceTagInfoToChain(configs.C.CtrlPrk, k, v[:8])
					if txhash == "" || err != nil {
						Tsfm.Sugar().Errorf("%v", err)
						continue
					}
					fillermetas[k] = v[8:]
					miner_addr, _ := tools.EncodeToCESSAddr(k[:])
					Tsfm.Sugar().Infof("[%v] %v", miner_addr, txhash)
				}
			}
		}
		if time.Since(t_inactive).Minutes() > 30 {
			t_inactive = time.Now()
			for k, v := range fillermetas {
				_, ok := co.conns[string(k[:])]
				if !ok && len(v) > 0 {
					txhash, err = chain.PutSpaceTagInfoToChain(configs.C.CtrlPrk, k, v[:])
					if txhash == "" || err != nil {
						Tsfm.Sugar().Errorf("%v", err)
						continue
					}
					delete(fillermetas, k)
					miner_addr, _ := tools.EncodeToCESSAddr(k[:])
					Tsfm.Sugar().Infof("[%v] %v", miner_addr, txhash)
				}
			}
		}
	}
}

func CombineFillerMeta(addr, fileid, fpath string, pubkey []byte) (chain.SpaceFileInfo, error) {
	var metainfo chain.SpaceFileInfo
	metainfo.FileId = []byte(fileid)
	metainfo.Index = 0
	fstat, err := os.Stat(fpath)
	if err != nil {
		Flr.Sugar().Errorf("[%v] os.Stat [%v] err: %v", addr, fpath, err)
		return metainfo, err
	}

	hash, err := tools.CalcFileHash(fpath)
	if err != nil {
		Flr.Sugar().Errorf("[%v] CalcFileHash [%v] err: %v", addr, fpath, err)
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
