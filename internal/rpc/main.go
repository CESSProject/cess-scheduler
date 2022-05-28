package rpc

import (
	"cess-scheduler/configs"
	"cess-scheduler/internal/chain"
	"cess-scheduler/internal/encryption"
	. "cess-scheduler/internal/logger"
	proof "cess-scheduler/internal/proof/apiv1"
	"cess-scheduler/tools"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	. "cess-scheduler/internal/rpc/protobuf"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"google.golang.org/protobuf/proto"
	"storj.io/common/base58"
)

type WService struct {
}

// Start tcp service.
// If an error occurs, it will exit immediately.
func Rpc_Main() {
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

// WritefileAction is used to handle client requests to upload files.
// The return code is 0 for success, non-0 for failure.
// The returned Msg indicates the result reason.
func (WService) WritefileAction(body []byte) (proto.Message, error) {
	var (
		err error
		b   FileUploadInfo
	)

	err = proto.Unmarshal(body, &b)
	if err != nil {
		return &RespBody{Code: 400, Msg: "Http error"}, nil
	}

	Uld.Sugar().Infof("+++> Upload [%v] %v", b.FileId, b.BlockIndex)

	cachepath := filepath.Join(configs.FileCacheDir, b.FileId)
	fileFullPath := filepath.Join(cachepath, b.FileId+".cess")

	if b.BlockIndex == 0 {
		return &RespBody{Code: 400, Msg: "File block numbering starts from 1"}, nil
	}

	if b.BlockIndex == 1 {
		_, code, err := chain.GetFileMetaInfoOnChain(b.FileId)
		if err != nil {
			if code == configs.Code_404 {
				Uld.Sugar().Infof("[%v] File not found on chain", b.FileId)
				return &RespBody{Code: int32(code), Msg: "File not found on chain"}, nil
			}
			Uld.Sugar().Infof("[%v] GetFileMetaInfoOnChain err: %v", b.FileId, err)
			return &RespBody{Code: int32(code), Msg: err.Error()}, nil
		}
		err = tools.CreatDirIfNotExist(cachepath)
		if err != nil {
			Uld.Sugar().Infof("[%v] CreatDirIfNotExist err: %v", b.FileId, err)
			return &RespBody{Code: 500, Msg: err.Error()}, nil
		}
		_, err = os.Create(fileFullPath)
		if err != nil {
			Uld.Sugar().Infof("[%v] Create file err: %v", b.FileId, err)
			return &RespBody{Code: 500, Msg: err.Error()}, nil
		}
	}

	f, err := os.OpenFile(fileFullPath, os.O_RDWR|os.O_APPEND, os.ModePerm)
	if err != nil {
		Uld.Sugar().Infof("[%v] OpenFile-1 err: %v", b.FileId, err)
		return &RespBody{Code: 500, Msg: err.Error()}, nil
	}

	_, err = f.Write(b.Data)
	if err != nil {
		f.Close()
		Uld.Sugar().Infof("[%v] f.Write err: %v", b.FileId, err)
		return &RespBody{Code: 500, Msg: err.Error()}, nil
	}

	err = f.Sync()
	if err != nil {
		f.Close()
		Uld.Sugar().Infof("[%v] f.Sync err: %v", b.FileId, err)
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	}

	f.Close()

	if b.BlockIndex == b.BlockTotal {
		backupNum := configs.Backups_Min
		fmeta, _, err := chain.GetFileMetaInfoOnChain(b.FileId)
		if err == nil {
			backupNum = uint8(fmeta.Backups)
		}

		if backupNum < configs.Backups_Min {
			backupNum = configs.Backups_Min
		}
		if backupNum > configs.Backups_Max {
			backupNum = configs.Backups_Max
		}

		buf, err := os.ReadFile(fileFullPath)
		if err != nil {
			Uld.Sugar().Infof("[%v] ReadFile err: %v", b.FileId, err)
			return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
		}

		var duplnamelist = make([]string, 0)
		var duplkeynamelist = make([]string, 0)

		for i := uint8(0); i < backupNum; {
			// Generate 32-bit random key for aes encryption
			key := tools.GetRandomkey(32)
			key_base58 := base58.Encode([]byte(key))
			// Aes ctr mode encryption
			encrypted, err := encryption.AesCtrEncrypt(buf, []byte(key), []byte(key_base58)[:16])
			if err != nil {
				Uld.Sugar().Infof("[%v] AesCtrEncrypt err: %v", b.FileId, err)
				continue
			}

			duplname := b.FileId + ".d" + strconv.Itoa(int(i))
			duplFallpath := filepath.Join(cachepath, duplname)

			duplf, err := os.OpenFile(duplFallpath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.ModePerm)
			if err != nil {
				Uld.Sugar().Infof("[%v] [%v] OpenFile-2 err: %v", b.FileId, duplFallpath, err)
				continue
			}
			_, err = duplf.Write(encrypted)
			if err != nil {
				duplf.Close()
				os.Remove(duplFallpath)
				Uld.Sugar().Infof("[%v] [%v] duplf.Write err: %v", b.FileId, duplFallpath, err)
				continue
			}
			err = duplf.Sync()
			if err != nil {
				duplf.Close()
				os.Remove(duplFallpath)
				Uld.Sugar().Infof("[%v] [%v] f.Sync-2 err: %v", b.FileId, duplFallpath, err)
				continue
			}
			duplf.Close()
			duplkey := string(key_base58) + ".k" + strconv.Itoa(int(i))
			duplkeyFallpath := filepath.Join(cachepath, duplkey)
			_, err = os.Create(duplkeyFallpath)
			if err != nil {
				os.Remove(duplFallpath)
				Uld.Sugar().Infof("[%v] [%v] os.Create-2 err: %v", b.FileId, duplkeyFallpath, err)
				continue
			} else {
				duplnamelist = append(duplnamelist, duplFallpath)
				duplkeynamelist = append(duplkeynamelist, duplkeyFallpath)
				i++
			}
		}
		os.Remove(fileFullPath)
		go storeFiles(b.FileId, duplnamelist, duplkeynamelist)
		//go processingfile(t, b.FileId, cachepath, duplnamelist, duplkeynamelist)
		Uld.Sugar().Infof("[%v] All %v chunks are uploaded successfully", b.FileId, b.BlockTotal)
		return &RespBody{Code: 200, Msg: "success"}, nil

	}
	Uld.Sugar().Infof("[%v] The %v chunk uploaded successfully", b.FileId, b.BlockIndex)
	return &RespBody{Code: 200, Msg: "success", Data: nil}, nil
}

func storeFiles(fid string, duplnamelist, duplkeynamelist []string) {
	var (
		channel_map = make(map[int]chan uint8, len(duplnamelist))
	)

	Uld.Sugar().Infof("[%v] Prepare to store %v replicas to miners", fid, len(duplnamelist))

	for i := 0; i < len(duplnamelist); i++ {
		channel_map[i] = make(chan uint8, 1)
	}

	for i := 0; i < len(duplnamelist); i++ {
		go backupFile(channel_map[i], i, fid, duplnamelist[i], duplkeynamelist[i])
	}

	for {
		for k, v := range channel_map {
			result := <-v
			if result == 1 {
				go backupFile(channel_map[k], k, fid, duplnamelist[k], duplkeynamelist[k])
				continue
			}
			if result == 2 {
				delete(channel_map, k)
				Uld.Sugar().Infof("[%v] The %v copy is successfully stored", fid, k)
				break
			}
			if result == 3 {
				delete(channel_map, k)
				Uld.Sugar().Infof("[%v] The %v copy is failed stored", fid, k)
			}
		}
		if len(channel_map) == 0 {
			Uld.Sugar().Infof("[%v] All replicas stored successfully", fid)
			return
		}
	}
}

// ReadfileAction is used to handle client requests to download files.
// The return code is 0 for success, non-0 for failure.
// The returned Msg indicates the result reason.
func (WService) ReadfileAction(body []byte) (proto.Message, error) {
	var (
		err error
		b   FileDownloadReq
	)

	err = proto.Unmarshal(body, &b)
	if err != nil {
		return &RespBody{Code: 400, Msg: "Http error"}, nil
	}

	Dld.Sugar().Infof("---> Download [%v] %v", b.FileId, b.BlockIndex)

	if b.BlockIndex == 0 {
		return &RespBody{Code: 400, Msg: "File block numbering starts from 1"}, nil
	}

	if b.BlockIndex == 1 {
		uspace, code, err := chain.GetUserSpaceOnChain(b.WalletAddress)
		if err != nil {
			Dld.Sugar().Infof("[%v] GetUserSpaceOnChain err: %v", b.FileId, err)
			return &RespBody{Code: int32(code), Msg: err.Error()}, nil
		}

		fmeta, code, err := chain.GetFileMetaInfoOnChain(b.FileId)
		if err != nil {
			Dld.Sugar().Infof("[%v] GetFileMetaInfoOnChain err: %v", b.FileId, err)
			return &RespBody{Code: int32(code), Msg: err.Error()}, nil
		}

		if uspace.RemainingSpace.CmpAbs(new(big.Int).SetUint64(uint64(fmeta.FileSize))) == -1 {
			Dld.Sugar().Infof("[%v] Not enough space", b.FileId)
			return &RespBody{Code: 403, Msg: "Not enough space"}, nil
		}

		if string(fmeta.FileState) != "active" {
			Dld.Sugar().Infof("[%v] Please download later", b.FileId)
			return &RespBody{Code: 403, Msg: "Please download later"}, nil
		}
	}

	path := filepath.Join(configs.FileCacheDir, b.FileId)
	_, err = os.Stat(path)
	if err != nil {
		os.MkdirAll(path, os.ModeDir)
	}
	filefullname := filepath.Join(path, b.FileId+".u")
	_, err = os.Stat(filefullname)
	if err != nil {
		// file not exist, query dupl file
		for i := 0; i < int(configs.Backups_Max); i++ {
			duplname := filepath.Join(path, b.FileId+".d"+strconv.Itoa(i))
			_, err = os.Stat(duplname)
			if err == nil {
				buf, err := ioutil.ReadFile(duplname)
				if err != nil {
					Dld.Sugar().Infof("[%v] [%v] ReadFile-1 err: %v", b.FileId, duplname, err)
					os.Remove(duplname)
					continue
				}
				fmeta, code, err := chain.GetFileMetaInfoOnChain(b.FileId)
				if err != nil {
					Dld.Sugar().Infof("[%v] GetFileMetaInfoOnChain err: %v", b.FileId, err)
					return &RespBody{Code: int32(code), Msg: err.Error(), Data: nil}, nil
				}
				//aes decryption
				ivkey := string(fmeta.FileDupl[i].RandKey)[:16]
				bkey := base58.Decode(string(fmeta.FileDupl[i].RandKey))
				decrypted, err := encryption.AesCtrDecrypt(buf, []byte(bkey), []byte(ivkey))
				if err != nil {
					Dld.Sugar().Infof("[%v] [%v] AesCtrDecrypt-1 err: ", b.FileId, duplname, err)
					os.Remove(duplname)
					continue
				}
				fu, err := os.OpenFile(filefullname, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
				if err != nil {
					Dld.Sugar().Infof("[%v] [%v] OpenFile-1 err: %v", b.FileId, filefullname, err)
					continue
				}
				fu.Write(decrypted)
				err = fu.Sync()
				if err != nil {
					Dld.Sugar().Infof("[%v] [%v] fu.Sync err: %v", b.FileId, filefullname, err)
					fu.Close()
					os.Remove(filefullname)
					continue
				}
				fu.Close()
				break
			}
		}
	}

	fstat, err := os.Stat(filefullname)
	if err == nil {
		fuser, err := os.OpenFile(filefullname, os.O_RDONLY, os.ModePerm)
		if err != nil {
			Dld.Sugar().Infof("[%v] [%v] OpenFile-2 err: %v", b.FileId, filefullname, err)
			return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
		}
		defer fuser.Close()
		blockTotal := fstat.Size() / configs.RpcFileBuffer
		if fstat.Size()%configs.RpcFileBuffer != 0 {
			blockTotal += 1
		}
		var tmp = make([]byte, configs.RpcFileBuffer)
		var blockSize int32
		var n int

		fuser.Seek(int64((b.BlockIndex-1)*configs.RpcFileBuffer), 0)
		n, _ = fuser.Read(tmp)
		blockSize = int32(n)

		respb := &FileDownloadInfo{
			FileId:     b.FileId,
			BlockTotal: int32(blockTotal),
			BlockSize:  blockSize,
			BlockIndex: b.BlockIndex,
			Data:       tmp[:n],
		}
		protob, err := proto.Marshal(respb)
		if err != nil {
			Dld.Sugar().Infof("[%v] [%v] Marshal err: ", b.FileId, filefullname, err)
			return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
		}
		Dld.Sugar().Infof("[%v] [%v] download successful", b.FileId)
		return &RespBody{Code: 200, Msg: "success", Data: protob}, nil
	}

	// download dupl
	fmeta, code, err := chain.GetFileMetaInfoOnChain(b.FileId)
	if err != nil {
		Dld.Sugar().Infof("[%v] GetFileMetaInfoOnChain err: %v", b.FileId, err)
		return &RespBody{Code: int32(code), Msg: err.Error(), Data: nil}, nil
	}

	var client *Client
	var index int = -1
	for i := 0; i < len(fmeta.FileDupl); i++ {
		dstip := "ws://" + string(base58.Decode(string(fmeta.FileDupl[i].MinerIp)))
		ctx, _ := context.WithTimeout(context.Background(), 6*time.Second)
		client, err = DialWebsocket(ctx, dstip, "")
		if err != nil {
			continue
		}
		index = i
		break
	}

	if client != nil && index != -1 {
		err = ReadFile2(client, path, string(fmeta.FileDupl[index].DuplId), b.WalletAddress)
		if err != nil {
			Dld.Sugar().Infof("[%v] ReadFile2 err: %v", b.FileId, err)
			return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
		}
	} else {
		Dld.Sugar().Infof("[%v] All miners connection failed", b.FileId)
		return &RespBody{Code: 500, Msg: "No miners available", Data: nil}, nil
	}

	// file not exist, query dupl file
	for i := 0; i < len(fmeta.FileDupl); i++ {
		duplname := filepath.Join(path, b.FileId+".d"+strconv.Itoa(i))
		_, err = os.Stat(duplname)
		if err == nil {
			buf, err := ioutil.ReadFile(duplname)
			if err != nil {
				Dld.Sugar().Infof("[%v] [%v] ReadFile-3 err: ", b.FileId, duplname, err)
				os.Remove(duplname)
				continue
			}
			//aes decryption
			ivkey := string(fmeta.FileDupl[i].RandKey)[:16]
			bkey := base58.Decode(string(fmeta.FileDupl[i].RandKey))
			decrypted, err := encryption.AesCtrDecrypt(buf, bkey, []byte(ivkey))
			if err != nil {
				Dld.Sugar().Infof("[%v] [%v] AesCtrDecrypt-2 err: %v", b.FileId, duplname, err)
				os.Remove(duplname)
				continue
			}
			fu, err := os.OpenFile(filefullname, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
			if err != nil {
				Dld.Sugar().Infof("[%v] [%v] OpenFile-3 err: %v", b.FileId, filefullname, err)
				continue
			}
			fu.Write(decrypted)
			err = fu.Sync()
			if err != nil {
				Dld.Sugar().Infof("[%v] [%v] fu.Sync err: %v", b.FileId, filefullname, err)
				fu.Close()
				os.Remove(filefullname)
				continue
			}
			fu.Close()
			break
		}
	}

	fstat, err = os.Stat(filefullname)
	if err == nil {
		fuser, err := os.OpenFile(filefullname, os.O_RDONLY, os.ModePerm)
		if err != nil {
			Dld.Sugar().Infof("[%v] [%v] ReadFile-4 err: ", b.FileId, filefullname, err)
			return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
		}
		defer fuser.Close()
		blockTotal := fstat.Size() / configs.RpcFileBuffer
		if fstat.Size()%configs.RpcFileBuffer != 0 {
			blockTotal += 1
		}
		var tmp = make([]byte, configs.RpcFileBuffer)
		var blockSize int32
		var n int
		fuser.Seek(int64(b.BlockIndex*configs.RpcFileBuffer), 0)
		n, _ = fuser.Read(tmp)
		blockSize = int32(n)
		respb := &FileDownloadInfo{
			FileId:     b.FileId,
			BlockTotal: int32(blockTotal),
			BlockSize:  blockSize,
			BlockIndex: b.BlockIndex,
			Data:       tmp[:n],
		}
		protob, err := proto.Marshal(respb)
		if err != nil {
			Dld.Sugar().Infof("[%v] [%v] Marshal-2 err: %v", b.FileId, filefullname, err)
			return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
		}
		Dld.Sugar().Infof("[%v] Download successful", b.FileId)
		return &RespBody{Code: 200, Msg: "success", Data: protob}, nil
	}
	Dld.Sugar().Infof("[%v] Download failed", b.FileId)
	return &RespBody{Code: 500, Msg: "fail", Data: nil}, nil
}

type RespSpacetagInfo struct {
	FileId string         `json:"fileId"`
	T      proof.FileTagT `json:"file_tag_t"`
	Sigmas [][]byte       `json:"sigmas"`
}
type RespSpacefileInfo struct {
	FileId     string `json:"fileId"`
	FileHash   string `json:"fileHash"`
	BlockTotal uint32 `json:"blockTotal"`
	BlockIndex uint32 `json:"blockIndex"`
	BlockData  []byte `json:"blockData"`
}

//
func (WService) SpacefileAction(body []byte) (proto.Message, error) {
	var (
		err error
		b   SpaceFileReq
	)
	t := tools.RandomInRange(100000000, 999999999)
	Out.Sugar().Infof("[%v]Receive space request", t)

	err = proto.Unmarshal(body, &b)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive space request err:%v", t, err)
		return &RespBody{Code: 400, Msg: err.Error(), Data: nil}, nil
	}

	mdata, code, err := chain.GetMinerDataOnChain(b.Acc)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive space request err:%v", t, err)
		return &RespBody{Code: int32(code), Msg: err.Error(), Data: nil}, nil
	}
	if len(mdata.Publickey) == 0 {
		Out.Sugar().Infof("[%v]Receive space request err:%v", t, err)
		return &RespBody{Code: 500, Msg: "internal err", Data: nil}, nil
	}
	pubkey, err := encryption.ParsePublicKey(mdata.Publickey)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive space request err:%v", t, err)
		return &RespBody{Code: 400, Msg: err.Error(), Data: nil}, nil
	}
	ok := encryption.VerifySign([]byte(b.Acc), b.Sign, pubkey)
	if !ok {
		Out.Sugar().Infof("[%v]Receive space request err: Invalid signature", t)
		return &RespBody{Code: 403, Msg: "Invalid signature", Data: nil}, nil
	}

	filebasedir := filepath.Join(configs.SpaceCacheDir, base58.Encode([]byte(b.Acc)))
	_, err = os.Stat(filebasedir)
	if err != nil {
		err = os.MkdirAll(filebasedir, os.ModeDir)
		if err != nil {
			Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
			return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
		}
	}

	if b.Fileid != "" {
		filefullpath := filepath.Join(filebasedir, b.Fileid)
		fi, err := os.Stat(filefullpath)
		if err == nil {
			var respfile RespSpacefileInfo
			respfile.FileId = b.Fileid
			respfile.BlockIndex = b.BlockIndex
			f, err := os.OpenFile(filefullpath, os.O_RDONLY, os.ModePerm)
			if err != nil {
				Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
				os.Remove(filefullpath)
				return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
			}

			blockTotal := fi.Size() / configs.RpcFileBuffer
			respfile.BlockTotal = uint32(blockTotal)
			if b.BlockIndex >= uint32(blockTotal) {
				f.Close()
				Out.Sugar().Infof("[%v]Receive space request err: Invalid block index", t)
				return &RespBody{Code: 400, Msg: "Invalid block index", Data: nil}, nil
			}
			offset, err := f.Seek(int64(b.BlockIndex*configs.RpcFileBuffer), 0)
			if err != nil {
				f.Close()
				Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
				return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
			}
			var buf = make([]byte, configs.RpcFileBuffer)
			_, err = f.ReadAt(buf, offset)
			if err != nil {
				f.Close()
				os.Remove(filefullpath)
				Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
				return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
			}
			f.Close()
			respfile.FileHash = ""
			respfile.BlockData = buf
			if b.BlockIndex+1 == uint32(blockTotal) {
				hash, err := tools.CalcFileHash(filefullpath)
				if err != nil {
					os.Remove(filefullpath)
					Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
					return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
				}
				respfile.FileHash = hash
			}
			respfile_b, err := json.Marshal(respfile)
			if err != nil {
				os.Remove(filefullpath)
				Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
				return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
			}
			return &RespBody{Code: 200, Msg: "Invalid block index", Data: respfile_b}, nil
		}
	}
	if b.SizeMb > 32 || b.SizeMb == 0 {
		Out.Sugar().Infof("[%v]Receive space request err: SizeMb up to 32 and not 0", t)
		return &RespBody{Code: 400, Msg: "SizeMb up to 32 and not 0", Data: nil}, nil
	}

	lines := b.SizeMb * 1024 * 1024 / configs.LengthOfALine
	filename := fmt.Sprintf("%v_%d%d%d", b.Acc[len(b.Acc)-5:], tools.RandomInRange(1000, 9999), time.Now().Unix(), tools.RandomInRange(1000, 9999))
	filefullpath := filepath.Join(filebasedir, filename)
	f, err := os.OpenFile(filefullpath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.ModePerm)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	}
	var i uint32 = 0
	for i = 0; i < lines; i++ {
		f.WriteString(tools.RandStr(configs.LengthOfALine - 1))
		f.WriteString("\n")
	}
	err = f.Sync()
	if err != nil {
		Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
		f.Close()
		os.Remove(filefullpath)
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	}
	f.Close()

	fi, err := os.Stat(filefullpath)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
		os.Remove(filefullpath)
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	}

	var respfile RespSpacefileInfo
	respfile.FileId = filename
	respfile.BlockIndex = 0
	respfile.FileHash = ""
	blockTotal := fi.Size() / configs.RpcFileBuffer
	respfile.BlockTotal = uint32(blockTotal)
	f, err = os.OpenFile(filefullpath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
		os.Remove(filefullpath)
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	}

	var buf = make([]byte, configs.RpcFileBuffer)
	_, err = f.Read(buf)
	if err != nil {
		f.Close()
		os.Remove(filefullpath)
		Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	}
	respfile.BlockData = buf
	respfile_b, err := json.Marshal(respfile)
	if err != nil {
		f.Close()
		os.Remove(filefullpath)
		Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	}
	f.Close()
	fmt.Println("--> A new file: ", filefullpath)
	return &RespBody{Code: 200, Msg: "success", Data: respfile_b}, nil
}

//
func (WService) SpacetagAction(body []byte) (proto.Message, error) {
	var (
		err error
		b   SpaceTagReq
	)
	t := tools.RandomInRange(100000000, 999999999)
	Out.Sugar().Infof("[%v]Receive space tag request", t)
	err = proto.Unmarshal(body, &b)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive space request err:%v", t, err)
		return &RespBody{Code: 400, Msg: err.Error(), Data: nil}, nil
	}
	mdata, code, err := chain.GetMinerDataOnChain(b.Acc)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive space request err:%v", t, err)
		return &RespBody{Code: int32(code), Msg: err.Error(), Data: nil}, nil
	}
	pubkey, err := encryption.ParsePublicKey(mdata.Publickey)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive space request err:%v", t, err)
		return &RespBody{Code: 400, Msg: err.Error(), Data: nil}, nil
	}
	ok := encryption.VerifySign([]byte(b.Acc), b.Sign, pubkey)
	if !ok {
		Out.Sugar().Infof("[%v]Receive space request err: Invalid signature", t)
		return &RespBody{Code: 403, Msg: "Invalid signature", Data: nil}, nil
	}

	filebasedir := filepath.Join(configs.SpaceCacheDir, base58.Encode([]byte(b.Acc)))
	_, err = os.Stat(filebasedir)
	if err != nil {
		err = os.MkdirAll(filebasedir, os.ModeDir)
		if err != nil {
			Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
			return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
		}
	}
	filefullpath := filepath.Join(filebasedir, b.Fileid)
	_, err = os.Stat(filefullpath)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
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
			runtime.UnlockOSThread()
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
		Out.Sugar().Infof("[%v]Receive space request err: PoDR2ProofCommit", t)
		return &RespBody{Code: 500, Msg: "unexpected system error", Data: nil}, nil
	}

	var resp RespSpacetagInfo
	resp.FileId = b.Fileid
	resp.T = commitResponse.T
	resp.Sigmas = commitResponse.Sigmas
	resp_b, err := json.Marshal(resp)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	}
	fmt.Println("--> File tag: ", b.Fileid)
	return &RespBody{Code: 200, Msg: "success", Data: resp_b}, nil
}

//
func (WService) FilebackAction(body []byte) (proto.Message, error) {
	var (
		err error
		b   FileBackReq
	)
	t := tools.RandomInRange(100000000, 999999999)
	Out.Sugar().Infof("[%v]Receive space tag request", t)
	err = proto.Unmarshal(body, &b)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive space request err:%v", t, err)
		return &RespBody{Code: 400, Msg: err.Error(), Data: nil}, nil
	}
	mdata, code, err := chain.GetMinerDataOnChain(b.Acc)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive space request err:%v", t, err)
		return &RespBody{Code: int32(code), Msg: err.Error(), Data: nil}, nil
	}
	pubkey, err := encryption.ParsePublicKey(mdata.Publickey)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive space request err:%v", t, err)
		return &RespBody{Code: 400, Msg: err.Error(), Data: nil}, nil
	}
	ok := encryption.VerifySign([]byte(b.Acc), b.Sign, pubkey)
	if !ok {
		Out.Sugar().Infof("[%v]Receive space request err: Invalid signature", t)
		return &RespBody{Code: 403, Msg: "Invalid signature", Data: nil}, nil
	}
	filebasedir := filepath.Join(configs.SpaceCacheDir, base58.Encode([]byte(b.Acc)))
	_, err = os.Stat(filebasedir)
	if err != nil {
		err = os.MkdirAll(filebasedir, os.ModeDir)
		if err != nil {
			Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
			return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
		}
	}
	filefullpath := filepath.Join(filebasedir, b.Fileid)
	fi, err := os.Stat(filefullpath)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	}
	// up-chain meta info
	var metainfo = make([]chain.SpaceFileInfo, 1)
	metainfo[0].FileId = []byte(b.Fileid)
	metainfo[0].FileHash = []byte(b.Filehash)
	metainfo[0].FileSize = types.U64(uint64(fi.Size()))
	var pre []byte
	if configs.NewTestAddr {
		pre = tools.ChainCessTestPrefix
	} else {
		pre = tools.SubstratePrefix
	}
	wal, err := tools.DecodeToPub(b.Acc, pre)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	}
	metainfo[0].Acc = types.NewAccountID(wal)
	metainfo[0].MinerId = mdata.Peerid
	// f, err := os.Open(filefullpath)
	// if err != nil {
	// 	Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
	// 	return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	// }
	_, n, err := tools.Split(filefullpath, configs.BlockSize, fi.Size())
	if err != nil {
		Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	}
	metainfo[0].BlockNum = types.U32(n)
	metainfo[0].ScanSize = types.U32(uint32(configs.ScanBlockSize))
	var file_blocks = make([]chain.BlockInfo, n)
	for i := uint64(1); i <= n; i++ {
		file_blocks[i-1].BlockIndex, _ = tools.IntegerToBytes(uint32(i))
		file_blocks[i-1].BlockSize = types.U32(configs.BlockSize)
	}
	metainfo[0].BlockInfo = file_blocks

	txhash, code, err := chain.PutSpaceTagInfoToChain(
		configs.C.CtrlPrk,
		mdata.Peerid,
		metainfo,
	)
	os.Remove(filefullpath)
	Out.Sugar().Infof("[T:%v][%v][%v]File meta on chain", t, b.Fileid, txhash)
	fmt.Println("--> File meta on chain: ", b.Fileid, " ", code, " ", txhash)
	return &RespBody{Code: int32(code), Msg: "Check status code", Data: []byte(txhash)}, nil
}

// Combine the file segments uploaded by the client into a complete file.
func combinationFile(fid, dir string, num int32) (string, error) {
	completefile := filepath.Join(dir, fid+".cess")
	cf, err := os.OpenFile(completefile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC|os.O_APPEND, os.ModePerm)
	if err != nil {
		return completefile, err
	}
	defer cf.Close()
	for i := int32(1); i <= num; i++ {
		path := filepath.Join(dir, fid+"_"+strconv.Itoa(int(i)))
		f, err := os.Open(path)
		if err != nil {
			return completefile, err
		}
		defer f.Close()
		b, err := ioutil.ReadAll(f)
		if err != nil {
			return completefile, err
		}
		cf.Write(b)
	}
	err = cf.Sync()
	if err != nil {
		return completefile, err
	}
	return completefile, nil
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

// processingfile is used to process all copies of the file and the corresponding tag information
// func processingfile(t int, fid, dir string, duplnamelist, duplkeynamelist []string) {
// 	var (
// 		err      error
// 		code     int
// 		mDatas   = make([]chain.CessChain_AllMinerInfo, 0)
// 		filedump = make([]chain.FileDuplicateInfo, len(duplnamelist))
// 		mips     = make([]string, len(duplnamelist))
// 	)

// 	Out.Sugar().Infof("[T:%v] [%v] Start assigning files to miners", t, fid)

// 	for len(mDatas) != 0 {
// 		mDatas, _, _ = chain.GetAllMinerDataOnChain()
// 		if len(mDatas) == 0 {
// 			time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
// 		}
// 	}

// 	for i := 0; i < len(duplnamelist); i++ {
// 		duplname := filepath.Base(duplnamelist[i])
// 		fi, err := os.Stat(duplnamelist[i])
// 		if err != nil {
// 			Out.Sugar().Infof("[T:%v] [%v] os.Stat err: ", t, duplnamelist[i], err)
// 			continue
// 		}
// 		f, err := os.OpenFile(duplnamelist[i], os.O_RDONLY, os.ModePerm)
// 		if err != nil {
// 			Out.Sugar().Infof("[T:%v] [%v] os.OpenFile err: ", t, duplnamelist[i], err)
// 			continue
// 		}
// 		blockTotal := fi.Size() / configs.RpcFileBuffer
// 		if fi.Size()%configs.RpcFileBuffer != 0 {
// 			blockTotal += 1
// 		}

// 		var failminer = make(map[uint64]bool, 0)
// 		var index int = 0
// 		var mip = ""
// 		var n int
// 		for j := int64(0); j < blockTotal; j++ {
// 			var buf = make([]byte, configs.RpcFileBuffer)
// 			if j+1 == blockTotal {
// 				n, _ = f.Read(buf)
// 			} else {
// 				n, err = f.ReadAt(buf, int64(j*configs.RpcFileBuffer))
// 				if err != nil {
// 					Out.Sugar().Infof("[T:%v] [%v] [%v] [%v] f.ReadAt err: %v", t, duplnamelist[i], fi.Size(), j*configs.RpcFileBuffer, err)
// 					continue
// 				}
// 			}

// 			var bo = PutFileToBucket{
// 				FileId:     duplname,
// 				FileHash:   "",
// 				BlockTotal: uint32(blockTotal),
// 				BlockSize:  uint32(n),
// 				BlockIndex: uint32(j),
// 				BlockData:  buf[:n],
// 			}
// 			bob, err := proto.Marshal(&bo)
// 			if err != nil {
// 				Out.Sugar().Infof("[T:%v] [%v] Marshal err: %v", t, duplnamelist[i], err)
// 				continue
// 			}
// 			for {
// 				if mip == "" {
// 					index = tools.RandomInRange(0, len(mDatas))
// 					_, ok := failminer[uint64(mDatas[index].Peerid)]
// 					if ok {
// 						continue
// 					}
// 					_, err = WriteData(string(mDatas[index].Ip), configs.RpcService_Miner, configs.RpcMethod_Miner_WriteFile, bob)
// 					if err == nil {
// 						mip = string(mDatas[index].Ip)
// 						Out.Sugar().Infof("[T:%v] [%v-%v] [%v] WriteData suc: %v", t, duplnamelist[i], j, mip)
// 						break
// 					}
// 					failminer[uint64(mDatas[index].Peerid)] = true
// 				} else {
// 					_, err = WriteData(mip, configs.RpcService_Miner, configs.RpcMethod_Miner_WriteFile, bob)
// 					if err != nil {
// 						failminer[uint64(mDatas[index].Peerid)] = true
// 						time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
// 						continue
// 					}
// 					break
// 				}
// 			}
// 		}

// 		filedump[i].DuplId = types.Bytes([]byte(duplname))
// 		key := filepath.Base(duplkeynamelist[i])
// 		sufffex := filepath.Ext(key)
// 		strings.TrimSuffix(key, sufffex)
// 		filedump[i].RandKey = types.Bytes([]byte(strings.TrimSuffix(key, sufffex)))
// 		filedump[i].MinerId = mDatas[index].Peerid
// 		filedump[i].MinerIp = mDatas[index].Ip
// 		bs, sbs := CalcFileBlockSizeAndScanSize(fi.Size())
// 		filedump[i].ScanSize = types.U32(sbs)
// 		mips[i] = string(mDatas[index].Ip)
// 		// Query miner information by id
// 		var mdetails chain.Chain_MinerDetails
// 		for {
// 			mdetails, _, err = chain.GetMinerDetailsById(uint64(mDatas[index].Peerid))
// 			if err != nil {
// 				Err.Sugar().Errorf("[%v][%v][%v]", t, duplnamelist[i], err)
// 				time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
// 				continue
// 			}
// 			break
// 		}
// 		filedump[i].Acc = mdetails.Address
// 		// fmt.Println(fi.Size())
// 		// fmt.Println(bs)
// 		matrix, _, n, err := tools.Split(f, bs)
// 		if err != nil {
// 			f.Close()
// 			Err.Sugar().Errorf("[%v][%v][%v]", t, duplnamelist[i], err)
// 			continue
// 		}
// 		f.Close()
// 		filedump[i].BlockNum = types.U32(uint32(n))
// 		var blockinfo = make([]chain.BlockInfo, n)
// 		for x := uint64(1); x <= n; x++ {
// 			blockinfo[x-1].BlockIndex, _ = tools.IntegerToBytes(uint32(x))
// 			blockinfo[x-1].BlockSize = types.U32(uint32(len(matrix[x-1])))
// 		}
// 		filedump[i].BlockInfo = blockinfo
// 	}

// 	// calculate file tag info
// 	for i := 0; i < len(filedump); i++ {
// 		var PoDR2commit proof.PoDR2Commit
// 		var commitResponse proof.PoDR2CommitResponse
// 		PoDR2commit.FilePath = duplnamelist[i]
// 		fs, err := os.Stat(duplnamelist[i])
// 		if err != nil {
// 			Err.Sugar().Errorf("[%v][%v][%v]", t, duplnamelist[i], err)
// 			continue
// 		}
// 		bs, sbs := CalcFileBlockSizeAndScanSize(fs.Size())
// 		PoDR2commit.BlockSize = bs
// 		commitResponseCh, err := PoDR2commit.PoDR2ProofCommit(proof.Key_Ssk, string(proof.Key_SharedParams), sbs)
// 		if err != nil {
// 			Err.Sugar().Errorf("[%v][%v][%v]", t, filedump[i], err)
// 			continue
// 		}
// 		select {
// 		case commitResponse = <-commitResponseCh:
// 		}
// 		if commitResponse.StatueMsg.StatusCode != proof.Success {
// 			Err.Sugar().Errorf("[%v][%v][%v]", t, filedump[i], err)
// 			continue
// 		}
// 		var resp PutTagToBucket
// 		resp.FileId = string(filedump[i].DuplId)
// 		resp.Name = commitResponse.T.Name
// 		resp.N = commitResponse.T.N
// 		resp.U = commitResponse.T.U
// 		resp.Signature = commitResponse.T.Signature
// 		resp.Sigmas = commitResponse.Sigmas
// 		resp_proto, err := proto.Marshal(&resp)
// 		if err != nil {
// 			Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
// 			continue
// 		}

// 		_, err = WriteData(mips[i], configs.RpcService_Miner, configs.RpcMethod_Miner_WriteFileTag, resp_proto)
// 		if err != nil {
// 			Err.Sugar().Errorf("[%v][%v][%v]%v", t, mips[i], duplnamelist[i], err)
// 			time.Sleep(time.Second * time.Duration(tools.RandomInRange(2, 5)))
// 			continue
// 		}
// 	}

// 	// Upload the file meta information to the chain and write it to the cache
// 	for {
// 		ok, err := chain.PutMetaInfoToChain(configs.C.CtrlPrk, fid, filedump)
// 		if !ok || err != nil {
// 			Err.Sugar().Errorf("[%v][%v][%v]", t, fid, err)
// 			time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
// 			continue
// 		}
// 		Out.Sugar().Infof("[%v][%v]File metainfo up chain success", t, fid)
// 		c, err := cache.GetCache()
// 		if err != nil {
// 			Err.Sugar().Errorf("[%v][%v][%v]", t, fid, err)
// 		} else {
// 			b, err := json.Marshal(filedump)
// 			if err != nil {
// 				Err.Sugar().Errorf("[%v][%v][%v]", t, fid, err)
// 			} else {
// 				err = c.Put([]byte(fid), b)
// 				if err != nil {
// 					Err.Sugar().Errorf("[%v][%v][%v]", t, fid, err)
// 				} else {
// 					Out.Sugar().Infof("[%v][%v]File metainfo write cache success", t, fid)
// 				}
// 			}
// 		}
// 		break
// 	}
// }

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

func cutDataRule(size uint64) (uint64, uint64, uint8, error) {
	if size <= 0 {
		return 0, 0, 0, errors.New("file size is 0")
	}
	num := size / configs.RpcFileBuffer
	slicesize := size / (num + 1)
	tailsize := size - slicesize*(num+1)
	return uint64(slicesize), uint64(slicesize + tailsize), uint8(num) + 1, nil
}

// processingfile is used to process all copies of the file and the corresponding tag information
func backupFile(ch chan uint8, num int, fid, fileFullPath, duplkeyname string) {
	var (
		err    error
		mDatas = make([]chain.CessChain_AllMinerInfo, 0)
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

	Uld.Sugar().Infof("[%v] Prepare to store the %vrd replica", fid, num)

	for len(mDatas) == 0 {
		mDatas, _, err = chain.GetAllMinerDataOnChain()
		if err != nil {
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
		}
	}

	Uld.Sugar().Infof("[%v] %v miners found", fid, len(mDatas))
	for i := 0; i < len(mDatas); i++ {
		Uld.Sugar().Infof("[%v] %v: %v", fid, i, string(mDatas[i].Ip))
	}

	duplname := filepath.Base(fileFullPath)
	fstat, err := os.Stat(fileFullPath)
	if err != nil {
		ch <- 3
		Uld.Sugar().Infof("[%v] [%v] The copy was deleted and cannot be stored: %v", fid, fileFullPath, err)
		return
	}

	f, err := os.OpenFile(fileFullPath, os.O_RDONLY, os.ModePerm)
	if err != nil {
		ch <- 3
		Uld.Sugar().Infof("[%v] [%v] Failed to read replica file: %v", fid, fileFullPath, err)
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
	for j := int64(0); j < blockTotal; j++ {
		var buf = make([]byte, configs.RpcFileBuffer)
		f.Seek(j*configs.RpcFileBuffer, 0)
		n, _ = f.Read(buf)

		var bo = PutFileToBucket{
			FileId:     duplname,
			FileHash:   "",
			BlockTotal: uint32(blockTotal),
			BlockSize:  uint32(n),
			BlockIndex: uint32(j),
			BlockData:  buf[:n],
		}
		bob, _ := proto.Marshal(&bo)
		if err != nil {
			ch <- 3
			Uld.Sugar().Infof("[%v] [%v] Marshal err: %v", fid, fileFullPath, err)
			return
		}
		var failcount uint8
		for {
			if mip == "" {
				index = tools.RandomInRange(0, len(mDatas))
				if _, ok := filedIndex[index]; ok {
					continue
				}
				dstip := "ws://" + string(base58.Decode(string(mDatas[index].Ip)))
				ctx, _ := context.WithTimeout(context.Background(), 6*time.Second)
				client, err = DialWebsocket(ctx, dstip, "")
				if err != nil {
					filedIndex[index] = struct{}{}
					continue
				}
				Uld.Sugar().Infof("[%v] [%v] [%v] connection suc", fid, fileFullPath, mip)
				_, err = WriteData2(client, configs.RpcService_Miner, configs.RpcMethod_Miner_WriteFile, bob)
				if err == nil {
					mip = string(mDatas[index].Ip)
					Uld.Sugar().Infof("[%v] [%v-%v] transfer suc", fid, fileFullPath, j)
					break
				}
				filedIndex[index] = struct{}{}
			} else {
				_, err = WriteData2(client, configs.RpcService_Miner, configs.RpcMethod_Miner_WriteFile, bob)
				if err != nil {
					failcount++
					if failcount >= 5 {
						ch <- 1
						Uld.Sugar().Infof("[%v] [%v-%v] transfer failed: %v", fid, fileFullPath, j)
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
	var filedump = make([]chain.FileDuplicateInfo, 1)

	filedump[0].DuplId = types.Bytes([]byte(duplname))
	key := filepath.Base(duplkeyname)
	sufffex := filepath.Ext(key)
	filedump[0].RandKey = types.Bytes([]byte(strings.TrimSuffix(key, sufffex)))
	filedump[0].MinerId = mDatas[index].Peerid
	filedump[0].MinerIp = mDatas[index].Ip
	bs, sbs := CalcFileBlockSizeAndScanSize(fstat.Size())
	filedump[0].ScanSize = types.U32(sbs)

	// Query miner information by id
	var mdetails chain.Chain_MinerDetails
	for {
		mdetails, _, err = chain.GetMinerDetailsById(uint64(mDatas[index].Peerid))
		if err != nil {
			Uld.Sugar().Infof("[%v] [%v] [%v] GetMinerDetailsById err: %v", fid, mDatas[index].Peerid, fileFullPath, err)
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
			continue
		}
		break
	}
	filedump[0].Acc = mdetails.Address
	matrix, blocknum, err := tools.Split(fileFullPath, bs, fstat.Size())
	if err != nil {
		ch <- 1
		Uld.Sugar().Infof("[%v] [%v] [%v] [%v] Split err: %v", fid, fileFullPath, fstat.Size(), bs, err)
		return
	}

	filedump[0].BlockNum = types.U32(uint32(blocknum))
	var blockinfo = make([]chain.BlockInfo, blocknum)
	for x := uint64(1); x <= blocknum; x++ {
		blockinfo[x-1].BlockIndex, _ = tools.IntegerToBytes(uint32(x))
		blockinfo[x-1].BlockSize = types.U32(uint32(len(matrix[x-1])))
	}
	filedump[0].BlockInfo = blockinfo

	// calculate file tag info
	for i := 0; i < len(filedump); i++ {
		var PoDR2commit proof.PoDR2Commit
		var commitResponse proof.PoDR2CommitResponse
		PoDR2commit.FilePath = fileFullPath
		bs, sbs := CalcFileBlockSizeAndScanSize(fstat.Size())
		PoDR2commit.BlockSize = bs
		commitResponseCh, err := PoDR2commit.PoDR2ProofCommit(proof.Key_Ssk, string(proof.Key_SharedParams), sbs)
		if err != nil {
			ch <- 1
			Uld.Sugar().Infof("[%v] [%v] [%v] PoDR2ProofCommit err: %v", fid, fileFullPath, sbs, err)
			return
		}
		select {
		case commitResponse = <-commitResponseCh:
		}
		if commitResponse.StatueMsg.StatusCode != proof.Success {
			ch <- 1
			Uld.Sugar().Infof("[%v] [%v] [%v] PoDR2ProofCommit failed", fid, fileFullPath, sbs)
			return
		}
		var resp PutTagToBucket
		resp.FileId = string(filedump[i].DuplId)
		resp.Name = commitResponse.T.Name
		resp.N = commitResponse.T.N
		resp.U = commitResponse.T.U
		resp.Signature = commitResponse.T.Signature
		resp.Sigmas = commitResponse.Sigmas
		resp_proto, err := proto.Marshal(&resp)
		if err != nil {
			ch <- 1
			Uld.Sugar().Infof("[%v] [%v] Marshal resp err: %v", fid, fileFullPath, err)
			return
		}

		_, err = WriteData2(client, configs.RpcService_Miner, configs.RpcMethod_Miner_WriteFileTag, resp_proto)
		if err != nil {
			ch <- 1
			Uld.Sugar().Infof("[%v] [%v] [%v] WriteData2 tag err: %v", fid, fileFullPath, mip, err)
			return
		}
	}

	// Upload the file meta information to the chain and write it to the cache
	for {
		ok, err := chain.PutMetaInfoToChain(configs.C.CtrlPrk, fid, filedump)
		if !ok || err != nil {
			Uld.Sugar().Infof("[%v] [%v] PutMetaInfoToChain err: %v", fid, fileFullPath, err)
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
			continue
		}
		Uld.Sugar().Infof("[%v] The metadata of the %v replica was successfully uploaded to the chain", fid, num)
		// c, err := cache.GetCache()
		// if err != nil {
		// 	Err.Sugar().Errorf("[%v][%v][%v]", t, fid, err)
		// } else {
		// 	b, err := json.Marshal(filedump)
		// 	if err != nil {
		// 		Err.Sugar().Errorf("[%v][%v][%v]", t, fid, err)
		// 	} else {
		// 		err = c.Put([]byte(fid), b)
		// 		if err != nil {
		// 			Err.Sugar().Errorf("[%v][%v][%v]", t, fid, err)
		// 		} else {
		// 			Out.Sugar().Infof("[%v][%v]File metainfo write cache success", t, fid)
		// 		}
		// 	}
		// }
		ch <- 2
		break
	}
}
