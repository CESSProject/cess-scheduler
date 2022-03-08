package rpc

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"scheduler-mining/configs"
	"scheduler-mining/internal/chain"
	"scheduler-mining/internal/encryption"
	"scheduler-mining/internal/fileshards"
	"scheduler-mining/tools"
	"strconv"

	myproto "scheduler-mining/rpc/proto"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/golang/protobuf/proto"
)

type WService struct {
}

func Rpc_Init() {
	if err := tools.CreatDirIfNotExist(configs.CacheFilePath); err != nil {
		panic(err)
	}
}

func Rpc_Main() {
	srv := NewServer()
	srv.Register("wservice", WService{})

	err := http.ListenAndServe(":"+configs.Confile.SchedulerInfo.ServicePort, srv.WebsocketHandler([]string{"*"}))
	if err != nil {
		panic(err)
	}
}

// Test
func (WService) TestAction(body []byte) (proto.Message, error) {
	return &Err{Msg: "test hello"}, nil
}

// Write file from client
func (WService) WritefileAction(body []byte) (proto.Message, error) {
	var (
		b         myproto.FileUploadInfo
		cachepath string
		fmeta     chain.FileMetaInfo
	)
	err := proto.Unmarshal(body, &b)
	if err != nil {
		return &Err{Code: 400, Msg: "body format error"}, nil
	}

	err = tools.CreatDirIfNotExist(configs.CacheFilePath)
	if err == nil {
		cachepath = filepath.Join(configs.CacheFilePath, b.FileId)
	} else {
		cachepath = filepath.Join("./cesscache", b.FileId)
	}

	_, err = os.Stat(cachepath)
	if err != nil {
		fmeta, err = chain.GetFileMetaInfoOnChain(configs.ChainModule_FileMap, configs.ChainModule_FileMap_FileMetaInfo, b.FileId)
		if err != nil {
			return &Err{Code: 500, Msg: "Net error"}, nil
		}
		if string(fmeta.FileId) == b.FileId {
			err = os.MkdirAll(cachepath, os.ModeDir)
			if err != nil {
				return &Err{Code: 500, Msg: "mkdir error"}, nil
			}
		} else {
			return &Err{Code: 400, Msg: "fileid error"}, nil
		}
	}

	filename := cachepath + b.FileId + "_" + fmt.Sprintf("%d", b.BlockNum)
	f, err := os.Create(filename)
	if err != nil {
		return &Err{Code: 500, Msg: "mkdir error"}, nil
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	nn, err := w.Write(b.Data)
	if err != nil {
		return &Err{Code: 500, Msg: "write file error"}, nil
	}
	if nn != int(b.BlockSize) {
		return &Err{Code: 400, Msg: "block size error"}, nil
	}
	err = w.Flush()
	if err != nil {
		return &Err{Code: 500, Msg: "write flush error"}, nil
	}
	if b.BlockNum+1 == b.Blocks {
		go recvCallBack(b.FileId, cachepath, int(b.Blocks), fmeta)
	}
	return &Err{Code: 200, Msg: "sucess"}, nil
}

// Read file from client
func (WService) ReadfileAction(body []byte) (proto.Message, error) {

	return &RespMsg{Id: 200, Body: nil}, nil
}

func recvCallBack(fid, dir string, num int, meta chain.FileMetaInfo) {
	completefile, err := combinationFile(fid, dir, num)
	if err != nil {
		//TODO
	} else {
		for i := 0; i < num; i++ {
			os.Remove(dir + "/" + fid + "_" + strconv.Itoa(int(i)))
		}
	}
	h, err := tools.CalcFileHash(completefile)
	if err != nil {
		//TODO
	}
	if h != string(meta.FileHash) {
		//TODO
	}

	f, err := os.Open(completefile)
	if err != nil {
		//TODO
	}
	defer f.Close()
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, f); err != nil {
		//TODO
	}
	// Multiple copies
	for i := 0; i < int(meta.Backups); i++ {
		var filedump = make([]chain.FileDuplicateInfo, meta.Backups)
		//	aes encrypt
		key := tools.GetRandomkey(32)
		key_base58 := base64.StdEncoding.EncodeToString(tools.S2B(key))

		encrypted, err := encryption.AesCtrEncrypt(buf.Bytes(), []byte(key), []byte(key_base58[:8]))
		if err != nil {
			//TODO
		}
		enfile := completefile + "-" + fmt.Sprintf("%d", i)
		f, err := os.OpenFile(enfile, os.O_CREATE|os.O_WRONLY, os.ModePerm)
		if err != nil {
			//TODO
		}
		f.Write(encrypted)
		f.Close()
		filedump[i].DuplId = types.NewBytes([]byte(string(meta.FileId) + "." + strconv.Itoa(i)))
		filedump[i].RandKey = types.NewBytes([]byte(key_base58))
		fileshards, err := fileshards.CutFile(enfile)
		if err != nil {
			//TODO
		}
		filedump[i].SliceNum = types.NewU32(uint32(len(fileshards)))
		for j := 0; j < len(fileshards); j++ {

		}
	}
}

func combinationFile(fid, dir string, num int) (string, error) {
	completefile := filepath.Join(dir, fid+".cess")
	cf, err := os.OpenFile(completefile, os.O_CREATE|os.O_WRONLY|os.O_APPEND|os.O_TRUNC, os.ModePerm)
	if err != nil {
		fmt.Println(err)
		return completefile, err
	}
	defer cf.Close()
	for i := 0; i < num; i++ {
		f, err := os.OpenFile(dir+"/"+fid+"_"+strconv.Itoa(int(i)), os.O_RDONLY, os.ModePerm)
		if err != nil {
			fmt.Println(err)
			return completefile, err
		}
		defer f.Close()
		b, err := ioutil.ReadAll(f)
		if err != nil {
			fmt.Println(err)
			return completefile, err
		}
		cf.Write(b)
	}
	return completefile, nil
}
