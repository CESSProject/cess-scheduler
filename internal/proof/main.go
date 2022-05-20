package proof

import (
	"cess-scheduler/configs"
	"cess-scheduler/internal/chain"
	"cess-scheduler/internal/encryption"
	. "cess-scheduler/internal/logger"
	api "cess-scheduler/internal/proof/apiv1"
	proof "cess-scheduler/internal/proof/apiv1"
	"cess-scheduler/internal/rpc"
	p "cess-scheduler/internal/rpc/protobuf"
	"cess-scheduler/tools"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"storj.io/common/base58"

	"google.golang.org/protobuf/proto"
)

type TagInfo struct {
	T      api.FileTagT `json:"file_tag_t"`
	Sigmas [][]byte     `json:"sigmas"`
}

// Enable the verification proof module
func Chain_Main() {
	var (
		channel_1 = make(chan bool, 1)
		channel_2 = make(chan bool, 1)
	)
	go task_ValidateProof(channel_1)
	go task_RecoveryFiles(channel_2)
	for {
		select {
		case <-channel_1:
			go task_ValidateProof(channel_1)
		case <-channel_2:
			go task_RecoveryFiles(channel_2)
		}
	}
}

func task_ValidateProof(ch chan bool) {
	var (
		err         error
		code        int
		puk         chain.Chain_SchedulerPuk
		poDR2verify api.PoDR2Verify
		proofs      []chain.Chain_Proofs
	)
	defer func() {
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
		ch <- true
	}()
	Out.Info(">>>Start task_ValidateProof task<<<")

	puk, _, err = chain.GetSchedulerPukFromChain()
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}

	for {
		time.Sleep(time.Second * time.Duration(tools.RandomInRange(10, 30)))
		proofs, code, err = chain.GetProofsFromChain(configs.C.CtrlPrk)
		if err != nil {
			if code != configs.Code_404 {
				Err.Sugar().Errorf("%v", err)
			}
			continue
		}

		for i := 0; i < len(proofs); i++ {
			var reqtag p.ReadTagReq
			reqtag.FileId = string(proofs[i].Challenge_info.File_id)
			reqtag.Acc, err = chain.GetAddressByPrk(configs.C.CtrlPrk)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", proofs[i].Miner_id, err)
				continue
			}
			req_proto, err := proto.Marshal(&reqtag)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", proofs[i].Miner_id, err)
				continue
			}
			minerDetails, _, err := chain.GetMinerDetailsById(uint64(proofs[i].Miner_id))
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", proofs[i].Miner_id, err)
				continue
			}
			respData, err := rpc.WriteData(string(minerDetails.Ip), configs.RpcService_Miner, configs.RpcMethod_Miner_ReadFileTag, req_proto)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", proofs[i].Miner_id, err)
				continue
			}
			var tag TagInfo
			err = json.Unmarshal(respData, &tag)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", proofs[i].Miner_id, err)
				continue
			}
			qSlice, err := api.PoDR2ChallengeGenerateFromChain(proofs[i].Challenge_info.Block_list, proofs[i].Challenge_info.Random)
			if err != nil {
				Err.Sugar().Errorf("[%v] %v", proofs[i].Miner_id, err)
				continue
			}

			poDR2verify.QSlice = qSlice
			poDR2verify.MU = make([][]byte, len(proofs[i].Mu))
			for j := 0; j < len(proofs[i].Mu); j++ {
				poDR2verify.MU[j] = append(poDR2verify.MU[j], proofs[i].Mu[j]...)
			}
			poDR2verify.Sigma = proofs[i].Sigma
			poDR2verify.T = tag.T
			result := poDR2verify.PoDR2ProofVerify(puk.Shared_g, puk.Spk, string(puk.Shared_params))

			code = 0
			ts := time.Now().Unix()
			for code != int(configs.Code_200) && code != int(configs.Code_600) {
				code, err = chain.PutProofResult(configs.C.CtrlPrk, proofs[i].Miner_id, proofs[i].Challenge_info.File_id, result)
				if err == nil {
					Out.Sugar().Infof("[%v][%v]Proof result submitted successfully", uint64(proofs[i].Miner_id), result)
					break
				}
				if time.Since(time.Unix(ts, 0)).Minutes() > 10.0 {
					Err.Sugar().Errorf("[%v][%v]Proof result submitted failed:%v", uint64(proofs[i].Miner_id), result, err)
					break
				}
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 20)))
			}
		}
	}
}

func task_RecoveryFiles(ch chan bool) {
	var (
		recoverFlag  bool
		index        int
		recoverIndex int
		fileid       string
		filename     string
		filebasedir  string
		fileFullPath string
		mDatas       []chain.CessChain_AllMinerInfo
	)
	defer func() {
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
		ch <- true
	}()
	Out.Info(">>>Start task_RecoveryFiles task<<<")
	for {
		time.Sleep(time.Second * time.Duration(tools.RandomInRange(10, 30)))
		recoverylist, _, err := chain.GetFileRecoveryByAcc(configs.C.CtrlPrk)
		if err != nil {
			continue
		}
		for i := 0; i < len(recoverylist); i++ {
			ext := filepath.Ext(string(recoverylist[i]))
			fileid = strings.TrimSuffix(string(recoverylist[i]), ext)
			fmeta, _, err := chain.GetFileMetaInfoOnChain(fileid)
			if err != nil {
				Err.Sugar().Errorf("%v", err)
				continue
			}
			// query all miner
			for {
				mDatas, _, err = chain.GetAllMinerDataOnChain()
				if err == nil {
					break
				}
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(10, 30)))
			}

			filebasedir = filepath.Join(configs.FileCacheDir, fileid)

			_, err = os.Stat(filebasedir)
			if err != nil {
				err = os.Mkdir(filebasedir, os.ModeDir)
				if err != nil {
					Err.Sugar().Errorf("%v", err)
					continue
				}
			}
			index = 0
			recoverIndex = 0
			for d := 0; d < len(fmeta.FileDupl); d++ {
				if string(fmeta.FileDupl[d].DuplId) == filename {
					recoverIndex = d
					break
				}
			}
			recoverFlag = false
			filename = string(recoverylist[i])
			fileFullPath = filepath.Join(filebasedir, filename)
			fi, err := os.Stat(fileFullPath)
			if err == nil {
				for {
					var randkey types.Bytes
					filedump := make([]chain.FileDuplicateInfo, 1)
					randkey = fmeta.FileDupl[recoverIndex].RandKey
					if len(randkey) == 0 {
						break
					}
					f, err := os.OpenFile(fileFullPath, os.O_RDONLY, os.ModePerm)
					if err != nil {
						Err.Sugar().Errorf("[%v][%v]", fileFullPath, err)
						continue
					}
					blockTotal := fi.Size() / configs.RpcFileBuffer
					if fi.Size()%configs.RpcFileBuffer > 0 {
						blockTotal += 1
					}
					var blockinfo = make([]chain.BlockInfo, blockTotal)
					var failminer = make(map[uint64]bool, 0)
					var mip = ""
					for j := int64(0); j < blockTotal; j++ {
						_, err := f.Seek(int64(j*2*1024*1024), 0)
						if err != nil {
							Err.Sugar().Errorf("[%v][%v]", fileFullPath, err)
							f.Close()
							continue
						}
						var buf = make([]byte, configs.RpcFileBuffer)
						n, err := f.Read(buf)
						if err != nil {
							Err.Sugar().Errorf("[%v][%v]", fileFullPath, err)
							f.Close()
							continue
						}

						var bo = p.PutFileToBucket{
							FileId:     string(recoverylist[i]),
							FileHash:   "",
							BlockTotal: uint32(blockTotal),
							BlockSize:  uint32(n),
							BlockIndex: uint32(j),
							BlockData:  buf[:n],
						}
						bob, err := proto.Marshal(&bo)
						if err != nil {
							Err.Sugar().Errorf("[%v][%v]", fileFullPath, err)
							f.Close()
							continue
						}
						for {
							if mip == "" {
								index = tools.RandomInRange(0, len(mDatas))
								_, ok := failminer[uint64(mDatas[index].Peerid)]
								if ok {
									continue
								}
								_, err = rpc.WriteData(string(mDatas[index].Ip), configs.RpcService_Miner, configs.RpcMethod_Miner_WriteFile, bob)
								if err == nil {
									mip = string(mDatas[index].Ip)
									blockinfo[j].BlockIndex, _ = tools.IntegerToBytes(uint32(j))
									blockinfo[j].BlockSize = types.U32(uint32(n))
									break
								} else {
									failminer[uint64(mDatas[index].Peerid)] = true
									Err.Sugar().Errorf("[%v][%v]", fileFullPath, err)
									time.Sleep(time.Second * time.Duration(tools.RandomInRange(2, 5)))
								}
							} else {
								_, err = rpc.WriteData(mip, configs.RpcService_Miner, configs.RpcMethod_Miner_WriteFile, bob)
								if err != nil {
									failminer[uint64(mDatas[index].Peerid)] = true
									Err.Sugar().Errorf("[%v][%v]", fileFullPath, err)
									time.Sleep(time.Second * time.Duration(tools.RandomInRange(2, 5)))
									continue
								}
								blockinfo[j].BlockIndex, _ = tools.IntegerToBytes(uint32(j))
								blockinfo[j].BlockSize = types.U32(uint32(n))
								break
							}
						}
					}
					f.Close()
					filedump[0].DuplId = types.Bytes([]byte(string(recoverylist[i])))
					filedump[0].RandKey = randkey
					filedump[0].MinerId = mDatas[index].Peerid
					filedump[0].MinerIp = mDatas[index].Ip
					filedump[0].ScanSize = types.U32(configs.ScanBlockSize)
					//mips[i] = string(mDatas[index].Ip)
					// Query miner information by id
					var mdetails chain.Chain_MinerDetails
					for {
						mdetails, _, err = chain.GetMinerDetailsById(uint64(mDatas[index].Peerid))
						if err != nil {
							Err.Sugar().Errorf("[%v]%v", uint64(mDatas[index].Peerid), err)
							time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
							continue
						}
						break
					}
					filedump[0].Acc = mdetails.Address
					filedump[0].BlockNum = types.U32(uint32(blockTotal))
					filedump[0].BlockInfo = blockinfo
					// Upload the file meta information to the chain and write it to the cache
					for {
						_, err = chain.PutMetaInfoToChain(configs.C.CtrlPrk, fileid, filedump)
						if err != nil {
							Err.Sugar().Errorf("[%v][%v]", fileid, err)
							time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
							continue
						}
						Out.Sugar().Infof("[%v]The copy recovery meta information is successfully uploaded to the chain", fileid)
						// c, err := cache.GetCache()
						// if err != nil {
						// 	Err.Sugar().Errorf("[%v][%v][%v]", t, fileid, err)
						// } else {
						// 	b, err := json.Marshal(filedump)
						// 	if err != nil {
						// 		Err.Sugar().Errorf("[%v][%v][%v]", t, fileid, err)
						// 	} else {
						// 		err = c.Put([]byte(fid), b)
						// 		if err != nil {
						// 			Err.Sugar().Errorf("[%v][%v][%v]", t, fileid, err)
						// 		} else {
						// 			Out.Sugar().Infof("[%v][%v]File metainfo write cache success", t, fid)
						// 		}
						// 	}
						// }
						break
					}

					// calculate file tag info
					var PoDR2commit proof.PoDR2Commit
					var commitResponse proof.PoDR2CommitResponse
					PoDR2commit.FilePath = fileFullPath
					PoDR2commit.BlockSize = configs.BlockSize
					commitResponseCh, err := PoDR2commit.PoDR2ProofCommit(proof.Key_Ssk, string(proof.Key_SharedParams), int64(configs.ScanBlockSize))
					if err != nil {
						Err.Sugar().Errorf("[%v]%v", fileid, err)
						break
					}
					select {
					case commitResponse = <-commitResponseCh:
					}
					if commitResponse.StatueMsg.StatusCode != proof.Success {
						Err.Sugar().Errorf("[%v][%v]", fileid, err)
						break
					}
					var resp p.PutTagToBucket
					resp.FileId = string(recoverylist[i])
					resp.Name = commitResponse.T.Name
					resp.N = commitResponse.T.N
					resp.U = commitResponse.T.U
					resp.Signature = commitResponse.T.Signature
					resp.Sigmas = commitResponse.Sigmas
					resp_proto, err := proto.Marshal(&resp)
					if err != nil {
						Err.Sugar().Errorf("[%v]%v", fileid, err)
						break
					}
					_, err = rpc.WriteData(string(mDatas[index].Ip), configs.RpcService_Miner, configs.RpcMethod_Miner_WriteFileTag, resp_proto)
					if err != nil {
						Err.Sugar().Errorf("[%v]%v", fileid, err)
						break
					}

					_, err = chain.ClearRecoveredFileNoChain(configs.C.CtrlPrk, recoverylist[i])
					if err != nil {
						Err.Sugar().Errorf("[%v]%v", fileid, err)
						break
					}
					Out.Sugar().Infof("[%v] File recovery succeeded", string(recoverylist[i]))
					recoverFlag = true
					break
				}
			}
			if recoverFlag {
				continue
			}
			newFilename := fileid + ".u"
			fileuserfullname := filepath.Join(filebasedir, newFilename)
			_, err = os.Stat(fileuserfullname)
			// download dupl
			if err != nil {
				for k := 0; k < len(fmeta.FileDupl); k++ {
					if string(fmeta.FileDupl[k].DuplId) == filename {
						continue
					}
					filename = string(fmeta.FileDupl[k].DuplId)
					fileFullPath = filepath.Join(filebasedir, filename)
					_, err = os.Stat(fileFullPath)
					if err != nil {
						err = rpc.ReadFile(string(fmeta.FileDupl[k].MinerIp), filebasedir, filename, "")
						if err != nil {
							Err.Sugar().Errorf("[%v]%v", string(fmeta.FileDupl[k].DuplId), err)
							continue
						}
					}

					// decryption dupl file
					_, err = os.Stat(fileFullPath)
					if err == nil {
						buf, err := ioutil.ReadFile(fileFullPath)
						if err != nil {
							Err.Sugar().Errorf("[%v]%v", fileFullPath, err)
							os.Remove(fileFullPath)
							continue
						}
						//aes decryption
						ivkey := string(fmeta.FileDupl[k].RandKey)[:16]
						bkey := base58.Decode(string(fmeta.FileDupl[k].RandKey))
						decrypted, err := encryption.AesCtrDecrypt(buf, []byte(bkey), []byte(ivkey))
						if err != nil {
							Err.Sugar().Errorf("[%v]%v", fileFullPath, err)
							os.Remove(fileFullPath)
							continue
						}
						fr, err := os.OpenFile(fileuserfullname, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
						if err != nil {
							Err.Sugar().Errorf("[%v]%v", fileuserfullname, err)
							continue
						}
						fr.Write(decrypted)
						err = fr.Sync()
						if err != nil {
							Err.Sugar().Errorf("[%v]%v", fileuserfullname, err)
							fr.Close()
							os.Remove(fileuserfullname)
							continue
						}
						fr.Close()
					}
				}
			}
			_, err = os.Stat(fileuserfullname)
			if err != nil {
				Err.Sugar().Errorf("[%v] File recovery failed", fileid)
				continue
			}

			buf, err := os.ReadFile(fileuserfullname)
			if err != nil {
				os.Remove(fileuserfullname)
				Err.Sugar().Errorf("[%v] File recovery failed: %v", fileid, err)
				continue
			}

			// Generate 32-bit random key for aes encryption
			key := tools.GetRandomkey(32)
			key_base58 := base58.Encode([]byte(key))
			// Aes ctr mode encryption
			encrypted, err := encryption.AesCtrEncrypt(buf, []byte(key), []byte(key_base58[:16]))
			if err != nil {
				Err.Sugar().Errorf("[%v] File recovery failed: %v", fileid, err)
				continue
			}
			duplname := string(recoverylist[i])

			duplFallpath := filepath.Join(filebasedir, duplname)
			duplf, err := os.OpenFile(duplFallpath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, os.ModePerm)
			if err != nil {
				Err.Sugar().Errorf("[%v] File recovery failed: %v", fileid, err)
				continue
			}
			_, err = duplf.Write(encrypted)
			if err != nil {
				Err.Sugar().Errorf("[%v] File recovery failed: %v", fileid, err)
				duplf.Close()
				os.Remove(duplFallpath)
				continue
			}
			err = duplf.Sync()
			if err != nil {
				Err.Sugar().Errorf("[%v] File recovery failed: %v", fileid, err)
				duplf.Close()
				os.Remove(duplFallpath)
				continue
			}
			duplf.Close()
			duplkey := key_base58 + ".k" + strconv.Itoa(recoverIndex)
			duplkeyFallpath := filepath.Join(filebasedir, duplkey)
			_, err = os.Create(duplkeyFallpath)
			if err != nil {
				os.Remove(duplFallpath)
				Err.Sugar().Errorf("[%v] File recovery failed: %v", fileid, err)
				continue
			}

			for {
				fi, err = os.Stat(duplFallpath)
				if err != nil {
					Err.Sugar().Errorf("[%v] File recovery failed: %v", fileid, err)
					break
				}
				filedump := make([]chain.FileDuplicateInfo, 1)
				f, err := os.OpenFile(duplFallpath, os.O_RDONLY, os.ModePerm)
				if err != nil {
					Err.Sugar().Errorf("[%v] File recovery failed: %v", fileid, err)
					break
				}
				blockTotal := fi.Size() / configs.RpcFileBuffer
				if fi.Size()%configs.RpcFileBuffer > 0 {
					blockTotal += 1
				}
				var blockinfo = make([]chain.BlockInfo, blockTotal)
				var failminer = make(map[uint64]bool, 0)
				var mip = ""
				for j := int64(0); j < blockTotal; j++ {
					_, err := f.Seek(int64(j*2*1024*1024), 0)
					if err != nil {
						Err.Sugar().Errorf("[%v] File recovery failed: %v", fileid, err)
						f.Close()
						break
					}
					var buf = make([]byte, configs.RpcFileBuffer)
					n, err := f.Read(buf)
					if err != nil {
						Err.Sugar().Errorf("[%v] File recovery failed: %v", fileid, err)
						f.Close()
						break
					}

					var bo = p.PutFileToBucket{
						FileId:     string(recoverylist[i]),
						FileHash:   "",
						BlockTotal: uint32(blockTotal),
						BlockSize:  uint32(n),
						BlockIndex: uint32(j),
						BlockData:  buf[:n],
					}
					bob, err := proto.Marshal(&bo)
					if err != nil {
						Err.Sugar().Errorf("[%v] File recovery failed: %v", fileid, err)
						f.Close()
						break
					}
					for {
						if mip == "" {
							index = tools.RandomInRange(0, len(mDatas))
							_, ok := failminer[uint64(mDatas[index].Peerid)]
							if ok {
								continue
							}
							_, err = rpc.WriteData(string(mDatas[index].Ip), configs.RpcService_Miner, configs.RpcMethod_Miner_WriteFile, bob)
							if err == nil {
								mip = string(mDatas[index].Ip)
								blockinfo[j].BlockIndex, _ = tools.IntegerToBytes(uint32(j))
								blockinfo[j].BlockSize = types.U32(uint32(n))
								break
							} else {
								failminer[uint64(mDatas[index].Peerid)] = true
								Err.Sugar().Errorf("[%v][%v]", fileFullPath, err)
								time.Sleep(time.Second * time.Duration(tools.RandomInRange(2, 5)))
							}
						} else {
							_, err = rpc.WriteData(mip, configs.RpcService_Miner, configs.RpcMethod_Miner_WriteFile, bob)
							if err != nil {
								failminer[uint64(mDatas[index].Peerid)] = true
								Err.Sugar().Errorf("[%v][%v]", fileFullPath, err)
								time.Sleep(time.Second * time.Duration(tools.RandomInRange(2, 5)))
								continue
							}
							blockinfo[j].BlockIndex, _ = tools.IntegerToBytes(uint32(j))
							blockinfo[j].BlockSize = types.U32(uint32(n))
							break
						}
					}
				}
				f.Close()
				filedump[0].DuplId = recoverylist[i]
				filedump[0].RandKey = types.Bytes(key_base58)
				filedump[0].MinerId = mDatas[index].Peerid
				filedump[0].MinerIp = mDatas[index].Ip
				filedump[0].ScanSize = types.U32(configs.ScanBlockSize)
				//mips[i] = string(mDatas[index].Ip)
				// Query miner information by id
				var mdetails chain.Chain_MinerDetails
				for {
					mdetails, _, err = chain.GetMinerDetailsById(uint64(mDatas[index].Peerid))
					if err != nil {
						Err.Sugar().Errorf("[%v]%v", uint64(mDatas[index].Peerid), err)
						time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
						continue
					}
					break
				}
				filedump[0].Acc = mdetails.Address
				filedump[0].BlockNum = types.U32(uint32(blockTotal))
				filedump[0].BlockInfo = blockinfo
				// Upload the file meta information to the chain and write it to the cache
				for {
					_, err = chain.PutMetaInfoToChain(configs.C.CtrlPrk, fileid, filedump)
					if err != nil {
						Err.Sugar().Errorf("[%v][%v]", fileid, err)
						time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 10)))
						continue
					}
					Out.Sugar().Infof("[%v]The copy recovery meta information is successfully uploaded to the chain", fileid)
					// c, err := cache.GetCache()
					// if err != nil {
					// 	Err.Sugar().Errorf("[%v][%v][%v]", t, fileid, err)
					// } else {
					// 	b, err := json.Marshal(filedump)
					// 	if err != nil {
					// 		Err.Sugar().Errorf("[%v][%v][%v]", t, fileid, err)
					// 	} else {
					// 		err = c.Put([]byte(fid), b)
					// 		if err != nil {
					// 			Err.Sugar().Errorf("[%v][%v][%v]", t, fileid, err)
					// 		} else {
					// 			Out.Sugar().Infof("[%v][%v]File metainfo write cache success", t, fid)
					// 		}
					// 	}
					// }
					break
				}

				// calculate file tag info
				var PoDR2commit proof.PoDR2Commit
				var commitResponse proof.PoDR2CommitResponse
				PoDR2commit.FilePath = duplFallpath
				PoDR2commit.BlockSize = configs.BlockSize
				commitResponseCh, err := PoDR2commit.PoDR2ProofCommit(proof.Key_Ssk, string(proof.Key_SharedParams), int64(configs.ScanBlockSize))
				if err != nil {
					Err.Sugar().Errorf("[%v]%v", fileid, err)
					break
				}
				select {
				case commitResponse = <-commitResponseCh:
				}
				if commitResponse.StatueMsg.StatusCode != proof.Success {
					Err.Sugar().Errorf("[%v][%v]", fileid, err)
					break
				}
				var resp p.PutTagToBucket
				resp.FileId = string(recoverylist[i])
				resp.Name = commitResponse.T.Name
				resp.N = commitResponse.T.N
				resp.U = commitResponse.T.U
				resp.Signature = commitResponse.T.Signature
				resp.Sigmas = commitResponse.Sigmas
				resp_proto, err := proto.Marshal(&resp)
				if err != nil {
					Err.Sugar().Errorf("[%v]%v", fileid, err)
					break
				}
				_, err = rpc.WriteData(string(mDatas[index].Ip), configs.RpcService_Miner, configs.RpcMethod_Miner_WriteFileTag, resp_proto)
				if err != nil {
					Err.Sugar().Errorf("[%v]%v", fileid, err)
					break
				}
				_, err = chain.ClearRecoveredFileNoChain(configs.C.CtrlPrk, recoverylist[i])
				if err != nil {
					Err.Sugar().Errorf("[%v]%v", fileid, err)
					break
				}
				Out.Sugar().Infof("[%v] File recovery succeeded", string(recoverylist[i]))
				break
			}
		}
	}
}
