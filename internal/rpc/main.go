package rpc

import (
	"cess-scheduler/configs"
	"cess-scheduler/internal/chain"
	"cess-scheduler/internal/db"
	"cess-scheduler/internal/fileHandling"
	. "cess-scheduler/internal/logger"
	apiv1 "cess-scheduler/internal/proof/apiv1"
	"cess-scheduler/tools"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
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

type authinfo struct {
	publicKey  string
	fileId     string
	fileName   string
	updateTime int64
	blockTotal uint32
}

type authspaceinfo struct {
	publicKey  string
	ip         string
	fillerId   string
	updateTime int64
}

type fillermetamap struct {
	lock        *sync.Mutex
	fillermetas map[string][]chain.SpaceFileInfo
}

type authmap struct {
	lock   *sync.Mutex
	users  map[string]string
	tokens map[string]authinfo
}

type spacemap struct {
	lock      *sync.Mutex
	miners    map[string]string
	blacklist map[string]int64
	tokens    map[string]authspaceinfo
}

type connmap struct {
	lock  *sync.Mutex
	conns map[string]int64
}

type filler struct {
	FillerId string
	Path     string
	T        apiv1.FileTagT
	Sigmas   [][]byte `json:"sigmas"`
}

type baseFiller struct {
	MinerIp  []string `json:"minerIp"`
	FillerId string   `json:"fillerId"`
}

type baseFillerList struct {
	Lock       *sync.Mutex
	BaseFiller []baseFiller
}

var am *authmap
var sm *spacemap
var co *connmap
var ctl *calcTagLock
var fm *fillermetamap
var chan_FillerMeta chan chain.SpaceFileInfo
var chan_Filler chan filler
var bs *baseFillerList
var cacheSt bool
var globalTransport *http.Transport

// init
func init() {
	globalTransport = &http.Transport{
		DisableKeepAlives: true,
	}

	am = new(authmap)
	am.lock = new(sync.Mutex)
	am.users = make(map[string]string, 10)
	am.tokens = make(map[string]authinfo, 10)

	sm = new(spacemap)
	sm.lock = new(sync.Mutex)
	sm.miners = make(map[string]string, 10)
	sm.blacklist = make(map[string]int64, 10)
	sm.tokens = make(map[string]authspaceinfo, 10)

	co = new(connmap)
	co.lock = new(sync.Mutex)
	co.conns = make(map[string]int64, 10)

	ctl = new(calcTagLock)
	ctl.lock = new(sync.Mutex)
	ctl.flag = false

	fm = new(fillermetamap)
	fm.fillermetas = make(map[string][]chain.SpaceFileInfo)
	fm.lock = new(sync.Mutex)

	chan_FillerMeta = make(chan chain.SpaceFileInfo, 100)
	chan_Filler = make(chan filler, 10)

	bs = new(baseFillerList)
	bs.Lock = new(sync.Mutex)
	bs.BaseFiller = make([]baseFiller, 0)
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
	log.Println("Start and listen on port ", configs.C.ServicePort, "...")
	err = http.ListenAndServe(":"+configs.C.ServicePort, srv.WebsocketHandler([]string{"*"}))
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
}

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

func (this *spacemap) UpdateTimeIfExists(key, ip, fid string) string {
	this.lock.Lock()
	defer this.lock.Unlock()

	v, _ := this.miners[key]
	info, ok2 := this.tokens[v]
	if ok2 {
		info.updateTime = time.Now().Unix()
		info.fillerId = fid
		this.tokens[v] = info
		return v
	}
	token := tools.RandStr(16)
	data := authspaceinfo{}
	data.publicKey = key
	data.ip = ip
	data.fillerId = fid
	data.updateTime = time.Now().Unix()
	this.miners[key] = token
	this.tokens[token] = data
	return token
}

func (this *spacemap) VerifyToken(token string) (string, string, string, error) {
	this.lock.Lock()
	defer this.lock.Unlock()
	v, ok := this.tokens[token]
	if !ok {
		return "", "", "", errors.New("Invalid token")
	}
	return v.publicKey, v.fillerId, v.ip, nil
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
		if time.Since(time.Unix(v.updateTime, 0)).Minutes() > 5 {
			delete(this.miners, v.publicKey)
			delete(this.tokens, k)
		}
	}

	for k, v := range this.miners {
		t, err := strconv.ParseInt(v, 10, 64)
		if err == nil {
			if time.Since(time.Unix(t, 0)).Minutes() > 5 {
				delete(this.miners, k)
			}
		}
	}
}

func (this *spacemap) GetConnsMinerNum() int {
	this.lock.Lock()
	defer this.lock.Unlock()
	return len(this.miners)
}

func (this *spacemap) IsExit(pubkey string) bool {
	this.lock.Lock()
	defer this.lock.Unlock()
	_, ok := this.miners[pubkey]
	return ok
}

func (this *spacemap) FilterBlacklist(pubkey string) bool {
	this.lock.Lock()
	defer this.lock.Unlock()

	v, ok := this.blacklist[pubkey]
	if ok {
		if time.Since(time.Unix(v, 0)).Minutes() > 10 {
			delete(this.blacklist, pubkey)
			return false
		}
	}
	return ok
}

func (this *spacemap) Connect(pubkey string) bool {
	this.lock.Lock()
	defer this.lock.Unlock()
	v, ok := this.miners[pubkey]
	if !ok {
		if len(this.miners) < 10 {
			this.miners[pubkey] = fmt.Sprintf("%v", time.Now().Unix())
			return true
		} else {
			return false
		}
	}

	info, ok := this.tokens[v]
	if ok {
		if time.Since(time.Unix(info.updateTime, 0)).Seconds() < 5 {
			delete(this.miners, pubkey)
			delete(this.tokens, v)
			this.blacklist[pubkey] = time.Now().Unix()
			return false
		}

		info.updateTime = time.Now().Unix()
		this.tokens[v] = info
	}
	return true
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
			addr, _ := tools.EncodeToCESSAddr([]byte(k))
			fmt.Println("delete: ", addr)
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

func (this *fillermetamap) Add(pubkey string, data chain.SpaceFileInfo) {
	this.lock.Lock()
	defer this.lock.Unlock()
	_, ok := this.fillermetas[pubkey]
	if !ok {
		this.fillermetas[pubkey] = make([]chain.SpaceFileInfo, 0)
	}
	this.fillermetas[pubkey] = append(this.fillermetas[pubkey], data)
}

func (this *fillermetamap) GetNum(pubkey string) int {
	this.lock.Lock()
	defer this.lock.Unlock()

	return len(this.fillermetas[pubkey])
}

func (this *fillermetamap) Delete(pubkey string) {
	this.lock.Lock()
	defer this.lock.Unlock()
	delete(this.fillermetas, pubkey)
}

func (this *fillermetamap) Lock() {
	this.lock.Lock()
}

func (this *fillermetamap) UnLock() {
	this.lock.Unlock()
}

func (this *baseFillerList) Add(elem baseFiller) {
	this.Lock.Lock()
	defer this.Lock.Unlock()
	this.BaseFiller = append(this.BaseFiller, elem)
}

func (this *baseFillerList) GetTheFirst() baseFiller {
	this.Lock.Lock()
	defer this.Lock.Unlock()
	return this.BaseFiller[0]
}

func (this *baseFillerList) GetAvailableAndInsert(ip string) (baseFiller, error) {
	this.Lock.Lock()
	defer this.Lock.Unlock()
	exist := false
	if len(this.BaseFiller) > 100 {
		this.BaseFiller = this.BaseFiller[1:]
	}
	for k1, v1 := range this.BaseFiller {
		exist = false
		for _, v2 := range v1.MinerIp {
			if v2 == ip {
				exist = true
				break
			}
		}
		if !exist && len(v1.MinerIp) < 3 {
			this.BaseFiller[k1].MinerIp = append(this.BaseFiller[k1].MinerIp, ip)
			return v1, nil
		}
	}
	return baseFiller{}, errors.New("None available")
}

func (this *baseFillerList) RemoveTheFirst() {
	this.Lock.Lock()
	defer this.Lock.Unlock()
	this.BaseFiller = this.BaseFiller[1:]
}

func (this *baseFillerList) GetLength() int {
	this.Lock.Lock()
	defer this.Lock.Unlock()
	return len(this.BaseFiller)
}

func (this *baseFillerList) AddIpInFirst(ip string) {
	this.Lock.Lock()
	defer this.Lock.Unlock()
	this.BaseFiller[0].MinerIp = append(this.BaseFiller[0].MinerIp, ip)
	return
}

func (this *baseFillerList) InsertIpInKey(key int, ip string) {
	this.Lock.Lock()
	defer this.Lock.Unlock()
	this.BaseFiller[key].MinerIp = append(this.BaseFiller[key].MinerIp, ip)
	return
}

func task_Management() {
	var (
		channel_1 = make(chan bool, 1)
		//channel_2 = make(chan bool, 1)
		channel_3 = make(chan bool, 1)
		channel_4 = make(chan bool, 1)
		channel_5 = make(chan bool, 1)
		channel_6 = make(chan bool, 1)
	)
	go task_SyncMinersInfo(channel_1)
	//go task_RecoveryFiles(channel_2)
	go task_ValidateProof(channel_3)
	go task_ClearAuthMap(channel_4)
	go task_SubmitFillerMeta(channel_5)
	go task_GenerateFiller(channel_6)
	for {
		select {
		case <-channel_1:
			go task_SyncMinersInfo(channel_1)
		// case <-channel_2:
		// 	go task_RecoveryFiles(channel_2)
		case <-channel_3:
			go task_ValidateProof(channel_3)
		case <-channel_4:
			go task_ClearAuthMap(channel_4)
		case <-channel_5:
			go task_SubmitFillerMeta(channel_5)
		case <-channel_6:
			go task_GenerateFiller(channel_6)
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
		time.Sleep(time.Minute)
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
	userSpace := chain.SpacePackage{}
	for code != configs.Code_200 {
		userSpace, code, err = chain.GetSpacePackageInfo(types.NewAccountID(b.PublicKey))
		if count > 3 && code != configs.Code_200 {
			Uld.Sugar().Infof("[%v] GetUserSpaceByPuk err: %v", b.FileId, err)
			return &RespBody{Code: int32(code), Msg: err.Error()}, nil
		}
		if code != configs.Code_200 {
			time.Sleep(time.Second * 3)
		} else {
			if new(big.Int).SetUint64(b.FileSize).CmpAbs(new(big.Int).SetBytes(userSpace.Remaining_space.Bytes())) == 1 {
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
	if !cacheSt {
		time.Sleep(time.Second * time.Duration(tools.RandomInRange(2, 5)))
		return &RespBody{Code: http.StatusServiceUnavailable, Msg: "ServiceUnavailable"}, nil
	}

	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	var b SpaceReq
	err := proto.Unmarshal(body, &b)
	if err != nil {
		return &RespBody{Code: http.StatusForbidden, Msg: "Bad request"}, nil
	}

	if sm.FilterBlacklist(string(b.Publickey)) {
		time.Sleep(time.Second * time.Duration(tools.RandomInRange(30, 60)))
		addr, _ := tools.EncodeToCESSAddr(b.Publickey)
		fpath := filepath.Join(configs.SpaceCacheDir, addr)
		os.RemoveAll(fpath)
		return &RespBody{Code: http.StatusForbidden, Msg: "Forbidden"}, nil
	}

	ok := sm.Connect(string(b.Publickey))
	if !ok {
		time.Sleep(time.Second * time.Duration(tools.RandomInRange(10, 30)))
		return &RespBody{Code: 503, Msg: "Server is busy"}, nil
	}

	addr, err := tools.EncodeToCESSAddr(b.Publickey)
	if err != nil {
		return &RespBody{Code: http.StatusForbidden, Msg: "Invalid public key"}, nil
	}

	//Verify miner identity
	ss58addr, err := tools.EncodeToSS58(b.Publickey)
	if err != nil {
		return &RespBody{Code: http.StatusForbidden, Msg: "Invalid public key"}, nil
	}
	verkr, _ := keyring.FromURI(ss58addr, keyring.NetSubstrate{})

	if len(b.Sign) < 64 {
		return &RespBody{Code: http.StatusForbidden, Msg: "Authentication failed"}, nil
	}

	var sign [64]byte
	for i := 0; i < 64; i++ {
		sign[i] = b.Sign[i]
	}

	ok = verkr.Verify(verkr.SigningContext(b.Msg), sign)
	if !ok {
		time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 30)))
		return &RespBody{Code: http.StatusForbidden, Msg: "Authentication failed"}, nil
	}

	c, err := db.GetCache()
	if err != nil {
		return &RespBody{Code: http.StatusInternalServerError, Msg: "InternalServerError"}, nil
	}
	minercache, err := c.Get(b.Publickey)
	if err != nil {
		return &RespBody{Code: http.StatusNotFound, Msg: "Not found"}, nil
	}

	var minerinfo chain.Cache_MinerInfo

	err = json.Unmarshal(minercache, &minerinfo)
	if err != nil {
		return &RespBody{Code: http.StatusInternalServerError, Msg: "Cache error"}, nil
	}

	if bs.GetLength() == 0 {
		if len(chan_Filler) == 0 {
			time.Sleep(time.Second)
			if len(chan_Filler) == 0 {
				return &RespBody{Code: http.StatusServiceUnavailable, Msg: "ServiceUnavailable"}, nil
			}
		}

		filler := <-chan_Filler
		var resp RespSpaceInfo
		resp.Token = sm.UpdateTimeIfExists(string(b.Publickey), minerinfo.Ip, filler.FillerId)
		resp.FileId = filler.FillerId
		resp.T = filler.T
		resp.Sigmas = filler.Sigmas
		resp_b, err := json.Marshal(resp)
		if err != nil {
			os.Remove(filler.Path)
			Flr.Sugar().Errorf("[%v] Marshal: %v", addr, err)
			return &RespBody{Code: http.StatusInternalServerError, Msg: err.Error()}, nil
		}

		Flr.Sugar().Infof("[%v] Base filler: %v", addr, filler.FillerId)
		return &RespBody{Code: 200, Msg: "success", Data: resp_b}, nil
	}

	basefiller, err := bs.GetAvailableAndInsert(minerinfo.Ip)
	if err != nil {
		if len(chan_Filler) == 0 {
			time.Sleep(time.Second)
			if len(chan_Filler) == 0 {
				return &RespBody{Code: http.StatusServiceUnavailable, Msg: "ServiceUnavailable"}, nil
			}
		}
		filler := <-chan_Filler
		var resp RespSpaceInfo
		resp.Token = sm.UpdateTimeIfExists(string(b.Publickey), minerinfo.Ip, filler.FillerId)
		resp.FileId = filler.FillerId
		resp.T = filler.T
		resp.Sigmas = filler.Sigmas
		resp_b, err := json.Marshal(resp)
		if err != nil {
			os.Remove(filler.Path)
			Flr.Sugar().Errorf("[%v] Marshal: %v", addr, err)
			return &RespBody{Code: http.StatusInternalServerError, Msg: err.Error()}, nil
		}

		Flr.Sugar().Infof("[%v] Base filler: %v", addr, filler.FillerId)
		return &RespBody{Code: 200, Msg: "success", Data: resp_b}, nil
	}

	resp_b, err := json.Marshal(basefiller)
	if err != nil {
		Flr.Sugar().Errorf("[%v] Marshal: %v", addr, err)
		return &RespBody{Code: http.StatusInternalServerError, Msg: err.Error()}, nil
	}
	time.Sleep(time.Second * 5)
	return &RespBody{Code: 201, Msg: "success", Data: resp_b}, nil
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

	pubkey, fname, ip, err := sm.VerifyToken(b.Token)
	if err != nil {
		return &RespBody{Code: 403, Msg: err.Error()}, nil
	}

	sm.UpdateTimeIfExists(pubkey, ip, fname)
	co.UpdateTime(pubkey)

	addr, err := tools.EncodeToCESSAddr([]byte(pubkey))
	if err != nil {
		return &RespBody{Code: 400, Msg: "Bad publickey"}, nil
	}

	filefullpath := filepath.Join(configs.SpaceCacheDir, fname)
	if b.BlockIndex == 16 {
		Flr.Sugar().Infof("[%v] Transferred filler: %v", addr, fname)
		var data chain.SpaceFileInfo
		data, err = CombineFillerMeta(addr, fname, filefullpath, []byte(pubkey))
		if err != nil {
			os.Remove(filefullpath)
			return &RespBody{Code: 500, Msg: err.Error()}, nil
		}
		chan_FillerMeta <- data
		os.Remove(filefullpath)
		var bf baseFiller
		bf.FillerId = fname
		bf.MinerIp = make([]string, 0)
		bf.MinerIp = append(bf.MinerIp, ip)
		bs.Add(bf)
		time.Sleep(time.Second * 5)
		return &RespBody{Code: 200, Msg: "success"}, nil
	}

	f, err := os.OpenFile(filefullpath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		os.Remove(filefullpath)
		return &RespBody{Code: 500, Msg: err.Error()}, nil
	}
	defer f.Close()
	var n = 0
	var buf = make([]byte, configs.RpcSpaceBuffer)
	f.Seek(int64(b.BlockIndex)*configs.RpcSpaceBuffer, 0)
	n, _ = f.Read(buf)
	return &RespBody{Code: 200, Msg: "success", Data: buf[:n]}, nil
}

// SpacefileAction is used to handle miner requests to download space files.
// The return code is 200 for success, non-200 for failure.
// The returned Msg indicates the result reason.
func (WService) FillerbackAction(body []byte) (proto.Message, error) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	var b FillerBackReq
	err := proto.Unmarshal(body, &b)
	if err != nil {
		return &RespBody{Code: 400, Msg: "Bad Request"}, nil
	}

	if len(b.FileId) == 0 || len(b.FileHash) == 0 {
		return &RespBody{Code: 400, Msg: "Bad Request"}, nil
	}

	var data chain.SpaceFileInfo
	data.FileId = b.FileId
	data.FileHash = b.FileHash
	data.Index = 0
	data.FileSize = 8388608
	data.Acc = types.NewAccountID(b.Publickey)
	blocknum := uint64(math.Ceil(float64(8386771 / configs.BlockSize)))
	if blocknum == 0 {
		blocknum = 1
	}
	data.BlockNum = types.U32(blocknum)
	data.BlockSize = types.U32(uint32(configs.BlockSize))
	data.ScanSize = types.U32(uint32(configs.ScanBlockSize))
	chan_FillerMeta <- data

	return &RespBody{Code: 200, Msg: "success"}, nil
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

func generateFiller(fpath string) error {
	f, err := os.OpenFile(fpath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	for i := 0; i < 2047; i++ {
		f.WriteString(tools.RandStr(4096) + "\n")
	}
	_, err = f.WriteString(tools.RandStr(212))
	if err != nil {
		os.Remove(fpath)
		return err
	}
	err = f.Sync()
	if err != nil {
		os.Remove(fpath)
		return err
	}
	return nil
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
		err    error
		txhash string
	)
	t_active := time.Now()
	for {
		time.Sleep(time.Second * 1)
		for len(chan_FillerMeta) > 0 {
			var tmp = <-chan_FillerMeta
			fm.Add(string(tmp.Acc[:]), tmp)
		}
		if time.Since(t_active).Seconds() > 5 {
			t_active = time.Now()
			for k, v := range fm.fillermetas {
				addr, _ := tools.EncodeToCESSAddr([]byte(k))
				if len(v) >= 8 {
					txhash, err = chain.PutSpaceTagInfoToChain(configs.C.CtrlPrk, types.NewAccountID([]byte(k)), v[:8])
					if txhash == "" || err != nil {
						Tsfm.Sugar().Errorf("%v", err)
						continue
					}
					fm.Delete(k)
					sm.Delete(k)
					fpath := filepath.Join(configs.SpaceCacheDir, addr)
					os.RemoveAll(fpath)
					Tsfm.Sugar().Infof("[%v] %v", addr, txhash)
				} else {
					ok := sm.IsExit(k)
					if !ok && len(v) > 0 {
						txhash, err = chain.PutSpaceTagInfoToChain(configs.C.CtrlPrk, types.NewAccountID([]byte(k)), v[:])
						if txhash == "" || err != nil {
							Tsfm.Sugar().Errorf("%v", err)
							continue
						}
						fm.Delete(k)
						fpath := filepath.Join(configs.SpaceCacheDir, addr)
						os.RemoveAll(fpath)
						Tsfm.Sugar().Infof("[%v] %v", addr, txhash)
					}
				}
			}
		}
	}
}

func task_GenerateFiller(ch chan bool) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
		ch <- true
	}()

	Tgf.Info("-----> Start task_GenerateFiller")

	var (
		err        error
		uid        string
		fillerpath string
	)

	for len(chan_Filler) < 10 {
		for {
			uid, _ = tools.GetGuid(int64(tools.RandomInRange(0, 1024)))
			fillerpath = filepath.Join(configs.SpaceCacheDir, fmt.Sprintf("%s", uid))
			_, err = os.Stat(fillerpath)
			if err != nil {
				break
			}
		}
		err = generateFiller(fillerpath)
		if err != nil {
			Tgf.Sugar().Errorf("generateFiller: %v", err)
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 30)))
			continue
		}

		fstat, _ := os.Stat(fillerpath)
		if fstat.Size() != 8386771 {
			Flr.Sugar().Errorf("file size err: %v", err)
			os.Remove(fillerpath)
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 30)))
			continue
		}

		// calculate file tag info
		var PoDR2commit apiv1.PoDR2Commit
		var commitResponse apiv1.PoDR2CommitResponse
		PoDR2commit.FilePath = fillerpath
		PoDR2commit.BlockSize = configs.BlockSize

		gWait := make(chan bool)
		go func(ch chan bool) {
			runtime.LockOSThread()
			commitResponseCh, err := PoDR2commit.PoDR2ProofCommit(
				apiv1.Key_Ssk,
				string(apiv1.Key_SharedParams),
				int64(configs.ScanBlockSize),
			)
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
			if commitResponse.StatueMsg.StatusCode != apiv1.Success {
				ch <- false
			} else {
				ch <- true
			}
		}(gWait)

		if rst := <-gWait; !rst {
			os.Remove(fillerpath)
			Flr.Sugar().Errorf("PoDR2ProofCommit false")
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 30)))
			continue
		}
		runtime.GC()

		var fillerEle filler
		fillerEle.FillerId = uid
		fillerEle.Path = fillerpath
		fillerEle.T = commitResponse.T
		fillerEle.Sigmas = commitResponse.Sigmas
		chan_Filler <- fillerEle
	}
}

func CombineFillerMeta(addr, fileid, fpath string, pubkey []byte) (chain.SpaceFileInfo, error) {
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

type TagInfo struct {
	T      apiv1.FileTagT
	Sigmas [][]byte `json:"sigmas"`
}

//
func task_ValidateProof(ch chan bool) {
	var (
		err         error
		goeson      bool
		code        int
		puk         chain.Chain_SchedulerPuk
		poDR2verify apiv1.PoDR2Verify
		reqtag      ReadTagReq
		proofs      = make([]chain.Chain_Proofs, 0)
	)
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
		ch <- true
	}()

	Tvp.Info("--> Start task_ValidateProof")

	reqtag.Acc, err = chain.GetPublicKeyByPrk(configs.C.CtrlPrk)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}

	Tvp.Sugar().Infof("--> %v", reqtag.Acc)

	for {
		puk, _, err = chain.GetSchedulerPukFromChain()
		if err != nil {
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 30)))
			continue
		}
		Tvp.Info("--> Successfully found puk")
		Tvp.Sugar().Infof("--> %v", puk.Shared_g)
		Tvp.Sugar().Infof("--> %v", puk.Shared_params)
		Tvp.Sugar().Infof("--> %v", puk.Spk)
		break
	}

	for {
		var verifyResults = make([]chain.VerifyResult, 0)
		proofs, code, err = chain.GetProofsFromChain(configs.C.CtrlPrk)
		if err != nil {
			if code != configs.Code_404 {
				Tvp.Sugar().Errorf("%v", err)
			}
			time.Sleep(time.Minute * time.Duration(tools.RandomInRange(3, 10)))
			continue
		}
		if len(proofs) == 0 {
			time.Sleep(time.Minute * time.Duration(tools.RandomInRange(3, 10)))
			continue
		}

		Tvp.Sugar().Infof("--> Ready to verify %v proofs", len(proofs))

		var respData []byte
		var tag TagInfo
		var minerInfo chain.MinerInfo
		for i := 0; i < len(proofs); i++ {
			if len(verifyResults) > 45 {
				break
			}
			goeson = false
			code = 0

			addr, err := tools.EncodeToCESSAddr(proofs[i].Miner_pubkey[:])
			if err != nil {
				Tvp.Sugar().Errorf("%v EncodeToCESSAddr: %v", proofs[i].Miner_pubkey, err)
			}
			reqtag.FileId = string(proofs[i].Challenge_info.File_id)
			req_proto, err := proto.Marshal(&reqtag)
			if err != nil {
				Tvp.Sugar().Errorf("[%v] Marshal: %v", addr, err)
			}

			for j := 0; j < 3; j++ {
				minerInfo, code, err = chain.GetMinerInfo(proofs[i].Miner_pubkey)
				if err != nil {
					Tvp.Sugar().Errorf("[%v] GetMinerDetailsById: %v", addr, err)
					time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 6)))
				}
				if code == configs.Code_404 {
					goeson = false
					break
				}
				if code == configs.Code_200 {
					goeson = true
					break
				}
			}

			if !goeson {
				resultTemp := chain.VerifyResult{}
				resultTemp.Miner_pubkey = proofs[i].Miner_pubkey
				resultTemp.FileId = proofs[i].Challenge_info.File_id
				resultTemp.Result = false
				verifyResults = append(verifyResults, resultTemp)
				continue
			}

			goeson = false
			for j := 0; j < 3; j++ {
				respData, err = WriteData(string(minerInfo.Ip), configs.RpcService_Miner, configs.RpcMethod_Miner_ReadFileTag, req_proto)
				if err != nil {
					Tvp.Sugar().Errorf("[%v] [%v] WriteData: %v", addr, string(minerInfo.Ip), err)
					time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 6)))
				} else {
					goeson = true
					break
				}
			}

			if !goeson {
				resultTemp := chain.VerifyResult{}
				resultTemp.Miner_pubkey = proofs[i].Miner_pubkey
				resultTemp.FileId = proofs[i].Challenge_info.File_id
				resultTemp.Result = false
				verifyResults = append(verifyResults, resultTemp)
				continue
			}

			err = json.Unmarshal(respData, &tag)
			if err != nil {
				Tvp.Sugar().Errorf("[%v] [%v] Unmarshal: %v", addr, string(minerInfo.Ip), err)
			}
			qSlice, err := apiv1.PoDR2ChallengeGenerateFromChain(proofs[i].Challenge_info.Block_list, proofs[i].Challenge_info.Random)
			if err != nil {
				Tvp.Sugar().Errorf("[%v] [%v] [%v] qslice: %v", addr, len(proofs[i].Challenge_info.Block_list), len(proofs[i].Challenge_info.Random), err)
			}

			poDR2verify.QSlice = qSlice
			poDR2verify.MU = make([][]byte, len(proofs[i].Mu))
			for j := 0; j < len(proofs[i].Mu); j++ {
				poDR2verify.MU[j] = append(poDR2verify.MU[j], proofs[i].Mu[j]...)
			}

			poDR2verify.Sigma = proofs[i].Sigma
			poDR2verify.T = tag.T

			gWait := make(chan bool)
			go func(ch chan bool) {
				runtime.LockOSThread()
				defer func() {
					if err := recover(); err != nil {
						ch <- true
						Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
					}
				}()
				result := poDR2verify.PoDR2ProofVerify(puk.Shared_g, puk.Spk, string(puk.Shared_params))
				ch <- result
			}(gWait)
			result := <-gWait
			resultTemp := chain.VerifyResult{}
			resultTemp.Miner_pubkey = proofs[i].Miner_pubkey
			resultTemp.FileId = proofs[i].Challenge_info.File_id
			resultTemp.Result = types.Bool(result)
			verifyResults = append(verifyResults, resultTemp)
		}
		go processProofResult(verifyResults)
	}
}

func processProofResult(data []chain.VerifyResult) {
	var (
		err    error
		txhash string
		ts     = time.Now().Unix()
		code   = 0
	)
	for code != int(configs.Code_200) && code != int(configs.Code_600) {
		txhash, err = chain.PutProofResult(configs.C.CtrlPrk, data)
		if txhash != "" {
			Tvp.Sugar().Infof("Proof result submitted: %v", txhash)
			break
		}
		if time.Since(time.Unix(ts, 0)).Minutes() > 2.0 {
			Tvp.Sugar().Errorf("Proof result submitted timeout: %v", err)
			break
		}
		time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 20)))
	}
}

//
// func task_RecoveryFiles(ch chan bool) {
// 	var (
// 		recoverFlag  bool
// 		index        int
// 		fileFullPath string
// 		mDatas       = make([]chain.CessChain_AllMinerInfo, 0)
// 	)
// 	defer func() {
// 		if err := recover(); err != nil {
// 			Gpnc.Sugar().Infof("%v", tools.RecoverError(err))
// 		}
// 		ch <- true
// 	}()

// 	Trf.Info("--> Start task_RecoveryFiles")

// 	for {
// 		recoverylist, code, err := chain.GetFileRecoveryByAcc(configs.C.CtrlPrk)
// 		if err != nil {
// 			if code != configs.Code_404 {
// 				Trf.Sugar().Infof(" [Err] GetFileRecoveryByAcc: %v", err)
// 			}
// 			time.Sleep(time.Second * time.Duration(tools.RandomInRange(30, 120)))
// 			continue
// 		}

// 		if len(recoverylist) == 0 {
// 			continue
// 		}

// 		Trf.Sugar().Infof("--> Ready to restore %v files", len(recoverylist))

// 		for i := 0; i < len(recoverylist); i++ {
// 			filename := string(recoverylist[i])
// 			ext := filepath.Ext(filename)
// 			fileid := strings.TrimSuffix(filename, ext)
// 			fmeta, _, err := chain.GetFileMetaInfoOnChain(fileid)
// 			if err != nil {
// 				Trf.Sugar().Infof("--> [Err] [%v] GetFileMetaInfoOnChain: %v", fileid, err)
// 				continue
// 			}

// 			for {
// 				mDatas, _, err = chain.GetAllMinerDataOnChain()
// 				if err == nil {
// 					break
// 				}
// 				time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 30)))
// 			}
// 			Trf.Sugar().Infof("--> Find %v miners", len(mDatas))

// 			filebasedir := filepath.Join(configs.FileCacheDir, fileid)

// 			_, err = os.Stat(filebasedir)
// 			if err != nil {
// 				err = os.Mkdir(filebasedir, os.ModeDir)
// 				if err != nil {
// 					Err.Sugar().Errorf("%v", err)
// 					continue
// 				}
// 			}

// 			index = 0
// 			var recoverIndex int = -1
// 			for d := 0; d < len(fmeta.FileDupl); d++ {
// 				if string(fmeta.FileDupl[d].DuplId) == filename {
// 					recoverIndex = d
// 					break
// 				}
// 			}

// 			if recoverIndex == -1 {
// 				Trf.Sugar().Infof("--> [Err] [%v] No dupl id found to restore", string(recoverylist[i]))
// 				continue
// 			}

// 			recoverFlag = false

// 			fileFullPath = filepath.Join(filebasedir, filename)
// 			fi, err := os.Stat(fileFullPath)
// 			if err == nil {
// 				for {
// 					var randkey types.Bytes
// 					filedump := make([]chain.FileDuplicateInfo, 1)
// 					randkey = fmeta.FileDupl[recoverIndex].RandKey
// 					if len(randkey) == 0 {
// 						break
// 					}
// 					f, err := os.OpenFile(fileFullPath, os.O_RDONLY, os.ModePerm)
// 					if err != nil {
// 						Err.Sugar().Errorf("[%v][%v]", fileFullPath, err)
// 						continue
// 					}
// 					blockTotal := fi.Size() / configs.RpcFileBuffer
// 					if fi.Size()%configs.RpcFileBuffer > 0 {
// 						blockTotal += 1
// 					}
// 					var blockinfo = make([]chain.BlockInfo, blockTotal)
// 					var failminer = make(map[uint64]bool, 0)
// 					var mip = ""
// 					for j := int64(0); j < blockTotal; j++ {
// 						_, err := f.Seek(int64(j*2*1024*1024), 0)
// 						if err != nil {
// 							Err.Sugar().Errorf("[%v][%v]", fileFullPath, err)
// 							f.Close()
// 							continue
// 						}
// 						var buf = make([]byte, configs.RpcFileBuffer)
// 						n, err := f.Read(buf)
// 						if err != nil {
// 							Err.Sugar().Errorf("[%v][%v]", fileFullPath, err)
// 							f.Close()
// 							continue
// 						}

// 						var bo = p.PutFileToBucket{
// 							FileId:     string(recoverylist[i]),
// 							FileHash:   "",
// 							BlockTotal: uint32(blockTotal),
// 							BlockSize:  uint32(n),
// 							BlockIndex: uint32(j),
// 							BlockData:  buf[:n],
// 						}
// 						bob, err := proto.Marshal(&bo)
// 						if err != nil {
// 							Err.Sugar().Errorf("[%v][%v]", fileFullPath, err)
// 							f.Close()
// 							continue
// 						}
// 						for {
// 							if mip == "" {
// 								index = tools.RandomInRange(0, len(mDatas))
// 								_, ok := failminer[uint64(mDatas[index].Peerid)]
// 								if ok {
// 									continue
// 								}
// 								_, err = rpc.WriteData(string(mDatas[index].Ip), configs.RpcService_Miner, configs.RpcMethod_Miner_WriteFile, bob)
// 								if err == nil {
// 									mip = string(mDatas[index].Ip)
// 									blockinfo[j].BlockIndex, _ = tools.IntegerToBytes(uint32(j))
// 									blockinfo[j].BlockSize = types.U32(uint32(n))
// 									break
// 								} else {
// 									failminer[uint64(mDatas[index].Peerid)] = true
// 									Err.Sugar().Errorf("[%v][%v]", fileFullPath, err)
// 									time.Sleep(time.Second * time.Duration(tools.RandomInRange(2, 5)))
// 								}
// 							} else {
// 								_, err = rpc.WriteData(mip, configs.RpcService_Miner, configs.RpcMethod_Miner_WriteFile, bob)
// 								if err != nil {
// 									failminer[uint64(mDatas[index].Peerid)] = true
// 									Err.Sugar().Errorf("[%v][%v]", fileFullPath, err)
// 									time.Sleep(time.Second * time.Duration(tools.RandomInRange(2, 5)))
// 									continue
// 								}
// 								blockinfo[j].BlockIndex, _ = tools.IntegerToBytes(uint32(j))
// 								blockinfo[j].BlockSize = types.U32(uint32(n))
// 								break
// 							}
// 						}
// 					}
// 					f.Close()
// 					filedump[0].DuplId = types.Bytes([]byte(string(recoverylist[i])))
// 					filedump[0].RandKey = randkey
// 					filedump[0].MinerId = mDatas[index].Peerid
// 					filedump[0].MinerIp = mDatas[index].Ip
// 					filedump[0].ScanSize = types.U32(configs.ScanBlockSize)
// 					//mips[i] = string(mDatas[index].Ip)
// 					// Query miner information by id
// 					var mdetails chain.Chain_MinerDetails
// 					for {
// 						mdetails, _, err = chain.GetMinerDetailsById(uint64(mDatas[index].Peerid))
// 						if err != nil {
// 							Err.Sugar().Errorf("[%v]%v", uint64(mDatas[index].Peerid), err)
// 							time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
// 							continue
// 						}
// 						break
// 					}
// 					filedump[0].Acc = mdetails.Address
// 					filedump[0].BlockNum = types.U32(uint32(blockTotal))
// 					filedump[0].BlockInfo = blockinfo
// 					// Upload the file meta information to the chain and write it to the cache
// 					for {
// 						_, err = chain.PutMetaInfoToChain(configs.C.CtrlPrk, fileid, filedump)
// 						if err != nil {
// 							Err.Sugar().Errorf("[%v][%v]", fileid, err)
// 							time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
// 							continue
// 						}
// 						Out.Sugar().Infof("[%v]The copy recovery meta information is successfully uploaded to the chain", fileid)
// 						// c, err := cache.GetCache()
// 						// if err != nil {
// 						// 	Err.Sugar().Errorf("[%v][%v][%v]", t, fileid, err)
// 						// } else {
// 						// 	b, err := json.Marshal(filedump)
// 						// 	if err != nil {
// 						// 		Err.Sugar().Errorf("[%v][%v][%v]", t, fileid, err)
// 						// 	} else {
// 						// 		err = c.Put([]byte(fid), b)
// 						// 		if err != nil {
// 						// 			Err.Sugar().Errorf("[%v][%v][%v]", t, fileid, err)
// 						// 		} else {
// 						// 			Out.Sugar().Infof("[%v][%v]File metainfo write cache success", t, fid)
// 						// 		}
// 						// 	}
// 						// }
// 						break
// 					}

// 					// calculate file tag info
// 					var PoDR2commit proof.PoDR2Commit
// 					var commitResponse proof.PoDR2CommitResponse
// 					PoDR2commit.FilePath = fileFullPath
// 					PoDR2commit.BlockSize = configs.BlockSize
// 					commitResponseCh, err := PoDR2commit.PoDR2ProofCommit(proof.Key_Ssk, string(proof.Key_SharedParams), int64(configs.ScanBlockSize))
// 					if err != nil {
// 						Err.Sugar().Errorf("[%v]%v", fileid, err)
// 						break
// 					}
// 					select {
// 					case commitResponse = <-commitResponseCh:
// 					}
// 					if commitResponse.StatueMsg.StatusCode != proof.Success {
// 						Err.Sugar().Errorf("[%v][%v]", fileid, err)
// 						break
// 					}
// 					var resp p.PutTagToBucket
// 					resp.FileId = string(recoverylist[i])
// 					resp.Name = commitResponse.T.Name
// 					resp.N = commitResponse.T.N
// 					resp.U = commitResponse.T.U
// 					resp.Signature = commitResponse.T.Signature
// 					resp.Sigmas = commitResponse.Sigmas
// 					resp_proto, err := proto.Marshal(&resp)
// 					if err != nil {
// 						Err.Sugar().Errorf("[%v]%v", fileid, err)
// 						break
// 					}
// 					_, err = rpc.WriteData(string(mDatas[index].Ip), configs.RpcService_Miner, configs.RpcMethod_Miner_WriteFileTag, resp_proto)
// 					if err != nil {
// 						Err.Sugar().Errorf("[%v]%v", fileid, err)
// 						break
// 					}

// 					_, err = chain.ClearRecoveredFileNoChain(configs.C.CtrlPrk, recoverylist[i])
// 					if err != nil {
// 						Err.Sugar().Errorf("[%v]%v", fileid, err)
// 						break
// 					}
// 					Out.Sugar().Infof("[%v] File recovery succeeded", string(recoverylist[i]))
// 					recoverFlag = true
// 					break
// 				}
// 			}
// 			if recoverFlag {
// 				continue
// 			}
// 			newFilename := fileid + ".u"
// 			fileuserfullname := filepath.Join(filebasedir, newFilename)
// 			_, err = os.Stat(fileuserfullname)
// 			// download dupl
// 			if err != nil {
// 				for k := 0; k < len(fmeta.FileDupl); k++ {
// 					if string(fmeta.FileDupl[k].DuplId) == filename {
// 						continue
// 					}
// 					filename = string(fmeta.FileDupl[k].DuplId)
// 					fileFullPath = filepath.Join(filebasedir, filename)
// 					_, err = os.Stat(fileFullPath)
// 					if err != nil {
// 						err = rpc.ReadFile(string(fmeta.FileDupl[k].MinerIp), filebasedir, filename, "")
// 						if err != nil {
// 							Err.Sugar().Errorf("[%v]%v", string(fmeta.FileDupl[k].DuplId), err)
// 							continue
// 						}
// 					}

// 					// decryption dupl file
// 					_, err = os.Stat(fileFullPath)
// 					if err == nil {
// 						buf, err := ioutil.ReadFile(fileFullPath)
// 						if err != nil {
// 							Err.Sugar().Errorf("[%v]%v", fileFullPath, err)
// 							os.Remove(fileFullPath)
// 							continue
// 						}
// 						//aes decryption
// 						ivkey := string(fmeta.FileDupl[k].RandKey)[:16]
// 						bkey := base58.Decode(string(fmeta.FileDupl[k].RandKey))
// 						decrypted, err := encryption.AesCtrDecrypt(buf, []byte(bkey), []byte(ivkey))
// 						if err != nil {
// 							Err.Sugar().Errorf("[%v]%v", fileFullPath, err)
// 							os.Remove(fileFullPath)
// 							continue
// 						}
// 						fr, err := os.OpenFile(fileuserfullname, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
// 						if err != nil {
// 							Err.Sugar().Errorf("[%v]%v", fileuserfullname, err)
// 							continue
// 						}
// 						fr.Write(decrypted)
// 						err = fr.Sync()
// 						if err != nil {
// 							Err.Sugar().Errorf("[%v]%v", fileuserfullname, err)
// 							fr.Close()
// 							os.Remove(fileuserfullname)
// 							continue
// 						}
// 						fr.Close()
// 					}
// 				}
// 			}
// 			_, err = os.Stat(fileuserfullname)
// 			if err != nil {
// 				Err.Sugar().Errorf("[%v] File recovery failed", fileid)
// 				continue
// 			}

// 			buf, err := os.ReadFile(fileuserfullname)
// 			if err != nil {
// 				os.Remove(fileuserfullname)
// 				Err.Sugar().Errorf("[%v] File recovery failed: %v", fileid, err)
// 				continue
// 			}

// 			// Generate 32-bit random key for aes encryption
// 			key := tools.GetRandomkey(32)
// 			key_base58 := base58.Encode([]byte(key))
// 			// Aes ctr mode encryption
// 			encrypted, err := encryption.AesCtrEncrypt(buf, []byte(key), []byte(key_base58[:16]))
// 			if err != nil {
// 				Err.Sugar().Errorf("[%v] File recovery failed: %v", fileid, err)
// 				continue
// 			}
// 			duplname := string(recoverylist[i])

// 			duplFallpath := filepath.Join(filebasedir, duplname)
// 			duplf, err := os.OpenFile(duplFallpath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.ModePerm)
// 			if err != nil {
// 				Err.Sugar().Errorf("[%v] File recovery failed: %v", fileid, err)
// 				continue
// 			}
// 			_, err = duplf.Write(encrypted)
// 			if err != nil {
// 				Err.Sugar().Errorf("[%v] File recovery failed: %v", fileid, err)
// 				duplf.Close()
// 				os.Remove(duplFallpath)
// 				continue
// 			}
// 			err = duplf.Sync()
// 			if err != nil {
// 				Err.Sugar().Errorf("[%v] File recovery failed: %v", fileid, err)
// 				duplf.Close()
// 				os.Remove(duplFallpath)
// 				continue
// 			}
// 			duplf.Close()
// 			duplkey := key_base58 + ".k" + strconv.Itoa(recoverIndex)
// 			duplkeyFallpath := filepath.Join(filebasedir, duplkey)
// 			_, err = os.Create(duplkeyFallpath)
// 			if err != nil {
// 				os.Remove(duplFallpath)
// 				Err.Sugar().Errorf("[%v] File recovery failed: %v", fileid, err)
// 				continue
// 			}

// 			for {
// 				fi, err = os.Stat(duplFallpath)
// 				if err != nil {
// 					Err.Sugar().Errorf("[%v] File recovery failed: %v", fileid, err)
// 					break
// 				}
// 				filedump := make([]chain.FileDuplicateInfo, 1)
// 				f, err := os.OpenFile(duplFallpath, os.O_RDONLY, os.ModePerm)
// 				if err != nil {
// 					Err.Sugar().Errorf("[%v] File recovery failed: %v", fileid, err)
// 					break
// 				}
// 				blockTotal := fi.Size() / configs.RpcFileBuffer
// 				if fi.Size()%configs.RpcFileBuffer > 0 {
// 					blockTotal += 1
// 				}
// 				var blockinfo = make([]chain.BlockInfo, blockTotal)
// 				var failminer = make(map[uint64]bool, 0)
// 				var mip = ""
// 				for j := int64(0); j < blockTotal; j++ {
// 					_, err := f.Seek(int64(j*2*1024*1024), 0)
// 					if err != nil {
// 						Err.Sugar().Errorf("[%v] File recovery failed: %v", fileid, err)
// 						f.Close()
// 						break
// 					}
// 					var buf = make([]byte, configs.RpcFileBuffer)
// 					n, err := f.Read(buf)
// 					if err != nil {
// 						Err.Sugar().Errorf("[%v] File recovery failed: %v", fileid, err)
// 						f.Close()
// 						break
// 					}

// 					var bo = p.PutFileToBucket{
// 						FileId:     string(recoverylist[i]),
// 						FileHash:   "",
// 						BlockTotal: uint32(blockTotal),
// 						BlockSize:  uint32(n),
// 						BlockIndex: uint32(j),
// 						BlockData:  buf[:n],
// 					}
// 					bob, err := proto.Marshal(&bo)
// 					if err != nil {
// 						Err.Sugar().Errorf("[%v] File recovery failed: %v", fileid, err)
// 						f.Close()
// 						break
// 					}
// 					for {
// 						if mip == "" {
// 							index = tools.RandomInRange(0, len(mDatas))
// 							_, ok := failminer[uint64(mDatas[index].Peerid)]
// 							if ok {
// 								continue
// 							}
// 							_, err = rpc.WriteData(string(mDatas[index].Ip), configs.RpcService_Miner, configs.RpcMethod_Miner_WriteFile, bob)
// 							if err == nil {
// 								mip = string(mDatas[index].Ip)
// 								blockinfo[j].BlockIndex, _ = tools.IntegerToBytes(uint32(j))
// 								blockinfo[j].BlockSize = types.U32(uint32(n))
// 								break
// 							} else {
// 								failminer[uint64(mDatas[index].Peerid)] = true
// 								Err.Sugar().Errorf("[%v][%v]", fileFullPath, err)
// 								time.Sleep(time.Second * time.Duration(tools.RandomInRange(2, 5)))
// 							}
// 						} else {
// 							_, err = rpc.WriteData(mip, configs.RpcService_Miner, configs.RpcMethod_Miner_WriteFile, bob)
// 							if err != nil {
// 								failminer[uint64(mDatas[index].Peerid)] = true
// 								Err.Sugar().Errorf("[%v][%v]", fileFullPath, err)
// 								time.Sleep(time.Second * time.Duration(tools.RandomInRange(2, 5)))
// 								continue
// 							}
// 							blockinfo[j].BlockIndex, _ = tools.IntegerToBytes(uint32(j))
// 							blockinfo[j].BlockSize = types.U32(uint32(n))
// 							break
// 						}
// 					}
// 				}
// 				f.Close()
// 				filedump[0].DuplId = recoverylist[i]
// 				filedump[0].RandKey = types.Bytes(key_base58)
// 				filedump[0].MinerId = mDatas[index].Peerid
// 				filedump[0].MinerIp = mDatas[index].Ip
// 				filedump[0].ScanSize = types.U32(configs.ScanBlockSize)
// 				//mips[i] = string(mDatas[index].Ip)
// 				// Query miner information by id
// 				var mdetails chain.Chain_MinerDetails
// 				for {
// 					mdetails, _, err = chain.GetMinerDetailsById(uint64(mDatas[index].Peerid))
// 					if err != nil {
// 						Err.Sugar().Errorf("[%v]%v", uint64(mDatas[index].Peerid), err)
// 						time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
// 						continue
// 					}
// 					break
// 				}
// 				filedump[0].Acc = mdetails.Address
// 				filedump[0].BlockNum = types.U32(uint32(blockTotal))
// 				filedump[0].BlockInfo = blockinfo
// 				// Upload the file meta information to the chain and write it to the cache
// 				for {
// 					_, err = chain.PutMetaInfoToChain(configs.C.CtrlPrk, fileid, filedump)
// 					if err != nil {
// 						Err.Sugar().Errorf("[%v][%v]", fileid, err)
// 						time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
// 						continue
// 					}
// 					Out.Sugar().Infof("[%v]The copy recovery meta information is successfully uploaded to the chain", fileid)
// 					// c, err := cache.GetCache()
// 					// if err != nil {
// 					// 	Err.Sugar().Errorf("[%v][%v][%v]", t, fileid, err)
// 					// } else {
// 					// 	b, err := json.Marshal(filedump)
// 					// 	if err != nil {
// 					// 		Err.Sugar().Errorf("[%v][%v][%v]", t, fileid, err)
// 					// 	} else {
// 					// 		err = c.Put([]byte(fid), b)
// 					// 		if err != nil {
// 					// 			Err.Sugar().Errorf("[%v][%v][%v]", t, fileid, err)
// 					// 		} else {
// 					// 			Out.Sugar().Infof("[%v][%v]File metainfo write cache success", t, fid)
// 					// 		}
// 					// 	}
// 					// }
// 					break
// 				}

// 				// calculate file tag info
// 				var PoDR2commit proof.PoDR2Commit
// 				var commitResponse proof.PoDR2CommitResponse
// 				PoDR2commit.FilePath = duplFallpath
// 				PoDR2commit.BlockSize = configs.BlockSize
// 				commitResponseCh, err := PoDR2commit.PoDR2ProofCommit(proof.Key_Ssk, string(proof.Key_SharedParams), int64(configs.ScanBlockSize))
// 				if err != nil {
// 					Err.Sugar().Errorf("[%v]%v", fileid, err)
// 					break
// 				}
// 				select {
// 				case commitResponse = <-commitResponseCh:
// 				}
// 				if commitResponse.StatueMsg.StatusCode != proof.Success {
// 					Err.Sugar().Errorf("[%v][%v]", fileid, err)
// 					break
// 				}
// 				var resp p.PutTagToBucket
// 				resp.FileId = string(recoverylist[i])
// 				resp.Name = commitResponse.T.Name
// 				resp.N = commitResponse.T.N
// 				resp.U = commitResponse.T.U
// 				resp.Signature = commitResponse.T.Signature
// 				resp.Sigmas = commitResponse.Sigmas
// 				resp_proto, err := proto.Marshal(&resp)
// 				if err != nil {
// 					Err.Sugar().Errorf("[%v]%v", fileid, err)
// 					break
// 				}
// 				_, err = rpc.WriteData(string(mDatas[index].Ip), configs.RpcService_Miner, configs.RpcMethod_Miner_WriteFileTag, resp_proto)
// 				if err != nil {
// 					Err.Sugar().Errorf("[%v]%v", fileid, err)
// 					break
// 				}
// 				_, err = chain.ClearRecoveredFileNoChain(configs.C.CtrlPrk, recoverylist[i])
// 				if err != nil {
// 					Err.Sugar().Errorf("[%v]%v", fileid, err)
// 					break
// 				}
// 				Out.Sugar().Infof("[%v] File recovery succeeded", string(recoverylist[i]))
// 				break
// 			}
// 		}
// 	}
// }

//
func task_SyncMinersInfo(ch chan bool) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
		ch <- true
	}()

	Tsmi.Info("-----> Start task_UpdateMinerInfo")
	cacheSt = false
	for {
		c, err := db.GetCache()
		if c == nil || err != nil {
			Tsmi.Sugar().Errorf("GetCache: %v", err)
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(10, 30)))
			continue
		}

		// keys, err := c.IteratorKeys()
		// if err != nil {
		// 	Tsmi.Sugar().Errorf("IteratorKeys: %v", err)
		// 	time.Sleep(time.Second * time.Duration(tools.RandomInRange(10, 30)))
		// 	continue
		// }
		// fmt.Println(keys)
		// isExist := false
		// allMinerAcc, code, _ := chain.GetAllMinerDataOnChain()
		// if code != configs.Code_500 {
		// 	if len(allMinerAcc) == 0 {
		// 		for i := 0; i < len(keys); i++ {
		// 			addr, _ := tools.EncodeToCESSAddr(keys[i])
		// 			err = c.Delete(keys[i])
		// 			if err != nil {
		// 				Tsmi.Sugar().Errorf("[%v] Delete failed: %v", addr, err)
		// 			} else {
		// 				Tsmi.Sugar().Infof("[%v] Delete suc", addr)
		// 			}
		// 		}
		// 	} else {
		// 		for i := 0; i < len(keys); i++ {
		// 			isExist = false
		// 			addr, err := tools.EncodeToCESSAddr(keys[i])
		// 			for j := 0; j < len(allMinerAcc); j++ {
		// 				if string(keys[i]) == string(allMinerAcc[j][:]) {
		// 					isExist = true
		// 					break
		// 				}
		// 			}
		// 			if !isExist {
		// 				err = c.Delete(keys[i])
		// 				if err != nil {
		// 					Tsmi.Sugar().Errorf("[%v] Delete cache failed: %v", addr, err)
		// 				} else {
		// 					Tsmi.Sugar().Infof("[%v] Delete cache", addr)
		// 				}
		// 			} else {
		// 				Tsmi.Sugar().Infof("[%v] Already Cached", addr)
		// 			}
		// 		}
		// 	}
		// }
		allMinerAcc, _, _ := chain.GetAllMinerDataOnChain()
		for i := 0; i < len(allMinerAcc); i++ {
			b := allMinerAcc[i][:]
			addr, err := tools.EncodeToCESSAddr(b)
			if err != nil {
				Tsmi.Sugar().Errorf("[%v] EncodeToCESSAddr: %v", allMinerAcc[i], err)
				continue
			}
			ok, err := c.Has(b)
			if err != nil {
				Tsmi.Sugar().Errorf("[%v] c.Has: %v", addr, err)
				continue
			}

			if ok {
				Tsmi.Sugar().Infof("[%v] Already Cached", addr)
				continue
			}

			var cm chain.Cache_MinerInfo

			mdata, _, err := chain.GetMinerInfo(allMinerAcc[i])
			if err != nil {
				Tsmi.Sugar().Errorf("[%v] GetMinerInfo: %v", addr, err)
				continue
			}
			cm.Peerid = uint64(mdata.PeerId)
			cm.Ip = string(mdata.Ip)
			cm.Pubkey = b

			value, err := json.Marshal(&cm)
			if err != nil {
				Tsmi.Sugar().Errorf("[%v] json.Marshal: %v", addr, err)
				continue
			}
			err = c.Put(b, value)
			if err != nil {
				Tsmi.Sugar().Errorf("[%v] c.Put: %v", addr, err)
			}
			Tsmi.Sugar().Infof("[%v] Cache succeeded", addr)
		}
		cacheSt = true
		if len(allMinerAcc) > 0 {
			time.Sleep(time.Minute * time.Duration(tools.RandomInRange(1, 5)))
		}
	}
}
