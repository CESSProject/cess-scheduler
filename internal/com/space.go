// /*
//    Copyright 2022 CESS scheduler authors

//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at

//         http://www.apache.org/licenses/LICENSE-2.0

//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.
// */

package com

// import (
// 	"encoding/json"
// 	"fmt"
// 	"math"

// 	"os"
// 	"path/filepath"
// 	"time"

// 	. "github.com/CESSProject/cess-scheduler/api/protobuf"
// 	"github.com/CESSProject/cess-scheduler/configs"
// 	"github.com/CESSProject/cess-scheduler/internal/pattern"
// 	"github.com/CESSProject/cess-scheduler/pkg/chain"
// 	"github.com/CESSProject/cess-scheduler/pkg/utils"

// 	keyring "github.com/CESSProject/go-keyring"
// 	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
// 	"github.com/pkg/errors"
// 	"google.golang.org/protobuf/proto"
// )

// // SpaceAction is used to handle miner requests to space files.
// // The return code is 200 for success, non-200 for failure.
// // The returned Msg indicates the result reason.
// func (w *WService) SpaceAction(body []byte) (proto.Message, error) {
// 	if pattern.ChainStatus.Load() == false {
// 		return &RespBody{Code: 500, Msg: chain.ERR_RPC_CONNECTION.Error()}, nil
// 	}

// 	if pattern.GetFileAuthLength() > 1 {
// 		return &RespBody{Code: 400, Msg: "busy"}, nil
// 	}

// 	var b SpaceReq
// 	err := proto.Unmarshal(body, &b)
// 	if err != nil {
// 		return &RespBody{Code: 400, Msg: "Bad request"}, nil
// 	}
// 	addr, err := utils.EncodePublicKeyAsCessAccount(b.Publickey)
// 	if err != nil {
// 		return &RespBody{Code: 403, Msg: "Invalid public key"}, nil
// 	}
// 	// ok, _ := w.Has(b.Publickey)
// 	// if !ok {
// 	// 	return &RespBody{Code: 403, Msg: "Invalid miner"}, nil
// 	// }

// 	if addr != "cXfyomKDABfehLkvARFE854wgDJFMbsxwAJEHezRb6mfcAi2y" &&
// 		addr != "cXgDBpxj2vHhR9qP8wTkZ5ZST9YMu6WznFsEAZi3SZPD4b4qw" &&
// 		addr != "cXjEGJCcQFST4P9ESL7QiP37R4SZPq7F8wzC7oizi4Ce5nXQp" {
// 		return &RespBody{Code: 403, Msg: "Invalid miner"}, nil
// 	}

// 	var isBaseFiller bool
// 	if !pattern.HasBaseFiller(string(b.Publickey)) {
// 		tk := time.After(time.Second * 5)
// 		select {
// 		case <-tk:
// 			fmt.Println(addr, " 没有基础filler,没有抢到名额")
// 			isBaseFiller = false
// 			break
// 		case <-pattern.C_Max_Miner_Filler:
// 			fmt.Println(addr, " 消费一个名额")
// 			isBaseFiller = true
// 			break
// 		}
// 	} else {
// 		fmt.Println(addr, " 有基础filler,不参与名额消费")
// 	}
// 	// if isBaseFiller && pattern.HasBaseFiller(string(b.Publickey)) {
// 	// 	fmt.Println("获得一个名额，但是有基础filler ", b.Publickey)
// 	// 	isBaseFiller = false
// 	// }

// 	var success bool
// 	defer func() {
// 		if err := recover(); err != nil {
// 			w.Log("panic", "error", utils.RecoverError(err))
// 		}
// 		if isBaseFiller && !success {
// 			fmt.Println(addr, " 传输失败，增加一个名额")
// 			pattern.C_Max_Miner_Filler <- true
// 		}
// 	}()

// 	// if !pattern.IsPass(string(b.Publickey)) {
// 	// 	return &RespBody{Code: 403, Msg: "Forbidden"}, nil
// 	// }

// 	// if pattern.IsMaxSpacem(string(b.Publickey)) {
// 	// 	return &RespBody{Code: 403, Msg: "Busy"}, nil
// 	// }

// 	var minerIp string
// 	minercache, err := w.Get(b.Publickey)
// 	if err != nil {
// 		//pattern.AddBlacklist(string(b.Publickey))
// 		mdata, err := w.GetStorageMinerInfo(b.Publickey)
// 		if err != nil {
// 			w.Log("filler", "error", errors.Errorf("%v", err))
// 			if err.Error() == chain.ERR_RPC_CONNECTION.Error() {
// 				pattern.ChainStatus.Store(false)
// 			}
// 			return &RespBody{Code: 400, Msg: err.Error()}, nil
// 		}

// 		if string(mdata.State) != "positive" {
// 			return &RespBody{Code: 403, Msg: "Inactive state"}, nil
// 		}
// 		minerIp = string(mdata.Ip)
// 	} else {
// 		var minerinfo chain.Cache_MinerInfo
// 		err = json.Unmarshal(minercache, &minerinfo)
// 		if err != nil {
// 			w.Delete(b.Publickey)
// 			return &RespBody{Code: 500, Msg: "Cache error"}, nil
// 		}
// 		minerIp = minerinfo.Ip
// 	}

// 	// addr, err := utils.EncodePublicKeyAsCessAccount(b.Publickey)
// 	// if err != nil {
// 	// 	return &RespBody{Code: 403, Msg: "Invalid public key"}, nil
// 	// }

// 	ss58addr, err := utils.EncodePublicKeyAsSubstrateAccount(b.Publickey)
// 	if err != nil {
// 		return &RespBody{Code: 403, Msg: "Invalid public key"}, nil
// 	}
// 	verkr, _ := keyring.FromURI(ss58addr, keyring.NetSubstrate{})

// 	if len(b.Sign) < 64 {
// 		return &RespBody{Code: 403, Msg: "Signature length error"}, nil
// 	}

// 	var sign [64]byte
// 	for i := 0; i < 64; i++ {
// 		sign[i] = b.Sign[i]
// 	}

// 	ok := verkr.Verify(verkr.SigningContext(b.Msg), sign)
// 	if !ok {
// 		return &RespBody{Code: 403, Msg: "Signature verification failed"}, nil
// 	}

// 	if isBaseFiller {
// 		if len(pattern.C_Filler) <= 0 {
// 			return &RespBody{Code: 500, Msg: "Base filler not available"}, nil
// 		}
// 		filler := <-pattern.C_Filler
// 		w.Log("gf", "info", errors.Errorf("Consumes a filler: %v", filler.FillerId))
// 		var resp RespSpaceInfo
// 		resp.Token = utils.RandStr(16)
// 		pattern.PutMinerToken(string(b.Publickey), resp.Token, filler.FillerId)
// 		resp.FileId = filler.FillerId
// 		resp.T = filler.T
// 		resp.Sigmas = filler.Sigmas
// 		resp_b, err := json.Marshal(resp)
// 		if err != nil {
// 			os.Remove(filler.Path)
// 			w.Log("filler", "error", errors.Errorf("[%v] Marshal: %v", addr, err))
// 			return &RespBody{Code: 500, Msg: err.Error()}, nil
// 		}
// 		w.Log("filler", "info", errors.Errorf("[%v] Base filler: %v", addr, filler.FillerId))
// 		pattern.AddBaseFiller(string(b.Publickey), minerIp, filler.FillerId)
// 		success = true
// 		return &RespBody{Code: 200, Msg: "success", Data: resp_b}, nil
// 	}

// 	fillerid, ip, err := pattern.GetBaseFiller(string(b.Publickey))
// 	if err != nil {
// 		return &RespBody{Code: 400, Msg: "Copy filler not available"}, nil
// 	}
// 	var resp pattern.BaseFiller
// 	resp.FillerId = fillerid
// 	resp.MinerIp = make([]string, 0)
// 	resp.MinerIp = append(resp.MinerIp, ip)
// 	resp_b, err := json.Marshal(resp)
// 	if err != nil {
// 		w.Log("filler", "error", errors.Errorf("[%v] Marshal: %v", addr, err))
// 		return &RespBody{Code: 500, Msg: err.Error()}, nil
// 	}
// 	w.Log("filler", "info", errors.Errorf("[%v] Copy filler: %v, %v", addr, fillerid, ip))
// 	return &RespBody{Code: 201, Msg: "success", Data: resp_b}, nil
// }

// // SpacefileAction is used to handle miner requests to download space files.
// // The return code is 200 for success, non-200 for failure.
// // The returned Msg indicates the result reason.
// func (w *WService) SpacefileAction(body []byte) (proto.Message, error) {
// 	defer func() {
// 		if err := recover(); err != nil {
// 			w.Log("panic", "error", utils.RecoverError(err))
// 		}
// 	}()

// 	var b SpaceFileReq
// 	err := proto.Unmarshal(body, &b)
// 	if err != nil {
// 		return &RespBody{Code: 400, Msg: "Bad Request"}, nil
// 	}

// 	if b.BlockIndex > 16 {
// 		return &RespBody{Code: 400, Msg: "Invalid blocknum"}, nil
// 	}

// 	if b.Token == "" {
// 		return &RespBody{Code: 400, Msg: "Empty token"}, nil
// 	}

// 	token, fid := pattern.GetMinerToken(string(b.Publickey))

// 	if b.Token != token {
// 		return &RespBody{Code: 403, Msg: "Authentication failed"}, nil
// 	}

// 	addr, err := utils.EncodePublicKeyAsCessAccount([]byte(b.Publickey))
// 	if err != nil {
// 		return &RespBody{Code: 400, Msg: "Bad publickey"}, nil
// 	}

// 	var success bool
// 	defer func() {
// 		if !success {
// 			fmt.Println(addr, " 传输完成，增加一个名额")
// 			pattern.C_Max_Miner_Filler <- true
// 		}
// 	}()

// 	// if b.BlockIndex == 1 {
// 	// 	pattern.UpdateSpacemap(pubkey, ip, fname)
// 	// }

// 	filefullpath := filepath.Join(w.fillerDir, fid)
// 	if b.BlockIndex == 16 {
// 		w.Log("filler", "info", errors.Errorf("[%v] Transferred filler: %v", addr, fid))
// 		var data chain.FillerMetaInfo
// 		data, err = combineFillerMeta(addr, fid, filefullpath, []byte(b.Publickey))
// 		if err != nil {
// 			os.Remove(filefullpath)
// 			return &RespBody{Code: 500, Msg: err.Error()}, nil
// 		}
// 		pattern.AppendBaseFiller(string(b.Publickey), string(b.Publickey))
// 		pattern.C_FillerMeta <- data
// 		os.Remove(filefullpath)
// 		return &RespBody{Code: 200, Msg: "success"}, nil
// 	}

// 	f, err := os.OpenFile(filefullpath, os.O_RDONLY, os.ModePerm)
// 	if err != nil {
// 		os.Remove(filefullpath)
// 		return &RespBody{Code: 500, Msg: err.Error()}, nil
// 	}
// 	defer f.Close()
// 	var n = 0
// 	var buf = make([]byte, RpcSpaceBuffer)
// 	f.Seek(int64(b.BlockIndex)*RpcSpaceBuffer, 0)
// 	n, _ = f.Read(buf)
// 	success = true
// 	return &RespBody{Code: 200, Msg: "success", Data: buf[:n]}, nil
// }

// // SpacefileAction is used to handle miner requests to download space files.
// // The return code is 200 for success, non-200 for failure.
// // The returned Msg indicates the result reason.
// func (w *WService) FillerbackAction(body []byte) (proto.Message, error) {
// 	defer func() {
// 		if err := recover(); err != nil {
// 			w.Log("panic", "error", utils.RecoverError(err))
// 		}
// 	}()

// 	var b FillerBackReq
// 	err := proto.Unmarshal(body, &b)
// 	if err != nil {
// 		return &RespBody{Code: 400, Msg: "Bad Request"}, nil
// 	}

// 	if len(b.FileId) == 0 {
// 		return &RespBody{Code: 400, Msg: "Bad Request"}, nil
// 	}

// 	var data chain.FillerMetaInfo
// 	data.Id = b.FileId
// 	data.Hash = b.FileHash
// 	data.Index = 0
// 	data.Size = 8386771
// 	data.Acc = types.NewAccountID(b.Publickey)
// 	blocknum := uint64(math.Ceil(float64(8386771 / configs.BlockSize)))
// 	if blocknum == 0 {
// 		blocknum = 1
// 	}
// 	data.BlockNum = types.U32(blocknum)
// 	data.BlockSize = types.U32(uint32(configs.BlockSize))
// 	pattern.C_FillerMeta <- data

// 	return &RespBody{Code: 200, Msg: "success"}, nil
// }

// func (w *WService) FillerfallAction(body []byte) (proto.Message, error) {
// 	defer func() {
// 		if err := recover(); err != nil {
// 			w.Log("panic", "error", utils.RecoverError(err))
// 		}
// 	}()

// 	var b FillerBackReq
// 	err := proto.Unmarshal(body, &b)
// 	if err != nil {
// 		return &RespBody{Code: 400, Msg: "Bad Request"}, nil
// 	}

// 	if len(b.FileId) == 0 {
// 		return &RespBody{Code: 400, Msg: "Bad Request"}, nil
// 	}

// 	if len(b.FileHash) == 0 {
// 		pattern.DelereBaseFiller(string(b.FileId))
// 	}

// 	return &RespBody{Code: 200, Msg: "success"}, nil
// }
