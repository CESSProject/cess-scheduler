package handler

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"scheduler-mining/configs"
	"scheduler-mining/internal/chain"
	"scheduler-mining/internal/fileauth"
	"scheduler-mining/internal/fileshards"
	"scheduler-mining/internal/logger"
	"scheduler-mining/internal/proof"
	"scheduler-mining/tools"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type PeerStorageInfo struct {
	Code     int    `json:"code"`
	Msg      string `json:"msg"`
	FileSize uint64 `json:"filesize"`
	FileHash string `json:"filehash"`
}

func UploadHandler(c *gin.Context) {
	var (
		tobj *Policy
		rsp  = configs.RespMsg{
			Code: -1,
			Msg:  "",
			Data: nil,
		}
	)
	tk := c.Query("token")

	tobj, ext, err := VerifyToken(tk)
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("%v", err)
		rsp.Msg = err.Error()
		c.JSON(http.StatusUnauthorized, rsp)
		return
	}
	fmt.Printf("extdData: %+v\n", ext)

	fname, hash, path, size, err := localSaveFile(c, tobj)
	if err != nil {
		rsp.Msg = "upload file failed"
		c.JSON(http.StatusBadRequest, rsp)
		return
	}
	renameFile(path, configs.CacheFilePath, hash)

	fileSuffix := filepath.Ext(fname)
	filePath := filepath.Join(configs.CacheFilePath, hash)
	go func(suf string, fi string, t *Policy) {
		simhash, hashtype, err := fileauth.GetFileSimhash(suf, fi)
		if err != nil {
			CallBack2(t, fmt.Sprintf("%d", size), "", hash, "-1", hashtype)
		} else {
			CallBack2(t, fmt.Sprintf("%d", size), "", hash, simhash, hashtype)
		}
	}(fileSuffix, filePath, tobj)
	go schedulerFileOnUpload(hash, size, ext, filePath)
}

func schedulerFileOnUpload(hash string, size uint64, ext AddExt, filePath string) {
	var (
		err        error
		tryCount   uint8 = 0
		dataShards int
		rduShards  int
		shardsPath = ""
		mDatas     []chain.CessChain_AllMinerItems
	)

	if size > uint64(configs.MinSegMentSize) {
		dataShards = int(size) / configs.MinSegMentSize
		if int(size)%configs.MinSegMentSize != 0 {
			dataShards += 1
		}
		if dataShards > 248 {
			dataShards = 248
		}
	}

	if dataShards > 0 {
		shardsPath = hash + "_" + fmt.Sprintf("%v", time.Now().UTC().Nanosecond())
		shardsPath = filepath.Join(configs.CacheFilePath, shardsPath)
		_, err = os.Stat(shardsPath)
		if err != nil {
			err = os.MkdirAll(shardsPath, os.ModePerm)
			if err != nil {
				fmt.Printf("\x1b[%dm[err]\x1b[0m MkdirAll for shards file failed, %v\n", 41, err)
				logger.ErrLogger.Sugar().Errorf("%v", err)
				return
			}
		}
		if dataShards < 4 {
			rduShards = 1
		} else {
			rduShards = 2
		}
		_, err = fileshards.Shards(filePath, shardsPath, dataShards, rduShards)
		if err != nil {
			fmt.Printf("\x1b[%dm[err]\x1b[0m Shards file failed, %v\n", 41, err)
			logger.ErrLogger.Sugar().Errorf("%v", err)
			return
		}
	}

	mDatas, err = chain.GetAllMinerDataOnChain(
		chain.SubstrateAPI_Read(),
		configs.ChainModule_Sminer,
		configs.ChainModule_Sminer_AllMinerItems,
	)
	for {
		if err != nil {
			tryCount++
		} else {
			break
		}
		if tryCount < 3 {
			mDatas, err = chain.GetAllMinerDataOnChain(
				chain.SubstrateAPI_Read(),
				configs.ChainModule_Sminer,
				configs.ChainModule_Sminer_AllMinerItems,
			)
			if err == nil {
				break
			}
		}
		if tryCount >= 3 {
			if err != nil {
				fmt.Printf("\x1b[%dm[err]\x1b[0m Get miners for file failed, %v\n", 41, err)
				logger.ErrLogger.Sugar().Errorf("Get miners for file failed, %v", err)
				return
			}
		}
	}

	fmt.Println("All miners:", mDatas)
	freeMiners := getFreeMiners(mDatas, size)
	fmt.Println("Free miners:", freeMiners)
	if len(mDatas) == 0 || len(freeMiners) == 0 {
		fmt.Printf("\x1b[%dm[err]\x1b[0m No miners storage file: %v\n", 41, hash)
		logger.ErrLogger.Sugar().Errorf("No miners storage file: %v", hash)
		return
	}

	count := 0
	if len(freeMiners) > int(ext.BackUpNum) {
		count = int(ext.BackUpNum)
	} else {
		count = len(freeMiners)
	}
	sortMiners(freeMiners)

	bkpNum := 0
	fdspath := "/" + hash
	for i := 0; i < len(freeMiners); i++ {
		ip := tools.Uint32ToIp(uint32(freeMiners[i].Ip))
		ports := strconv.Itoa(int(freeMiners[i].FilePort))
		serviceAddr := ip + ":" + ports
		url := "http://" + serviceAddr + "/group1/upload"
		ok := tools.TestConnectionWithTcp(serviceAddr)
		if ok {
			fmt.Println("ip:", serviceAddr)
			bkpNum++
			if dataShards > 0 {
				rdunum := 0
				for k := 0; k < (dataShards + rduShards); k++ {
					shardname := hash + fmt.Sprintf(".%v", k)
					shardpath_tmp := filepath.Join(shardsPath, shardname)
					shardhash, err := tools.CalcFileHash(shardpath_tmp)
					if err != nil {
						logger.ErrLogger.Sugar().Errorf("[%v],%v", shardpath_tmp, err)
						break
					}
					hashname := shardhash + fmt.Sprintf(".%v", k)

					if (k + 1) > dataShards {
						//shardname = hash + fmt.Sprintf(".r%v", rdunum)
						hashname = shardhash + fmt.Sprintf(".r%v", rdunum)
						rdunum++
					}
					renameFile(shardpath_tmp, shardsPath, hashname)
					shardfullpath := filepath.Join(shardsPath, hashname)
					params := map[string]string{
						"file":   hashname,
						"output": "json",
						"path":   fdspath,
					}
					err = upfileToFDS(url, shardfullpath, params)
					if err != nil {
						logger.ErrLogger.Sugar().Errorf("[%v],%v", hashname, err)
					} else {
						_, unsealedCIDs, _ := proof.GetPrePoRep(shardfullpath)
						if unsealedCIDs != nil {
							var uncid = make([][]byte, len(unsealedCIDs))
							for m := 0; m < len(unsealedCIDs); m++ {
								uncid[m] = make([]byte, 0)
								uncid[m] = append(uncid[m], []byte(unsealedCIDs[m].String())...)
							}
							err = chain.IntentSubmitToChain(
								chain.SubstrateAPI_Write(),
								configs.Confile.MinerData.IdAccountPhraseOrSeed,
								configs.ChainTx_SegmentBook_IntentSubmit,
								uint8(1),
								uint8(2),
								uint64(freeMiners[i].Peerid),
								uncid,
								[]byte(hash),
								[]byte(hashname),
							)
							if err != nil {
								logger.ErrLogger.Sugar().Errorf("%v", err)
							} else {
								fmt.Printf("\x1b[%dm[ok]\x1b[0m [C%v] submit uncid \n", 42, freeMiners[i].Peerid)
							}
						}
					}
				}
			} else {
				path := filepath.Join(configs.CacheFilePath, hash)
				params := map[string]string{
					"file":   hash + ".cess",
					"output": "json",
					"path":   fdspath,
				}
				err = upfileToFDS(url, path, params)
				if err != nil {
					logger.ErrLogger.Sugar().Errorf("[%v],%v", hash, err)
				} else {
					_, unsealedCIDs, _ := proof.GetPrePoRep(path)
					if unsealedCIDs != nil {
						fmt.Println(len(unsealedCIDs))
						var uncid = make([][]byte, len(unsealedCIDs))
						for m := 0; m < len(unsealedCIDs); m++ {
							uncid[m] = make([]byte, 0)
							uncid[m] = append(uncid[m], []byte(unsealedCIDs[m].String())...)
						}
						err = chain.IntentSubmitToChain(
							chain.SubstrateAPI_Write(),
							configs.Confile.MinerData.IdAccountPhraseOrSeed,
							configs.ChainTx_SegmentBook_IntentSubmit,
							uint8(1),
							uint8(2),
							uint64(freeMiners[i].Peerid),
							uncid,
							[]byte(hash),
							[]byte(hash),
						)
						if err != nil {
							fmt.Printf("\x1b[%dm[err]\x1b[0m [C%v] submit uncid failed\n", 41, freeMiners[i].Peerid)
							logger.ErrLogger.Sugar().Errorf("%v", err)
						} else {
							fmt.Printf("\x1b[%dm[ok]\x1b[0m [C%v] submit uncid\n", 42, freeMiners[i].Peerid)
						}
					}
				}
			}
		}
		if bkpNum >= count {
			break
		}
	}
}

func localSaveFile(c *gin.Context, to *Policy) (string, string, string, uint64, error) {
	var sameNameFileHash string = ""
	var sameNameFileSize uint64 = 0
	var rsp = configs.RespMsg{
		Code: -1,
		Msg:  "",
		Data: nil,
	}
	file, err := c.FormFile("file")
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("%v", err)
		return "", "", "", 0, err
	}
	filename := file.Filename
	if _, err := os.Stat(configs.CacheFilePath); err != nil {
		if err := os.MkdirAll(configs.CacheFilePath, os.ModePerm); err != nil {
			logger.ErrLogger.Sugar().Errorf("%v", err)
			return "", "", "", 0, err
		}
	}

	path := filepath.Join(configs.CacheFilePath, file.Filename)
	if _, err = os.Stat(path); err == nil {
		sameNameFileHash, _ = tools.CalcFileHash(path)
		sameNameFileSize = calcFileSize(path)
		filesuffix := filepath.Ext(file.Filename)
		fileprefix := file.Filename[0 : len(file.Filename)-len(filesuffix)]
		path = filepath.Join(configs.CacheFilePath, fileprefix)
		path = path + fmt.Sprintf(".%v", strconv.FormatInt(time.Now().UnixNano(), 10))
		path = path + filesuffix
	}

	if err := c.SaveUploadedFile(file, path); err != nil {
		logger.ErrLogger.Sugar().Errorf("%v", err)
		return "", "", "", 0, err
	} else {
		hash, _ := tools.CalcFileHash(path)
		size := calcFileSize(path)
		if sameNameFileHash != "" || sameNameFileSize != 0 {
			if hash == sameNameFileHash && sameNameFileSize == size {
				os.Remove(path)
				rsp.Code = 0
				rsp.Msg = "success"
				c.JSON(http.StatusOK, rsp)
				oldpath := filepath.Join(configs.CacheFilePath, file.Filename)
				CallBack(to, fmt.Sprintf("%d", size), "", hash, "0", "")
				return filename, hash, oldpath, size, nil
			}
		}
		if hash == "" || size == 0 || path == "" {
			logger.ErrLogger.Error("file para failed")
			return "", "", "", 0, errors.New("file para failed")
		}
		rsp.Code = 0
		rsp.Msg = "success"
		c.JSON(http.StatusOK, rsp)
		CallBack(to, fmt.Sprintf("%d", size), "", hash, "0", "")
		return filename, hash, path, size, nil
	}
}

func calcFileSize(fpath string) uint64 {
	fi, err := os.Stat(fpath)
	if err == nil {
		return uint64(fi.Size())
	}
	return 0
}

func renameFile(oldpath, sectorpath, hash string) {
	dstpath := filepath.Join(sectorpath, hash)
	os.Rename(oldpath, dstpath)
}

func getFreeMiners(in []chain.CessChain_AllMinerItems, size uint64) []chain.CessChain_AllMinerItems {
	var rtn = make([]chain.CessChain_AllMinerItems, 0)
	if len(in) <= 0 {
		return rtn
	}
	for i := 0; i < len(in); i++ {
		//if (in[i].Power.Uint64()-in[i].Space.Uint64())*configs.SegMentSize_1M >= size {
		rtn = append(rtn, in[i])
		//}
	}
	return rtn
}

func sortMiners(in []chain.CessChain_AllMinerItems) {
	if len(in) < 2 {
		return
	}
	for i := 0; i < len(in)-1; i++ {
		for j := i + 1; j < len(in); j++ {
			if (in[i].Power.Uint64() - in[i].Space.Uint64()) < (in[j].Power.Uint64() - in[j].Space.Uint64()) {
				in[i], in[j] = in[j], in[i]
			}
		}
	}
}

func upfileToFDS(uri string, filename string, params map[string]string) error {
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", params["file"])
	if err != nil {
		return err
	}
	src, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer src.Close()

	_, err = io.Copy(part, src)
	if err != nil {
		return err
	}
	for key, val := range params {
		writer.WriteField(key, val)
	}
	err = writer.Close()
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", uri, body)
	if err != nil {
		return err
	}
	request.Header.Add("Content-Type", writer.FormDataContentType())
	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// content, err := ioutil.ReadAll(resp.Body)
	// if err != nil {
	// 	logger.ErrLogger.Sugar().Errorf("%v", err)
	// 	return  err
	// }
	// fmt.Println(string(content))
	return nil
}
