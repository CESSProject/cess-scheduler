package rpc

import (
	"cess-scheduler/configs"
	"cess-scheduler/internal/chain"
	"cess-scheduler/internal/db"
	. "cess-scheduler/internal/logger"
	"cess-scheduler/internal/pattern"
	"cess-scheduler/tools"
	"encoding/json"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"time"

	. "cess-scheduler/internal/rpc/protobuf"

	keyring "github.com/CESSProject/go-keyring"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"google.golang.org/protobuf/proto"
)

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

	if pattern.IsPass(string(b.Publickey)) {
		return &RespBody{Code: 403, Msg: "Forbidden"}, nil
	}

	minercache, err := db.Get(b.Publickey)
	if err != nil {
		pattern.AddBlacklist(string(b.Publickey))
		return &RespBody{Code: http.StatusNotFound, Msg: "Not found"}, nil
	}

	addr, err := tools.EncodeToCESSAddr(b.Publickey)
	if err != nil {
		return &RespBody{Code: http.StatusForbidden, Msg: "Invalid public key"}, nil
	}

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

	ok := verkr.Verify(verkr.SigningContext(b.Msg), sign)
	if !ok {
		return &RespBody{Code: http.StatusForbidden, Msg: "Authentication failed"}, nil
	}

	var minerinfo chain.Cache_MinerInfo

	err = json.Unmarshal(minercache, &minerinfo)
	if err != nil {
		return &RespBody{Code: http.StatusInternalServerError, Msg: "Cache error"}, nil
	}

	if pattern.GetBaseFillerLength() == 0 {
		if len(pattern.Chan_Filler) == 0 {
			return &RespBody{Code: http.StatusServiceUnavailable, Msg: "Filler is empty"}, nil
		}

		filler := <-pattern.Chan_Filler
		Tgf.Sugar().Infof("Consumes a filler: %v", filler.FillerId)
		var resp RespSpaceInfo
		resp.Token = pattern.UpdateSpacemap(string(b.Publickey), minerinfo.Ip, filler.FillerId)
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

	fillerid, ip, err := pattern.GetAndInsertBaseFiller(minerinfo.Ip)
	if err != nil {
		if len(pattern.Chan_Filler) == 0 {
			return &RespBody{Code: http.StatusServiceUnavailable, Msg: "ServiceUnavailable"}, nil
		}
		filler := <-pattern.Chan_Filler
		Tgf.Sugar().Infof("Consumes a filler: %v", filler.FillerId)
		var resp RespSpaceInfo
		resp.Token = pattern.UpdateSpacemap(string(b.Publickey), minerinfo.Ip, filler.FillerId)
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
	var resp pattern.BaseFiller
	resp.FillerId = fillerid
	resp.MinerIp = make([]string, 0)
	resp.MinerIp = append(resp.MinerIp, ip)
	resp_b, err := json.Marshal(resp)
	if err != nil {
		Flr.Sugar().Errorf("[%v] Marshal: %v", addr, err)
		return &RespBody{Code: http.StatusInternalServerError, Msg: err.Error()}, nil
	}
	Flr.Sugar().Infof("[%v] Copy filler: %v, %v", addr, fillerid, ip)
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

	pubkey, fname, ip, err := pattern.VerifySpaceToken(b.Token)
	if err != nil {
		return &RespBody{Code: 403, Msg: err.Error()}, nil
	}
	if b.BlockIndex == 1 {
		sm.UpdateTimeIfExists(pubkey, ip, fname)
	}
	co.UpdateTime(pubkey)

	addr, err := tools.EncodeToCESSAddr([]byte(pubkey))
	if err != nil {
		return &RespBody{Code: 400, Msg: "Bad publickey"}, nil
	}

	filefullpath := filepath.Join(configs.SpaceCacheDir, fname)
	if b.BlockIndex == 16 {
		Flr.Sugar().Infof("[%v] Transferred filler: %v", addr, fname)
		var data chain.SpaceFileInfo
		data, err = combineFillerMeta(addr, fname, filefullpath, []byte(pubkey))
		if err != nil {
			os.Remove(filefullpath)
			return &RespBody{Code: 500, Msg: err.Error()}, nil
		}
		pattern.AddBaseFiller(ip, fname)
		pattern.Chan_FillerMeta <- data
		os.Remove(filefullpath)
		return &RespBody{Code: 200, Msg: "success"}, nil
	}

	f, err := os.OpenFile(filefullpath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		os.Remove(filefullpath)
		return &RespBody{Code: 500, Msg: err.Error()}, nil
	}
	defer f.Close()
	var n = 0
	var buf = make([]byte, RpcSpaceBuffer)
	f.Seek(int64(b.BlockIndex)*RpcSpaceBuffer, 0)
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
	pattern.Chan_FillerMeta <- data

	return &RespBody{Code: 200, Msg: "success"}, nil
}
