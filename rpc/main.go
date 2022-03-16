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
	fmt.Println("**** recv a writefile connect ****")
	err := proto.Unmarshal(body, &b)
	if err != nil {
		return &Err{Code: 400, Msg: "body format error"}, nil
	}
	fmt.Printf("req info: %v,  %v\n", b.FileId, b.FileHash)
	err = tools.CreatDirIfNotExist(configs.CacheFilePath)
	if err == nil {
		cachepath = filepath.Join(configs.CacheFilePath, b.FileId)
	} else {
		cachepath = filepath.Join("./cesscache", b.FileId)
	}

	_, err = os.Stat(cachepath)
	if err != nil {
		fmeta, err = chain.GetFileMetaInfoOnChain(configs.ChainModule_FileBank, configs.ChainModule_FileMap_FileMetaInfo, b.FileId)
		if err != nil {
			return &Err{Code: 500, Msg: "Net error"}, nil
		}
		fmt.Println("chainfile hash:", string(fmeta.FileHash), "backups: ", fmeta.Backups)
		if string(fmeta.FileHash) == b.FileHash {
			err = os.MkdirAll(cachepath, os.ModeDir)
			if err != nil {
				return &Err{Code: 500, Msg: "mkdir error"}, nil
			}
		} else {
			return &Err{Code: 400, Msg: "FileHash error"}, nil
		}
	}

	filename := filepath.Join(cachepath, b.FileId+"_"+fmt.Sprintf("%d", b.BlockNum))
	f, err := os.Create(filename)
	if err != nil {
		return &Err{Code: 500, Msg: "mkdir error"}, nil
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	_, err = w.Write(b.Data)
	if err != nil {
		return &Err{Code: 500, Msg: "write file error"}, nil
	}
	//fmt.Println(b.BlockSize)
	// if nn != int(b.BlockSize) {
	// 	return &Err{Code: 400, Msg: "block size error"}, nil
	// }
	err = w.Flush()
	if err != nil {
		return &Err{Code: 500, Msg: "write flush error"}, nil
	}
	if b.BlockNum == b.Blocks {
		go recvCallBack(b.FileId, cachepath, int(b.Blocks), uint8(fmeta.Backups))
	}
	return &Err{Code: 0, Msg: "sucess"}, nil
}

// Read file from client
func (WService) ReadfileAction(body []byte) (proto.Message, error) {
	var (
		b myproto.FileDownloadReq
	)

	fmt.Println("**** recv a readfile connect ****")
	err := proto.Unmarshal(body, &b)
	if err != nil {
		fmt.Println("proto.Unmarshal err: ", err)
		return &Err{Code: 400, Msg: "body format error"}, nil
	}
	fmt.Println("req info: ", b.FileId, "  ", b.Blocks, " ", b.WalletAddress)
	//Query client is able to read file
	fmeta, err := chain.GetFileMetaInfoOnChain(configs.ChainModule_FileBank, configs.ChainModule_FileMap_FileMetaInfo, b.FileId)
	if err != nil {
		fmt.Println("GetFileMetaInfoOnChain err: ", err)
		return &Err{Code: 500, Msg: "Network timeout, try again later!"}, nil
	}

	a, err := types.NewAddressFromHexAccountID(b.WalletAddress)
	if err != nil {
		fmt.Println("NewAddressFromHexAccountID err: ", err)
		//return &Err{Code: 500, Msg: "Network timeout, try again later!"}, nil
	}
	fmt.Println(a.AsAccountID)
	// if fmeta.UserAddr != types.NewAccountID(types.De) {
	// 	fmt.Println("NewAccountID err: ", err)
	// 	return &Err{Code: 400, Msg: "No permission"}, nil
	// }

	path := filepath.Join(configs.CacheFilePath, b.FileId)
	fmt.Println("path: ", path)
	_, err = os.Stat(filepath.Join(path, b.FileId+".user"))
	if err != nil {
		for i := 0; i < len(fmeta.FileDupl); i++ {
			for j := 0; j < int(fmeta.FileDupl[i].SliceNum); j++ {
				for k := 0; k < len(fmeta.FileDupl[i].FileSlice[j].FileShard.ShardHash); k++ {
					b := myproto.FileDownloadReq{
						FileId:        string(fmeta.FileDupl[i].FileSlice[j].FileShard.ShardHash[k]),
						WalletAddress: b.WalletAddress,
					}
					fmt.Println("will read: ", b.FileId)
					bo, err := proto.Marshal(&b)
					if err != nil {
						//TODO
						fmt.Println("proto.Marshal err: ", err)
					}
					fs, err := readFile(string(fmeta.FileDupl[i].FileSlice[j].FileShard.ShardAddr[k]), bo)
					if err != nil {
						//TODO
						fmt.Println("readFile err: ", err)
					}

					_, err = os.Stat(path)
					if err != nil {
						os.MkdirAll(path, os.ModeDir)
					}
					f, err := os.OpenFile(filepath.Join(path, string(fmeta.FileDupl[i].FileSlice[j].FileShard.ShardHash[k])), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
					if err != nil {
						//TODO
						fmt.Println("os.OpenFile err-167: ", err)
					}
					f.Write(fs)
					f.Close()
				}
				//reed solomon recover
				err = fileshards.ReedSolomon_Restore(filepath.Join(path, string(fmeta.FileDupl[i].FileSlice[j].SliceId)), int(fmeta.FileDupl[i].FileSlice[j].FileShard.DataShardNum), int(fmeta.FileDupl[i].FileSlice[j].FileShard.RedunShardNum))
				if err != nil {
					//TODO
					fmt.Println("ReedSolomon_Restore err: ", err)
				}
				if j+1 == int(fmeta.FileDupl[i].SliceNum) {
					fii, err := os.OpenFile(filepath.Join(path, b.FileId+".cess"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
					if err != nil {
						//TODO
						fmt.Println("os.OpenFile err-182: ", err)
					}
					defer fii.Close()
					for l := 0; l <= int(fmeta.FileDupl[i].SliceNum); l++ {
						f, err := os.OpenFile(filepath.Join(path, string(fmeta.FileDupl[i].FileSlice[l].SliceId)), os.O_RDONLY, os.ModePerm)
						if err != nil {
							//TODO
							fmt.Println("os.OpenFile err-189: ", err)
						}
						defer f.Close()
						b, err := ioutil.ReadAll(f)
						if err != nil {
							//TODO
							fmt.Println("os.OpenFile err-195: ", err)
						}
						fii.Write(b)
					}
					//aes decryption
					ivkey := string(fmeta.FileDupl[i].RandKey)[:8]
					bkey, err := base64.StdEncoding.DecodeString(string(fmeta.FileDupl[i].RandKey))
					if err != nil {
						//TODO
						fmt.Println("base64.StdEncoding.DecodeString err: ", err)
					}
					buf := bytes.NewBuffer(nil)
					if _, err := io.Copy(buf, fii); err != nil {
						//TODO
						fmt.Println("io.Copy err: ", err)
					}
					decrypted, err := encryption.AesCtrDecrypt(buf.Bytes(), bkey, []byte(ivkey))
					if err != nil {
						//TODO
						fmt.Println("encryption.AesCtrDecrypt err: ", err)
					}
					fuser, err := os.OpenFile(filepath.Join(path, b.FileId+".user"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
					if err != nil {
						//TODO
						fmt.Println("os.OpenFile err-219: ", err)
					}
					defer fuser.Close()
					fuser.Write(decrypted)

					slicesize, lastslicesize, num, err := fileshards.CutDataRule(uint64(len(decrypted)))
					if err != nil {
						//TODO
						fmt.Println("fileshards.CutDataRule err: ", err)
					}
					if b.Blocks >= int32(num) {
						fmt.Println(" b.Blocks >= int32(num) ")
						return &Err{Code: 400, Msg: "BlockNum err"}, nil
					}
					var tmp = make([]byte, 0)
					if b.Blocks+1 == int32(num) {
						tmp = decrypted[uint64(len(decrypted)-int(lastslicesize)):]
					} else {
						tmp = decrypted[uint64(uint64(b.Blocks)*slicesize):uint64(uint64(b.Blocks+1)*slicesize)]
					}
					respb := &myproto.FileDownloadInfo{
						FileId:    b.FileId,
						Blocks:    int32(num),
						BlockSize: int32(slicesize),
						BlockNum:  b.Blocks,
						Data:      tmp,
					}
					protob, err := proto.Marshal(respb)
					if err != nil {
						//TODO
						fmt.Println("proto.Marshal err-2: ", err)
					}
					return &RespMsg{Body: protob}, nil
				}
			}
		}
	} else {
		fuser, err := os.Open(filepath.Join(path, b.FileId+".user"))
		if err != nil {
			//TODO
			fmt.Println("os.Open err-259: ", err)
		}
		buf := bytes.NewBuffer(nil)
		if _, err := io.Copy(buf, fuser); err != nil {
			//TODO
			fmt.Println("io.Copy err-264: ", err)
		}
		slicesize, lastslicesize, num, err := fileshards.CutDataRule(uint64(buf.Len()))
		if err != nil {
			//TODO
			fmt.Println("CutDataRule err: ", err)
		}
		var tmp = make([]byte, 0)
		if b.Blocks+1 == int32(num) {
			tmp = buf.Bytes()[uint64(buf.Len()-int(lastslicesize)):]
		} else {
			tmp = buf.Bytes()[uint64(uint64(b.Blocks)*slicesize):uint64(uint64(b.Blocks+1)*slicesize)]
		}
		respb := &myproto.FileDownloadInfo{
			FileId:    b.FileId,
			Blocks:    int32(num),
			BlockSize: int32(slicesize),
			BlockNum:  b.Blocks,
			Data:      tmp,
		}
		protob, err := proto.Marshal(respb)
		if err != nil {
			//TODO
			fmt.Println("proto.Marshal err-287: ", err)
		}
		return &RespMsg{Body: protob}, nil
	}
	//fileshards.CutDataRule(uint64(fmeta.FileSize))
	return &Err{Code: 500, Msg: "fail"}, nil
}

func recvCallBack(fid, dir string, num int, bks uint8) {
	completefile, err := combinationFile(fid, dir, num)
	if err != nil {
		fmt.Println(err)
		return
	} else {
		for i := 1; i <= num; i++ {
			path := filepath.Join(dir, fid+"_"+strconv.Itoa(int(i)))
			os.Remove(path)
		}
	}
	// h, err := tools.CalcFileHash(completefile)
	// if err != nil {
	// 	//TODO
	// 	fmt.Println("CalcFileHash err: ", completefile, " ", err)
	// }
	//if h != string(meta.FileHash) {
	//TODO
	//fmt.Println("hash :", h, " ", err)
	//}

	fcess, err := os.Open(completefile)
	if err != nil {
		fmt.Println(err)
		return
	}
	buf := bytes.NewBuffer(nil)
	_, err = io.Copy(buf, fcess)
	if err != nil {
		fmt.Println(err)
		return
	}
	fcess.Close()
	bkups := 0
	if bks < 3 {
		bkups = 3
	} else {
		bkups = int(bks)
	}
	var filedump = make([]chain.FileDuplicateInfo, bkups)

	// Multiple copies
	for i := 0; i < int(bkups); i++ {
		//	aes encrypt
		key := tools.GetRandomkey(32)
		key_base58 := tools.Base58Encoding(key)
		fmt.Println("key: ", key, " ", "key_base58: ", key_base58)

		encrypted, err := encryption.AesCtrEncrypt(buf.Bytes(), []byte(key), []byte(key_base58[:16]))
		if err != nil {
			fmt.Println("AesCtrEncrypt err: ", err)
			return
		}
		enfile := completefile + "-" + fmt.Sprintf("%d", i)
		f, err := os.OpenFile(enfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND|os.O_TRUNC, os.ModePerm)
		if err != nil {
			fmt.Println("OpenFile err-340: ", err)
			return
		}
		f.Write(encrypted)
		f.Close()
		filedump[i].DuplId = types.NewBytes([]byte(fid + "-" + strconv.Itoa(i)))
		filedump[i].RandKey = types.NewBytes([]byte(key_base58))
		fileshard, slicesize, lastslicesize, err := fileshards.CutFile(enfile)
		if err != nil {
			fmt.Println("CutFile err: ", err)
			return
		}
		filedump[i].SliceNum = types.U16(uint16(len(fileshard)))
		filedump[i].FileSlice = make([]chain.FileSliceInfo, len(fileshard))

		//Query Miner and transport
		mDatas, err := chain.GetAllMinerDataOnChain(
			configs.ChainModule_Sminer,
			configs.ChainModule_Sminer_AllMinerItems,
		)
		if err != nil {
			fmt.Println("GetAllMinerDataOnChain err: ", err)
			return
		}
		for j := 0; j < len(fileshard); j++ {
			filedump[i].FileSlice[j].SliceId = []byte(filepath.Base(fileshard[j]))
			h, err := tools.CalcFileHash(fileshard[j])
			if err != nil {
				fmt.Println(err)
				return
			} else {
				filedump[i].FileSlice[j].SliceHash = types.Bytes([]byte(h))
			}

			if j+1 == len(fileshard) {
				filedump[i].FileSlice[j].SliceSize = types.U32(lastslicesize)
			} else {
				filedump[i].FileSlice[j].SliceSize = types.U32(slicesize)
			}
			shards, datashards, rdunshards, err := fileshards.ReedSolomon(fileshard[j])
			if err != nil {
				fmt.Println("ReedSolomon err: ", err)
				return
			} else {
				fmt.Println("ReedSolomon ", fileshard[j], ": ", shards, datashards, rdunshards)
			}
			filedump[i].FileSlice[j].FileShard.DataShardNum = types.NewU8(uint8(datashards))
			filedump[i].FileSlice[j].FileShard.RedunShardNum = types.NewU8(uint8(rdunshards))
			var shardshash []types.Bytes = make([]types.Bytes, len(shards))
			var shardaddr []types.Bytes = make([]types.Bytes, len(shards))
			var mineraccount = make([]types.U64, 0)
			for k := 0; k < len(shards); k++ {
				shardshash[k] = make(types.Bytes, 0)
				shardaddr[k] = make(types.Bytes, 0)
				shardshash[k] = append(shardshash[k], types.Bytes([]byte(filepath.Base(shards[k])))...)
				fmt.Println("shards[", k, "]: ", shards[k])
				fn, err := os.Open(shards[k])
				if err != nil {
					fmt.Println("Open err: ", err)
					return
				}
				buf := bytes.NewBuffer(nil)
				if _, err = io.Copy(buf, fn); err != nil {
					fmt.Println("Copy err: ", err)
					return
				}
				fn.Close()
				var bo = myproto.FileUploadInfo{
					FileId:    shards[k],
					FileHash:  "",
					Backups:   "",
					Blocks:    0,
					BlockSize: 0,
					BlockNum:  0,
					Data:      buf.Bytes(),
				}
				bob, err := proto.Marshal(&bo)
				if err != nil {
					fmt.Println("proto.Marshal err: ", err)
					return
				}
				for {
					index := tools.RandomInRange(0, len(mDatas))
					err = writeFile(string(mDatas[index].Ip), bob)
					if err == nil {
						fmt.Println("writeFile ok")
						shardaddr[k] = append(shardaddr[k], mDatas[index].Ip...)
						mineraccount = append(mineraccount, mDatas[index].Peerid)
						break
					} else {
						fmt.Println("writeFile failed: ", err)
					}
					time.Sleep(time.Second * 3)
				}
			}
			if len(shardshash) == len(shardaddr) {
				filedump[i].FileSlice[j].FileShard.ShardHash = shardshash
				filedump[i].FileSlice[j].FileShard.ShardAddr = shardaddr
				filedump[i].FileSlice[j].FileShard.Peerid = mineraccount
			} else {
				//TODO
				fmt.Println("------------------------err----------------------------")
			}
		}
	}

	// file meta info up chain
	for {
		ok, err := chain.PutMetaInfoToChain(configs.Confile.SchedulerInfo.TransactionPrK, configs.ChainTx_FileBank_PutMetaInfo, fid, filedump)
		if err != nil {
			fmt.Println(err)
		}
		if !ok {
			fmt.Println("------------------------File meta up chain false----------------------------")
		} else {
			fmt.Println("------------------------File meta up chain success----------------------------")
			return
		}
		time.Sleep(time.Second * 5)
	}
}

//
func combinationFile(fid, dir string, num int) (string, error) {
	completefile := filepath.Join(dir, fid+".cess")
	cf, err := os.OpenFile(completefile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC|os.O_APPEND, os.ModePerm)
	if err != nil {
		fmt.Println(err)
		return completefile, err
	}
	defer cf.Close()
	for i := 1; i <= num; i++ {
		path := filepath.Join(dir, fid+"_"+strconv.Itoa(int(i)))
		f, err := os.Open(path)
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
	wsURL := "ws://" + strings.TrimPrefix(dstip, "http://")
	fmt.Println("wsURL: ", wsURL)
	req := &ReqMsg{
		Service: configs.RpcService_Miner,
		Method:  configs.RpcMethod_Miner_WriteFile,
		Body:    body,
	}
	ctx1, cancel1 := context.WithTimeout(context.Background(), 10*time.Second)
	client, err := DialWebsocket(ctx1, wsURL, "")
	defer cancel1()
	if err != nil {
		return err
	}
	defer client.Close()
	ctx2, cancel2 := context.WithTimeout(context.Background(), 90*time.Second)
	resp, err := client.Call(ctx2, req)
	defer cancel2()
	if err != nil {
		return err
	}

	var b Err
	err = proto.Unmarshal(resp.Body, &b)
	if err != nil {
		fmt.Println(err)
	}
	if b.Code == 0 {
		fmt.Println("code: ", b.Code)
		return nil
	}
	errstr := fmt.Sprintf("%d", b.Code)
	fmt.Println("errstr: ", errstr)
	return errors.New("return code:" + errstr)
}

//
func readFile(dst string, body []byte) ([]byte, error) {
	dstip := tools.Base58Decoding(dst)
	wsURL := "ws:" + strings.TrimPrefix(dstip, "http:")
	fmt.Println("will read dst: ", wsURL)
	req := &ReqMsg{
		Service: configs.RpcService_Miner,
		Method:  configs.RpcMethod_Miner_ReadFile,
		Body:    body,
	}
	client, err := DialWebsocket(context.Background(), wsURL, "")
	if err != nil {
		return nil, err
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	resp, err := client.Call(ctx, req)
	defer cancel()
	if err != nil {
		return nil, err
	}

	var b Err
	err = proto.Unmarshal(resp.Body, &b)
	if err != nil {
		return resp.Body, nil
	}
	errstr := fmt.Sprintf("%d", b.Code)
	return nil, errors.New(errstr)
}
