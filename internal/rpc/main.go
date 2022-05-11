package rpc

import (
	"cess-scheduler/configs"
	"cess-scheduler/internal/cache"
	"cess-scheduler/internal/chain"
	"cess-scheduler/internal/encryption"
	"cess-scheduler/internal/fileshards"
	. "cess-scheduler/internal/logger"
	proof "cess-scheduler/internal/proof/apiv1"
	"cess-scheduler/tools"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	. "cess-scheduler/internal/rpc/protobuf"
	rpc "cess-scheduler/internal/rpc/protobuf"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/golang/protobuf/proto"
	"storj.io/common/base58"
)

type WService struct {
}

// Start tcp service.
// If an error occurs, it will exit immediately.
func Rpc_Main() {
	srv := NewServer()
	srv.Register(configs.RpcService_Scheduler, WService{})
	err := http.ListenAndServe(":"+configs.Confile.SchedulerInfo.ServicePort, srv.WebsocketHandler([]string{"*"}))
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
		err       error
		t         int64
		cachepath string
		b         FileUploadInfo
		fmeta     chain.FileMetaInfo
	)
	t = time.Now().Unix()
	Out.Sugar().Infof("[%v]Receive upload request", t)
	err = proto.Unmarshal(body, &b)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive upload request err:%v", t, err)
		return &RespBody{Code: 400, Msg: err.Error(), Data: nil}, nil
	}

	Out.Sugar().Infof("[%v]Receive client upload request:[%v][%v][%v]", t, b.FileId, b.Blocks, len(b.Data))
	err = tools.CreatDirIfNotExist(configs.FileCacheDir)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive upload request err:%v", t, err)
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	}

	fmeta, code, err := chain.GetFileMetaInfoOnChain(b.FileId)
	if err != nil {
		Err.Sugar().Errorf("[%v][%v-%v]Failed to query file metadata.", t, b.FileId, b.FileHash)
		return &RespBody{Code: int32(code), Msg: err.Error(), Data: nil}, nil
	}

	if fmeta.FileDupl != nil {
		return &RespBody{Code: 400, Msg: "Your fileid already exists", Data: nil}, nil
	}

	cachepath = filepath.Join(configs.FileCacheDir, b.FileId)
	_, err = os.Stat(cachepath)
	if err != nil {
		if fmeta.FileSize > 0 {
			err = os.MkdirAll(cachepath, os.ModeDir)
			if err != nil {
				Err.Sugar().Errorf("[%v][%v-%v]%v", t, b.FileId, b.Backups, err)
				return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
			}
		} else {
			Err.Sugar().Errorf("[%v][%v-%v]%v", t, b.FileId, b.Backups, err)
			return &RespBody{Code: 400, Msg: "invalid file", Data: nil}, nil
		}
		// if string(fmeta.FileHash) == b.FileHash {
		// 	err = os.MkdirAll(cachepath, os.ModeDir)
		// 	if err != nil {
		// 		Err.Sugar().Errorf("[%v][%v-%v]%v", t, b.FileId, b.Backups, err)
		// 		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
		// 	}
		// } else {
		// 	Err.Sugar().Errorf("[%v][%v-%v]%v", t, b.FileId, b.Backups, err)
		// 	return &RespBody{Code: 400, Msg: "file hash error", Data: nil}, nil
		// }
	} else {
		filename0 := filepath.Join(cachepath, b.FileId+".d0")
		filename1 := filepath.Join(cachepath, b.FileId+".d1")
		filename2 := filepath.Join(cachepath, b.FileId+".d2")
		filenamecess := filepath.Join(cachepath, b.FileId+".cess")
		if _, err = os.Stat(filename0); err == nil {
			return &RespBody{Code: 400, Msg: "Your fileid already exists", Data: nil}, nil
		}
		if _, err = os.Stat(filename1); err == nil {
			return &RespBody{Code: 400, Msg: "Your fileid already exists", Data: nil}, nil
		}
		if _, err = os.Stat(filename2); err == nil {
			return &RespBody{Code: 400, Msg: "Your fileid already exists", Data: nil}, nil
		}
		if _, err = os.Stat(filenamecess); err == nil {
			return &RespBody{Code: 400, Msg: "Your fileid already exists", Data: nil}, nil
		}
	}

	filename := filepath.Join(cachepath, b.FileId+"_"+fmt.Sprintf("%d", b.BlockNum))
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.ModePerm)
	if err != nil {
		Err.Sugar().Errorf("[%v][%v-%v]%v", t, b.FileId, b.FileHash, err)
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	}
	_, err = f.Write(b.Data)
	if err != nil {
		Err.Sugar().Errorf("[%v][%v-%v]%v", t, b.FileId, b.FileHash, err)
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	}
	err = f.Sync()
	if err != nil {
		Err.Sugar().Errorf("[%v][%v-%v]%v", t, b.FileId, b.FileHash, err)
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	}
	f.Close()
	if b.BlockNum == b.Blocks {
		completefile, err := combinationFile(b.FileId, cachepath, b.Blocks)
		if err != nil {
			os.Remove(completefile)
			Err.Sugar().Errorf("[%v]%v,%v,%v Incomplete chunking of file", t, b.FileId, b.Blocks, err)
			return &RespBody{Code: 400, Msg: "Incomplete chunking of file", Data: nil}, nil
		}
		// delete file segments
		for i := 1; i <= int(b.Blocks); i++ {
			path := filepath.Join(cachepath, b.FileId+"_"+strconv.Itoa(int(i)))
			os.Remove(path)
		}

		backupNum := uint8(3)
		if backupNum < uint8(fmeta.Backups) {
			backupNum = uint8(fmeta.Backups)
		}
		if backupNum > uint8(6) {
			backupNum = uint8(6)
		}
		buf, err := os.ReadFile(completefile)
		if err != nil {
			os.Remove(completefile)
			Err.Sugar().Errorf("[%v]%v,%v,%v", t, b.FileId, b.Blocks, err)
			return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
		}
		var duplnamelist = make([]string, 0)
		var duplkeynamelist = make([]string, 0)
		for i := uint8(0); i < backupNum; {
			// Generate 32-bit random key for aes encryption
			key := tools.GetRandomkey(32)
			key_base58 := tools.Base58Encoding(key)
			// Aes ctr mode encryption
			encrypted, err := encryption.AesCtrEncrypt(buf, []byte(key), []byte(key_base58[:16]))
			if err != nil {
				Err.Sugar().Errorf("[%v][%v][%v]", t, completefile, err)
				continue
			}
			duplname := b.FileId + ".d" + strconv.Itoa(int(i))
			//filedump[i].DuplId = types.Bytes([]byte(duplname))
			//filedump[i].RandKey = types.Bytes([]byte(key_base58))

			duplFallpath := filepath.Join(cachepath, duplname)
			duplf, err := os.OpenFile(duplFallpath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.ModePerm)
			if err != nil {
				Err.Sugar().Errorf("[%v][%v][%v]", t, completefile, err)
				continue
			}
			_, err = duplf.Write(encrypted)
			if err != nil {
				Err.Sugar().Errorf("[%v][%v][%v]", t, completefile, err)
				duplf.Close()
				os.Remove(duplFallpath)
				continue
			}
			err = duplf.Sync()
			if err != nil {
				Err.Sugar().Errorf("[%v][%v][%v]", t, completefile, err)
				duplf.Close()
				os.Remove(duplFallpath)
				continue
			}
			duplf.Close()
			duplkey := key_base58 + ".k" + strconv.Itoa(int(i))
			duplkeyFallpath := filepath.Join(cachepath, duplkey)
			_, err = os.Create(duplkeyFallpath)
			if err != nil {
				os.Remove(duplFallpath)
			} else {
				duplnamelist = append(duplnamelist, duplFallpath)
				duplkeynamelist = append(duplkeynamelist, duplkeyFallpath)
				i++
			}
		}
		os.Remove(completefile)
		go processingfile(t, b.FileId, cachepath, duplnamelist, duplkeynamelist)
		Out.Sugar().Infof("[%v][%v-%v-%v]success", t, b.FileId, b.Blocks, b.BlockNum)
	}
	Out.Sugar().Infof("[%v][%v-%v]success", t, b.FileId, b.Blocks)
	return &RespBody{Code: 200, Msg: "success", Data: nil}, nil
}

// ReadfileAction is used to handle client requests to download files.
// The return code is 0 for success, non-0 for failure.
// The returned Msg indicates the result reason.
func (WService) ReadfileAction(body []byte) (proto.Message, error) {
	var (
		err   error
		t     int64
		code  int
		b     FileDownloadReq
		fmeta chain.FileMetaInfo
	)
	t = time.Now().Unix()
	Out.Sugar().Infof("[%v]Receive download request", t)
	err = proto.Unmarshal(body, &b)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive download request err:%v", t, err)
		return &RespBody{Code: 400, Msg: err.Error(), Data: nil}, nil
	}
	//Query file meta information
	// c, err := cache.GetCache()
	// if err != nil {
	// 	Err.Sugar().Errorf("[%v][%v-%v]%v", t, b.FileId, b.Blocks, err)
	// } else {
	// 	cachedata, err := c.Get([]byte(b.FileId))
	// 	if err == nil {
	// 		err = json.Unmarshal(cachedata, &fmeta.FileDupl)
	// 		if err != nil {
	// 			Err.Sugar().Errorf("[%v][%v-%v]%v", t, b.FileId, b.Blocks, err)
	// 		}
	// 	}
	// }
	if fmeta.FileDupl == nil {
		fmeta, code, err = chain.GetFileMetaInfoOnChain(b.FileId)
		if err != nil {
			Err.Sugar().Errorf("[%v][%v-%v]%v", t, b.FileId, b.Blocks, err)
			return &RespBody{Code: int32(code), Msg: err.Error(), Data: nil}, nil
		}
		if string(fmeta.FileState) != "active" {
			Err.Sugar().Errorf("[%v]Download prohibited", b.FileId)
			return &RespBody{Code: 403, Msg: "Download prohibited"}, nil
		}
	}
	// Determine whether the user has download permission
	// a, err := types.NewAddressFromHexAccountID(b.WalletAddress)
	// if err != nil {
	// 	Err.Sugar().Errorf("[%v]%v", b.FileId, err)
	// 	return &RespBody{Code: 400, Msg: "invalid wallet address"}, nil
	// }
	// if a.AsAccountID != fmeta.UserAddr {
	// 	Err.Sugar().Errorf("[%v]No permission", b.FileId)
	// 	return &RespBody{Code: 400, Msg: "No permission"}, nil
	// }

	path := filepath.Join(configs.FileCacheDir, b.FileId)
	_, err = os.Stat(path)
	if err != nil {
		os.MkdirAll(path, os.ModeDir)
	}
	filefullname := filepath.Join(path, b.FileId+".u")
	_, err = os.Stat(filefullname)
	if err != nil {
		// file not exist, query dupl file
		for i := 0; i < len(fmeta.FileDupl); i++ {
			duplname := filepath.Join(path, b.FileId+".d"+strconv.Itoa(i))
			_, err = os.Stat(duplname)
			if err == nil {
				buf, err := ioutil.ReadFile(duplname)
				if err != nil {
					Err.Sugar().Errorf("[%v][%v]%v", t, duplname, err)
					os.Remove(duplname)
					continue
				}
				//aes decryption
				ivkey := string(fmeta.FileDupl[i].RandKey)[:16]
				bkey := tools.Base58Decoding(string(fmeta.FileDupl[i].RandKey))
				decrypted, err := encryption.AesCtrDecrypt(buf, []byte(bkey), []byte(ivkey))
				if err != nil {
					Err.Sugar().Errorf("[%v][%v]%v", t, duplname, err)
					os.Remove(duplname)
					continue
				}
				fu, err := os.OpenFile(filefullname, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
				if err != nil {
					Err.Sugar().Errorf("[%v][%v][%v]%v", t, b.FileId, string(fmeta.FileDupl[i].DuplId), err)
					continue
				}
				fu.Write(decrypted)
				err = fu.Sync()
				if err != nil {
					Err.Sugar().Errorf("[%v][%v]%v", t, duplname, err)
					fu.Close()
					os.Remove(filefullname)
					continue
				}
				fu.Close()
				break
			}
		}
	}

	_, err = os.Stat(filefullname)
	if err == nil {
		fuser, err := os.ReadFile(filefullname)
		if err != nil {
			Err.Sugar().Errorf("[%v][%v-%v]%v", t, b.FileId, b.Blocks, err)
			return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
		}
		slicesize, lastslicesize, num, err := fileshards.CutDataRule(uint64(len(fuser)))
		if err != nil {
			Err.Sugar().Errorf("[%v][%v-%v]%v", t, b.FileId, b.Blocks, err)
			return &RespBody{Code: 400, Msg: err.Error(), Data: nil}, nil
		}
		var tmp = make([]byte, 0)
		var blockSize int32
		if b.Blocks == int32(num) {
			tmp = fuser[uint64(len(fuser)-int(lastslicesize)):]
			blockSize = int32(lastslicesize)
		} else {
			tmp = fuser[uint64(uint64(b.Blocks-1)*slicesize):uint64(uint64(b.Blocks)*slicesize)]
			blockSize = int32(slicesize)
		}
		respb := &FileDownloadInfo{
			FileId:    b.FileId,
			Blocks:    b.Blocks,
			BlockSize: blockSize,
			BlockNum:  int32(num),
			Data:      tmp,
		}
		protob, err := proto.Marshal(respb)
		if err != nil {
			Err.Sugar().Errorf("[%v][%v-%v]%v", t, b.FileId, b.Blocks, err)
			return &RespBody{Code: 400, Msg: err.Error(), Data: nil}, nil
		}
		Out.Sugar().Infof("[%v][%v-%v]success", t, b.FileId, b.Blocks)
		return &RespBody{Code: 200, Msg: "success", Data: protob}, nil
	}

	// download dupl
	for i := 0; i < len(fmeta.FileDupl); i++ {
		err = readFile(string(fmeta.FileDupl[i].MinerIp), path, string(fmeta.FileDupl[i].DuplId), b.WalletAddress)
		if err != nil {
			Err.Sugar().Errorf("[%v][%v]%v", t, string(fmeta.FileDupl[i].DuplId), err)
			continue
		}
		break
	}

	// file not exist, query dupl file
	for i := 0; i < len(fmeta.FileDupl); i++ {
		duplname := filepath.Join(path, b.FileId+".d"+strconv.Itoa(i))
		_, err = os.Stat(duplname)
		if err == nil {
			buf, err := ioutil.ReadFile(duplname)
			if err != nil {
				Err.Sugar().Errorf("[%v][%v]%v", t, duplname, err)
				os.Remove(duplname)
				continue
			}
			//aes decryption
			ivkey := string(fmeta.FileDupl[i].RandKey)[:16]
			bkey := tools.Base58Decoding(string(fmeta.FileDupl[i].RandKey))
			decrypted, err := encryption.AesCtrDecrypt(buf, []byte(bkey), []byte(ivkey))
			if err != nil {
				Err.Sugar().Errorf("[%v][%v]%v", t, duplname, err)
				os.Remove(duplname)
				continue
			}
			fu, err := os.OpenFile(filefullname, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
			if err != nil {
				Err.Sugar().Errorf("[%v][%v][%v]%v", t, b.FileId, string(fmeta.FileDupl[i].DuplId), err)
				continue
			}
			fu.Write(decrypted)
			err = fu.Sync()
			if err != nil {
				Err.Sugar().Errorf("[%v][%v]%v", t, duplname, err)
				fu.Close()
				os.Remove(filefullname)
				continue
			}
			fu.Close()
			break
		}
	}

	_, err = os.Stat(filefullname)
	if err == nil {
		fuser, err := os.ReadFile(filefullname)
		if err != nil {
			Err.Sugar().Errorf("[%v][%v-%v]%v", t, b.FileId, b.Blocks, err)
			return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
		}
		slicesize, lastslicesize, num, err := fileshards.CutDataRule(uint64(len(fuser)))
		if err != nil {
			Err.Sugar().Errorf("[%v][%v-%v]%v", t, b.FileId, b.Blocks, err)
			return &RespBody{Code: 400, Msg: err.Error(), Data: nil}, nil
		}
		var tmp = make([]byte, 0)
		var blockSize int32
		if b.Blocks == int32(num) {
			tmp = fuser[uint64(len(fuser)-int(lastslicesize)):]
			blockSize = int32(lastslicesize)
		} else {
			tmp = fuser[uint64(uint64(b.Blocks-1)*slicesize):uint64(uint64(b.Blocks)*slicesize)]
			blockSize = int32(slicesize)
		}
		respb := &FileDownloadInfo{
			FileId:    b.FileId,
			Blocks:    b.Blocks,
			BlockSize: blockSize,
			BlockNum:  int32(num),
			Data:      tmp,
		}
		protob, err := proto.Marshal(respb)
		if err != nil {
			Err.Sugar().Errorf("[%v][%v-%v]%v", t, b.FileId, b.Blocks, err)
			return &RespBody{Code: 400, Msg: err.Error(), Data: nil}, nil
		}
		Out.Sugar().Infof("[%v][%v-%v]success", t, b.FileId, b.Blocks)
		return &RespBody{Code: 200, Msg: "success", Data: protob}, nil
	}

	return &RespBody{Code: 500, Msg: "fail", Data: nil}, nil
}

type RespSpacetagInfo struct {
	FileId string         `json:"fileId"`
	T      proof.FileTagT `json:"file_tag_t"`
	Sigmas [][]byte       `json:"sigmas"`
}
type RespSpacefileInfo struct {
	FileId     string `json:"fileId"`
	BlockTotal uint32 `json:"blockTotal"`
	BlockIndex uint32 `json:"blockIndex"`
	BlockData  []byte `json:"blockData"`
}

//
func (WService) SpaceAction(body []byte) (proto.Message, error) {
	var (
		err error
		t   int64
		b   SpaceTagReq
	)
	t = time.Now().Unix()
	Out.Sugar().Infof("[%v]Receive space request", t)
	err = proto.Unmarshal(body, &b)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive space request err:%v", t, err)
		return &RespBody{Code: 400, Msg: err.Error(), Data: nil}, nil
	}

	mdata, code, err := chain.GetMinerDataOnChain(b.WalletAddress)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive space request err:%v", t, err)
		return &RespBody{Code: code, Msg: err.Error(), Data: nil}, nil
	}
	pubkey, err := encryption.ParsePublicKey(mdata.Publickey)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive space request err:%v", t, err)
		return &RespBody{Code: 400, Msg: err.Error(), Data: nil}, nil
	}
	ok := encryption.VerifySign([]byte(b.WalletAddress), b.Sign, pubkey)
	if !ok {
		Out.Sugar().Infof("[%v]Receive space request err: Invalid signature", t)
		return &RespBody{Code: 403, Msg: "Invalid signature", Data: nil}, nil
	}

	filebasedir := filepath.Join(configs.SpaceCacheDir, base58.Encode([]byte(b.WalletAddress)))
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

		blockTotal := fi.Size() / 2 / 1024 / 1024
		respfile.BlockTotal = uint32(blockTotal)

		if b.BlockIndex >= uint32(blockTotal) {
			f.Close()
			Out.Sugar().Infof("[%v]Receive space request err: Invalid block index", t)
			return &RespBody{Code: 400, Msg: "Invalid block index", Data: nil}, nil
		}
		offset, err := f.Seek(int64(b.BlockIndex*2*1024*1024), 0)
		if err != nil {
			f.Close()
			Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
			return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
		}
		var buf = make([]byte, 2*1024*1024)
		_, err = f.ReadAt(buf, offset)
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
		if b.BlockIndex+1 == uint32(blockTotal) {
			os.Remove(filefullpath)
		}
		return &RespBody{Code: 201, Msg: "Invalid block index", Data: respfile_b}, nil
	}

	if b.SizeMb > 8 || b.SizeMb == 0 {
		Out.Sugar().Infof("[%v]Receive space request err: SizeMb up to 8 and not 0", t)
		return &RespBody{Code: 400, Msg: "SizeMb up to 8 and not 0", Data: nil}, nil
	}

	lines := b.SizeMb * 1024 * 1024 / configs.LengthOfALine
	filename := fmt.Sprintf("C%d_%d", mdata.Peerid, time.Now().UnixNano())
	filefullpath = filepath.Join(filebasedir, filename)
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
	hash, err := tools.CalcFileHash(filefullpath)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	}
	// calculate file tag info
	var PoDR2commit proof.PoDR2Commit
	var commitResponse proof.PoDR2CommitResponse
	PoDR2commit.FilePath = filename
	PoDR2commit.BlockSize = configs.BlockSize
	commitResponseCh, err := PoDR2commit.PoDR2ProofCommit(proof.Key_Ssk, string(proof.Key_SharedParams), int64(configs.ScanBlockSize))
	if err != nil {
		Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	}
	select {
	case commitResponse = <-commitResponseCh:
	}
	if commitResponse.StatueMsg.StatusCode != proof.Success {
		Out.Sugar().Infof("[%v]Receive space request err: PoDR2ProofCommit", t)
		return &RespBody{Code: 500, Msg: "unexpected system error", Data: nil}, nil
	}

	// up-chain meta info
	var metainfo chain.SpaceFileInfo
	metainfo.FileId = []byte(filename)
	metainfo.FileHash = []byte(hash)
	metainfo.FileSize = types.U64(uint64(b.SizeMb * 1024 * 1024))
	wal, err := tools.DecodeToPub(b.WalletAddress)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	}
	metainfo.Acc = types.NewAccountID(wal)
	metainfo.MinerId = mdata.Peerid

	_, _, n, err := tools.Split(f, PoDR2commit.BlockSize)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	}
	metainfo.BlockNum = types.U32(n)
	metainfo.ScanSize = types.U32(uint32(configs.ScanBlockSize))
	var file_blocks = make([]chain.BlockInfo, n)
	for i := uint64(1); i <= n; i++ {
		file_blocks[i].BlockIndex = types.U32(i)
		file_blocks[i].BlockSize = types.U32(PoDR2commit.BlockSize)
	}
	metainfo.BlockInfo = file_blocks
	_, err = chain.PutSpaceTagInfoToChain(
		configs.Confile.SchedulerInfo.ControllerAccountPhrase,
		metainfo,
	)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	}

	var resp RespSpacetagInfo
	resp.FileId = filename
	resp.T = commitResponse.T
	resp.Sigmas = commitResponse.Sigmas
	resp_b, err := json.Marshal(resp)
	if err != nil {
		Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
		return &RespBody{Code: 500, Msg: err.Error(), Data: nil}, nil
	}
	return &RespBody{Code: 202, Msg: "success", Data: resp_b}, nil
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
func writeData(dst string, service, method string, body []byte) error {
	dstip := "ws://" + tools.Base58Decoding(dst)
	dstip = strings.Replace(dstip, " ", "", -1)
	req := &ReqMsg{
		Service: service,
		Method:  method,
		Body:    body,
	}
	client, err := DialWebsocket(context.Background(), dstip, "")
	if err != nil {
		return errors.Wrap(err, "DialWebsocket:")
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	resp, err := client.Call(ctx, req)
	if err != nil {
		return errors.Wrap(err, "Call err:")
	}

	var b RespBody
	err = proto.Unmarshal(resp.Body, &b)
	if err != nil {
		return errors.Wrap(err, "Unmarshal:")
	}
	if b.Code == 200 {
		return nil
	}
	errstr := fmt.Sprintf("%d", b.Code)
	return errors.New("return code:" + errstr)
}

//
func readFile(dst string, path, fid, walletaddr string) error {
	dstip := "ws://" + tools.Base58Decoding(dst)
	dstip = strings.Replace(dstip, " ", "", -1)
	reqbody := FileDownloadReq{
		FileId:        fid,
		WalletAddress: walletaddr,
		Blocks:        0,
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
		if b_data.BlockNum <= 1 {
			f, err := os.OpenFile(filepath.Join(path, fid), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
			if err != nil {
				return err
			}
			f.Write(b_data.Data)
			f.Close()
			return nil
		} else {
			if b_data.Blocks == 0 {
				f, err := os.OpenFile(filepath.Join(path, fid+"-0"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
				if err != nil {
					return err
				}
				f.Write(b_data.Data)
				f.Close()
			}
		}
		for i := int32(1); i < b_data.BlockNum; i++ {
			reqbody := FileDownloadReq{
				FileId:        fid,
				WalletAddress: walletaddr,
				Blocks:        i,
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
			if i+1 == b_data.BlockNum {
				completefile := filepath.Join(path, fid)
				cf, err := os.OpenFile(completefile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC|os.O_APPEND, os.ModePerm)
				if err != nil {
					return err
				}
				defer cf.Close()
				for j := 0; j < int(b_data.BlockNum); j++ {
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

// processingfile is used to process all copies of the file and the corresponding tag information
func processingfile(t int64, fid, dir string, duplnamelist, duplkeynamelist []string) {
	var (
		err  error
		code int
		// file meta information
		filedump = make([]chain.FileDuplicateInfo, len(duplnamelist))
		mips     = make([]string, len(duplnamelist))
	)
	// query all miner
	var mDatas []chain.CessChain_AllMinerInfo
	trycount := 0
	for {
		mDatas, code, err = chain.GetAllMinerDataOnChain()
		if err != nil && code != configs.Code_403 {
			trycount++
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
		} else {
			break
		}
		if trycount > 3 {
			Err.Sugar().Errorf("[%v][%v][%v]Failed to query miner info,Please check your network.", t, dir, fid)
			return
		}
	}

	for i := 0; i < len(duplnamelist); i++ {
		duplname := filepath.Base(duplnamelist[i])
		fi, err := os.Stat(duplnamelist[i])
		if err != nil {
			Err.Sugar().Errorf("[%v][%v][%v]", t, duplnamelist[i], err)
			continue
		}
		f, err := os.OpenFile(duplnamelist[i], os.O_RDONLY, os.ModePerm)
		if err != nil {
			Err.Sugar().Errorf("[%v][%v][%v]", t, duplnamelist[i], err)
			continue
		}
		blockTotal := fi.Size() / configs.RpcFileBuffer
		if fi.Size()%configs.RpcFileBuffer > 0 {
			blockTotal += 1
		}
		var blockinfo = make([]chain.BlockInfo, blockTotal)
		var failminer = make(map[uint64]bool, 0)
		var index int = 0
		var mip = ""
		for j := int64(0); j < blockTotal; j++ {
			_, err := f.Seek(int64(j*2*1024*1024), 0)
			if err != nil {
				Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
				continue
			}
			var buf = make([]byte, configs.RpcFileBuffer)
			n, err := f.Read(buf)
			if err != nil {
				Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
				continue
			}

			var bo = PutFileToBucket{
				FileId:     duplname,
				FileHash:   "",
				BlockTotal: uint32(blockTotal),
				BlockSize:  uint32(n),
				BlockIndex: uint32(j),
				BlockData:  buf[:n],
			}
			bob, err := proto.Marshal(&bo)
			if err != nil {
				Err.Sugar().Errorf("[%v][%v][%v]", t, duplnamelist[i], err)
				continue
			}
			for {
				if mip == "" {
					index = tools.RandomInRange(0, len(mDatas))
					_, ok := failminer[uint64(mDatas[index].Peerid)]
					if ok {
						continue
					}
					err = writeData(string(mDatas[index].Ip), configs.RpcService_Miner, configs.RpcMethod_Miner_WriteFile, bob)
					if err == nil {
						mip = string(mDatas[index].Ip)
						blockinfo[j].BlockIndex = types.U32(uint32(j))
						blockinfo[j].BlockSize = types.U32(uint32(n))
						break
					} else {
						failminer[uint64(mDatas[index].Peerid)] = true
						Err.Sugar().Errorf("[%v][%v][%v]", t, duplnamelist[i], err)
						time.Sleep(time.Second * time.Duration(tools.RandomInRange(2, 5)))
					}
				} else {
					err = writeData(mip, configs.RpcService_Miner, configs.RpcMethod_Miner_WriteFile, bob)
					if err != nil {
						failminer[uint64(mDatas[index].Peerid)] = true
						Err.Sugar().Errorf("[%v][%v][%v]", t, duplnamelist[i], err)
						time.Sleep(time.Second * time.Duration(tools.RandomInRange(2, 5)))
						continue
					}
					blockinfo[j].BlockIndex = types.U32(uint32(j))
					blockinfo[j].BlockSize = types.U32(uint32(n))
					break
				}
			}
		}
		f.Close()
		filedump[i].DuplId = types.Bytes([]byte(duplname))
		filedump[i].RandKey = types.Bytes([]byte(filepath.Base(duplkeynamelist[i])))
		filedump[i].MinerId = mDatas[index].Peerid
		filedump[i].MinerIp = mDatas[index].Ip
		filedump[i].ScanSize = types.U32(configs.ScanBlockSize)
		mips[i] = string(mDatas[index].Ip)
		// Query miner information by id
		var mdetails chain.Chain_MinerDetails
		for {
			mdetails, _, err = chain.GetMinerDetailsById(uint64(mDatas[index].Peerid))
			if err != nil {
				Err.Sugar().Errorf("[%v][%v][%v]", t, duplnamelist[i], err)
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
				continue
			}
			break
		}
		filedump[i].Acc = mdetails.Address
		filedump[i].BlockNum = types.U32(uint32(blockTotal))
		filedump[i].BlockInfo = blockinfo
	}

	// Upload the file meta information to the chain and write it to the cache
	for {
		ok, err := chain.PutMetaInfoToChain(configs.Confile.SchedulerInfo.ControllerAccountPhrase, fid, filedump)
		if !ok || err != nil {
			Err.Sugar().Errorf("[%v][%v][%v]", t, fid, err)
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
			continue
		}
		Out.Sugar().Infof("[%v][%v]File metainfo up chain success", t, fid)
		c, err := cache.GetCache()
		if err != nil {
			Err.Sugar().Errorf("[%v][%v][%v]", t, fid, err)
		} else {
			b, err := json.Marshal(filedump)
			if err != nil {
				Err.Sugar().Errorf("[%v][%v][%v]", t, fid, err)
			} else {
				err = c.Put([]byte(fid), b)
				if err != nil {
					Err.Sugar().Errorf("[%v][%v][%v]", t, fid, err)
				} else {
					Out.Sugar().Infof("[%v][%v]File metainfo write cache success", t, fid)
				}
			}
		}
		break
	}

	// calculate file tag info
	for i := 0; i < len(filedump); i++ {
		var PoDR2commit proof.PoDR2Commit
		var commitResponse proof.PoDR2CommitResponse
		PoDR2commit.FilePath = string(filedump[i].DuplId)
		PoDR2commit.BlockSize = configs.BlockSize
		commitResponseCh, err := PoDR2commit.PoDR2ProofCommit(proof.Key_Ssk, string(proof.Key_SharedParams), int64(configs.ScanBlockSize))
		if err != nil {
			Err.Sugar().Errorf("[%v][%v][%v]", t, filedump[i], err)
			continue
		}
		select {
		case commitResponse = <-commitResponseCh:
		}
		if commitResponse.StatueMsg.StatusCode != proof.Success {
			Err.Sugar().Errorf("[%v][%v][%v]", t, filedump[i], err)
			continue
		}
		var resp rpc.PutTagToBucket
		resp.FileId = string(filedump[i].DuplId)
		resp.Name = commitResponse.T.Name
		resp.N = commitResponse.T.N
		resp.U = commitResponse.T.U
		resp.Signature = commitResponse.T.Signature
		resp.Sigmas = commitResponse.Sigmas
		resp_proto, err := proto.Marshal(&resp)
		if err != nil {
			Out.Sugar().Infof("[%v]Receive space request err: %v", t, err)
			continue
		}

		err = writeData(mips[i], configs.RpcService_Miner, configs.RpcMethod_Miner_WriteFileTag, resp_proto)
		if err != nil {
			Err.Sugar().Errorf("[%v][%v][%v]%v", t, mips[i], duplnamelist[i], err)
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(2, 5)))
			continue
		}
	}
}
