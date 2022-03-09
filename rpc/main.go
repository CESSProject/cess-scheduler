package rpc

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
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
	"strings"
	"time"

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
	return &Err{Code: 0, Msg: "sucess"}, nil
}

// Read file from client
func (WService) ReadfileAction(body []byte) (proto.Message, error) {
	var (
		b myproto.FileDownloadReq
	)
	err := proto.Unmarshal(body, &b)
	if err != nil {
		return &Err{Code: 400, Msg: "body format error"}, nil
	}
	return &RespMsg{Id: 0, Body: nil}, nil
}

func recvCallBack(fid, dir string, num int, meta chain.FileMetaInfo) {
	completefile, err := combinationFile(fid, dir, num)
	if err != nil {
		//TODO
		fmt.Println(err)
	} else {
		for i := 0; i < num; i++ {
			os.Remove(dir + "/" + fid + "_" + strconv.Itoa(int(i)))
		}
	}
	h, err := tools.CalcFileHash(completefile)
	if err != nil {
		//TODO
		fmt.Println(err)
	}
	if h != string(meta.FileHash) {
		//TODO
		fmt.Println(err)
	}

	f, err := os.Open(completefile)
	if err != nil {
		//TODO
		fmt.Println(err)
	}
	defer f.Close()
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, f); err != nil {
		//TODO
		fmt.Println(err)
	}
	var filedump = make([]chain.FileDuplicateInfo, meta.Backups)
	// Multiple copies
	for i := 0; i < int(meta.Backups); i++ {
		//	aes encrypt
		key := tools.GetRandomkey(32)
		key_base58 := base64.StdEncoding.EncodeToString(tools.S2B(key))

		encrypted, err := encryption.AesCtrEncrypt(buf.Bytes(), []byte(key), []byte(key_base58[:8]))
		if err != nil {
			//TODO
			fmt.Println(err)
		}
		enfile := completefile + "-" + fmt.Sprintf("%d", i)
		f, err := os.OpenFile(enfile, os.O_CREATE|os.O_WRONLY, os.ModePerm)
		if err != nil {
			//TODO
			fmt.Println(err)
		}
		f.Write(encrypted)
		f.Close()
		filedump[i].DuplId = types.NewBytes([]byte(string(meta.FileId) + "." + strconv.Itoa(i)))
		filedump[i].RandKey = types.NewBytes([]byte(key_base58))
		fileshard, slicesize, lastslicesize, err := fileshards.CutFile(enfile)
		if err != nil {
			//TODO
			fmt.Println(err)
		}
		filedump[i].SliceNum = types.NewU32(uint32(len(fileshard)))
		filedump[i].FileSlice = make([]chain.FileSliceInfo, len(fileshard))

		//Query Miner and transport
		mDatas, err := chain.GetAllMinerDataOnChain(
			configs.ChainModule_Sminer,
			configs.ChainModule_Sminer_AllMinerItems,
		)
		if err != nil {
			//TODO
			fmt.Println(err)
		}

		for j := 0; j < len(fileshard); j++ {
			filedump[i].FileSlice[j].SliceId = []byte(enfile + "-" + fmt.Sprintf("%d", j))
			h, err := tools.CalcFileHash(fileshard[j])
			if err != nil {
				//TODO
				fmt.Println(err)
			} else {
				filedump[i].FileSlice[j].SliceHash = types.NewBytes([]byte(h))
			}

			if j+1 == len(fileshard) {
				filedump[i].FileSlice[j].SliceSize = types.U64(lastslicesize)
			} else {
				filedump[i].FileSlice[j].SliceSize = types.U64(slicesize)
			}
			shards, datashards, rdunshards, err := fileshards.ReedSolomon(fileshard[j])
			if err != nil {
				//TODO
				fmt.Println(err)
			}
			filedump[i].FileSlice[j].FileShard.DataShardNum = types.NewU16(uint16(datashards))
			filedump[i].FileSlice[j].FileShard.RedunShardNum = types.NewU16(uint16(rdunshards))
			var shardshash []types.Bytes = make([]types.Bytes, len(shards))
			var shardaddr []types.Bytes = make([]types.Bytes, len(shards))
			var mineraccount []types.Bytes = make([]types.Bytes, len(shards))
			for k := 0; k < len(shards); k++ {
				shardshash[k] = make(types.Bytes, 0)
				shardaddr[k] = make(types.Bytes, 0)
				mineraccount[k] = make(types.Bytes, 0)
				shardshash[k] = append(shardshash[k], types.NewBytes([]byte(shards[k]))...)
				fn, err := os.Open(shards[k])
				if err != nil {
					//TODO
					fmt.Println(err)
				}
				defer fn.Close()
				buf := bytes.NewBuffer(nil)
				if _, err = io.Copy(buf, fn); err != nil {
					//TODO
					fmt.Println(err)
				}
				var bo = myproto.FileUploadInfo{
					FileId:    fid,
					FileHash:  "",
					Backups:   "",
					Blocks:    0,
					BlockSize: 0,
					BlockNum:  0,
					Data:      buf.Bytes(),
				}
				bob, err := proto.Marshal(&bo)
				if err != nil {
					//TODO
					fmt.Println(err)
				}
				for {
					index := tools.RandomInRange(0, len(mDatas))
					err = writeFile(string(mDatas[index].Ip), bob)
					if err == nil {
						shardaddr[k] = append(shardaddr[k], mDatas[index].Ip...)
						mineraccount[k] = append(mineraccount[k], mDatas[index].AccountID[:]...)
						break
					}
				}
			}
			if len(shardshash) == len(shardaddr) {
				filedump[i].FileSlice[j].FileShard.ShardHash = shardshash
				filedump[i].FileSlice[j].FileShard.ShardAddr = shardaddr

			} else {
				//TODO
				fmt.Println("------------------------err----------------------------")
			}
		}
	}
	// file meta info up chain
	ok, err := chain.PutMetaInfoToChain(configs.Confile.SchedulerInfo.TransactionPrK, configs.ChainTx_FileBank_PutMetaInfo, filedump)
	if err != nil {
		fmt.Println(err)
	}
	if !ok {
		fmt.Println("------------------------File meta up chain false----------------------------")
	}
}

//
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

//
func writeFile(dst string, body []byte) error {
	dstip := tools.Base58Decoding(dst)
	wsURL := "ws:" + strings.TrimPrefix(dstip, "http:")
	req := &ReqMsg{
		Service: configs.RpcService_Miner,
		Method:  configs.RpcMethod_Miner_WriteFile,
		Body:    body,
	}
	client, err := DialWebsocket(context.Background(), wsURL, "")
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	resp, err := client.Call(ctx, req)
	if err != nil {
		return err
	}
	cancel()
	var b Err
	err = proto.Unmarshal(resp.Body, &b)
	if err != nil {
		fmt.Println(err)
	}
	if b.Code == 0 {
		return nil
	}
	errstr := fmt.Sprintf("%d", b.Code)
	return errors.New("return code:" + errstr)
}
