package handler

import (
	"bytes"
	"context"
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
	"strings"
	"time"

	"scheduler-mining/internal/rotation"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

type PeerStorageInfo struct {
	Code     int    `json:"code"`
	Msg      string `json:"msg"`
	FileSize uint64 `json:"filesize"`
	FileHash string `json:"filehash"`
}
type UploadRq struct {
	Token string `json:"token"`
}

/// Analyze the information of each specific file in multiple file uploads
type FileHeader struct {
	ContentDisposition string
	Name               string
	FileName           string
	ContentType        string
	ContentLength      int64
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
	tk := c.PostForm("token")

	tobj, ext, err := VerifyToken(tk)
	fmt.Println(tk)
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("%v", err)
		rsp.Msg = err.Error()
		c.JSON(http.StatusUnauthorized, rsp)
		return
	}
	fname, hash, path, size, err := localSaveFile(c)
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("upload file failed err: %v", err)
		rsp.Msg = "upload file failed err:" + err.Error()
		c.JSON(http.StatusBadRequest, rsp)
		return
	}
	//fmt.Printf("extdData: %+v\n", ext)
	logger.InfoLogger.Sugar().Infof("[upfile] %+v", ext)
	renameFile(path, configs.CacheFilePath, hash)

	fileSuffix := filepath.Ext(fname)
	filePath := filepath.Join(configs.CacheFilePath, hash)

	mDatas := chain.GetMinersDate()
	freeMiners := getFreeMiners(mDatas, size)
	if len(freeMiners) == 0 {
		logger.ErrLogger.Sugar().Errorf("No miners storage file: %v", hash)
		rsp.Msg = "No miners storage file"
		c.JSON(http.StatusForbidden, rsp)
		return
	} else {
		CallBack(tobj, fmt.Sprintf("%d", size), "", hash, "0", "")
		rsp.Code = 0
		rsp.Msg = "success"
		c.JSON(http.StatusOK, rsp)
	}

	go func(suf string, fi string, t *Policy, fid string) {
		simhash, hashtype, err := fileauth.GetFileSimhash(suf, fi)
		if err != nil {
			logger.InfoLogger.Sugar().Infof("Simhash failed [fid:%v][%v]", fid, err)
			CallBack2(t, fmt.Sprintf("%d", size), "", hash, "-1", hashtype, fid)
		} else {
			logger.InfoLogger.Sugar().Infof("Simhash suc [fid:%v][%v]", fid, simhash)
			CallBack2(t, fmt.Sprintf("%d", size), "", hash, simhash, hashtype, fid)
		}
	}(fileSuffix, filePath, tobj, ext.FileId)
	go schedulerFileOnUpload(freeMiners, hash, size, ext, filePath)
}

func schedulerFileOnUpload(mDatas []chain.CessChain_AllMinerItems, hash string, size uint64, ext AddExt, filePath string) {
	var (
		err               error
		dataShards        int
		rduShards         int
		shardsPath        = ""
		shardfilenamelist []string
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
				logger.ErrLogger.Sugar().Errorf("MkdirAll for shards file failed,[%v] [%v]", ext.FileId, err)
				return
			}
		}
		if dataShards < 4 {
			rduShards = 1
		} else {
			rduShards = 2
		}
		_, shardfilenamelist, err = fileshards.Shards(filePath, shardsPath, dataShards, rduShards)
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("Shards file failed,[%v] [%v]", ext.FileId, err)
			return
		}
		if len(shardfilenamelist) != (dataShards + rduShards) {
			logger.ErrLogger.Sugar().Errorf("Shards file num failed,[%v] [%v]", ext.FileId, err)
			return
		}
		fmt.Println(shardfilenamelist)
	}

	count := 0
	if len(mDatas) > int(ext.BackUpNum) {
		count = int(ext.BackUpNum)
	} else {
		count = len(mDatas)
	}
	sortMiners(mDatas)

	isStored := false
	bkpNum := 0
	fdspath := "/" + hash
	for i := 0; i < len(mDatas); i++ {
		if bkpNum >= count {
			break
		}
		isStored = false
		ip := tools.Uint32ToIp(uint32(mDatas[i].Ip))
		ports := strconv.Itoa(int(mDatas[i].FilePort))
		serviceAddr := ip + ":" + ports
		url := "http://" + serviceAddr + "/group1/upload"

		serport := strconv.Itoa(int(mDatas[i].Port))
		serAddr := ip + ":" + serport
		getRes, err := rotation.BalanceCon.Client.Get(context.Background(), hash)
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("Get [%v] of etcd failed, %v", hash, err)
		} else {
			for _, v := range getRes.Kvs {
				if serAddr == string(v.Value) {
					isStored = true
					bkpNum++
					logger.InfoLogger.Sugar().Infof("Already Put [%v][%v] to etcd suc", hash, serAddr)
					break
				}
			}
		}
		if isStored {
			continue
		}
		ok := tools.TestConnectionWithTcp(serviceAddr)
		if ok {
			if dataShards > 0 {
				for k := 0; k < (dataShards + rduShards); k++ {
					f, err := os.Stat(filepath.Join(configs.CurrentPath, shardfilenamelist[k]))
					if err != nil {
						logger.ErrLogger.Sugar().Errorf("[%v] not found", shardfilenamelist[k])
						break
					}
					params := map[string]string{
						"file":   f.Name(),
						"output": "json",
						"path":   fdspath,
					}
					err = upfileToFDS(url, shardfilenamelist[k], params)
					if err != nil {
						logger.ErrLogger.Sugar().Errorf("Save to DFS failed,[%v],%v", hash, err)
						continue
					} else {
						_, unsealedCIDs, _ := proof.GetPrePoRep(shardfilenamelist[k])
						if unsealedCIDs != nil {
							var uncid = make([][]byte, len(unsealedCIDs))
							for m := 0; m < len(unsealedCIDs); m++ {
								uncid[m] = make([]byte, 0)
								uncid[m] = append(uncid[m], []byte(unsealedCIDs[m].String())...)
							}
							for {
								err = chain.IntentSubmitToChain(
									configs.Confile.MinerData.IdAccountPhraseOrSeed,
									configs.ChainTx_SegmentBook_IntentSubmit,
									uint8(1),
									uint8(2),
									uint64(mDatas[i].Peerid),
									uncid,
									[]byte(hash),
									[]byte(f.Name()),
								)
								if err != nil {
									logger.ErrLogger.Sugar().Errorf("[C%v] submit uncid failed [%v] [err:%v]", mDatas[i].Peerid, hash, err)
									time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 15)))
								} else {
									logger.InfoLogger.Sugar().Infof("[C%v] submit uncid suc [%v]", mDatas[i].Peerid, hash)
									break
								}
							}
						} else {
							logger.ErrLogger.Sugar().Errorf("Calc unsealedCIDs failed [%v]", shardfilenamelist[k])
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
					logger.ErrLogger.Sugar().Errorf("Save to DFS failed,[%v],%v", hash, err)
					continue
				} else {
					_, unsealedCIDs, _ := proof.GetPrePoRep(path)
					if unsealedCIDs != nil {
						var uncid = make([][]byte, len(unsealedCIDs))
						for m := 0; m < len(unsealedCIDs); m++ {
							uncid[m] = make([]byte, 0)
							uncid[m] = append(uncid[m], []byte(unsealedCIDs[m].String())...)
						}
						for {
							err = chain.IntentSubmitToChain(
								configs.Confile.MinerData.IdAccountPhraseOrSeed,
								configs.ChainTx_SegmentBook_IntentSubmit,
								uint8(1),
								uint8(2),
								uint64(mDatas[i].Peerid),
								uncid,
								[]byte(hash),
								[]byte(hash),
							)
							if err != nil {
								logger.ErrLogger.Sugar().Errorf("[C%v] submit uncid failed [%v] [err:%v]", mDatas[i].Peerid, hash, err)
								time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 15)))
							} else {
								logger.InfoLogger.Sugar().Infof("[C%v] submit uncid suc [%v]", mDatas[i].Peerid, hash)
								break
							}
						}
					} else {
						logger.ErrLogger.Sugar().Errorf("Calc unsealedCIDs failed [%v]", path)
					}
				}
			}
			_, err = rotation.BalanceCon.Client.Put(context.Background(), hash, serAddr)
			if err != nil {
				logger.ErrLogger.Sugar().Errorf("Put [%v] to etcd failed, %v", hash, err)
				continue
			} else {
				bkpNum++
				logger.InfoLogger.Sugar().Infof("Put [%v][%v] to etcd suc", hash, serAddr)
			}
		} else {
			logger.ErrLogger.Sugar().Errorf("The network is blocked when the file is backed up to [%v],[%v] [%v]", mDatas[i].Peerid, serviceAddr, hash)
		}
	}
}

func localSaveFile(c *gin.Context) (string, string, string, uint64, error) {
	var sameNameFileHash string = ""
	var sameNameFileSize uint64 = 0
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

	if err = SaveBigFile2(c, path); err != nil {
		logger.ErrLogger.Sugar().Errorf("%v", err)
		return "", "", "", 0, err
	} else {
		hash, _ := tools.CalcFileHash(path)
		size := calcFileSize(path)
		//fmt.Println("path:", path)
		//fmt.Println("hash:", hash)
		if sameNameFileHash != "" || sameNameFileSize != 0 {
			if hash == sameNameFileHash && sameNameFileSize == size {
				os.Remove(path)
				oldpath := filepath.Join(configs.CacheFilePath, file.Filename)
				//CallBack(to, fmt.Sprintf("%d", size), "", hash, "0", "")
				return filename, hash, oldpath, size, nil
			}
		}
		if hash == "" || size == 0 || path == "" {
			logger.ErrLogger.Error("file para failed")
			return "", "", "", 0, errors.New("file para failed")
		}
		//CallBack(to, fmt.Sprintf("%d", size), "", hash, "0", "")
		return filename, hash, path, size, nil
	}
}

func SaveBigFile2(c *gin.Context, path string) error {
	var content_length int64
	content_length = c.Request.ContentLength
	if content_length <= 0 {
		return errors.New("empty file")
	}
	file, _, _ := c.Request.FormFile("file")
	f, err := os.Create(path)
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("create file fail:%v\n", err)
		return err
	}
	defer f.Close()
	for {
		buf := make([]byte, 1024*4)
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			fmt.Println(err)
			return err
		}
		if n == 0 {
			break
		}
		f.Write(buf[:n])
	}
	return nil
}

func SaveBigFile(c *gin.Context, path string) error {
	var content_length int64
	content_length = c.Request.ContentLength
	if content_length <= 0 || content_length > 1024*1024*1024*1024 {
		logger.ErrLogger.Sugar().Errorf("Uploading the file is too big")
		return errors.New("Uploading the file is too big")
	}
	content_type_, has_key := c.Request.Header["Content-Type"]
	if !has_key {
		logger.ErrLogger.Sugar().Errorf("No file uploaded, please upload file")
		return errors.New("No file uploaded, please upload file")
	}
	if len(content_type_) != 1 {
		logger.ErrLogger.Sugar().Errorf("Upload file error")
		return errors.New("Upload file error")
	}
	content_type := content_type_[0]
	const BOUNDARY string = "; boundary="
	loc := strings.Index(content_type, BOUNDARY)
	if -1 == loc {
		logger.ErrLogger.Sugar().Errorf("Content-Type error, no boundary")
		return errors.New("Content-Type error, no boundary")
	}
	boundary := []byte(content_type[(loc + len(BOUNDARY)):])
	logger.InfoLogger.Sugar().Infof("[FileBoundary]:[%s]\n", boundary)
	//Read length
	read_data := make([]byte, 1024*12)
	var read_total int = 0
	for {
		file_header, file_data, err := ParseFromHead(read_data, read_total, append(boundary, []byte("\r\n")...), c)
		if err != nil {
			fmt.Println(err)
			logger.ErrLogger.Sugar().Errorf("Parse the header of the file:%v", err)
			return err
		}
		logger.InfoLogger.Sugar().Infof("file :%s\n", file_header.FileName)

		f, err := os.Create(path)
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("create file fail:%v\n", err)
			return err
		}
		f.Write(file_data)
		file_data = nil

		//Need to search boundary repeatedly
		temp_data, reach_end, err := ReadToBoundary(boundary, c.Request.Body, f)
		f.Close()
		if err != nil {
			logger.ErrLogger.Sugar().Infof("read to boundary error:%v\n", err)
			return err
		}
		if reach_end {
			break
		} else {
			copy(read_data[0:], temp_data)
			read_total = len(temp_data)
			continue
		}
	}
	return nil
}

/// Parsing header of description file information
/// @return FileHeader Structure of file name and other information
/// @return bool Resolution success or failure
func ParseFileHeader(h []byte) (FileHeader, bool) {
	arr := bytes.Split(h, []byte("\r\n"))
	var out_header FileHeader
	out_header.ContentLength = -1
	const (
		CONTENT_DISPOSITION = "Content-Disposition: "
		NAME                = "name=\""
		FILENAME            = "filename=\""
		CONTENT_TYPE        = "Content-Type: "
		CONTENT_LENGTH      = "Content-Length: "
	)
	for _, item := range arr {
		if bytes.HasPrefix(item, []byte(CONTENT_DISPOSITION)) {
			l := len(CONTENT_DISPOSITION)
			arr1 := bytes.Split(item[l:], []byte("; "))
			out_header.ContentDisposition = string(arr1[0])
			if bytes.HasPrefix(arr1[1], []byte(NAME)) {
				out_header.Name = string(arr1[1][len(NAME) : len(arr1[1])-1])
			}
			l = len(arr1[2])
			if bytes.HasPrefix(arr1[2], []byte(FILENAME)) && arr1[2][l-1] == 0x22 {
				out_header.FileName = string(arr1[2][len(FILENAME) : l-1])
			}
		} else if bytes.HasPrefix(item, []byte(CONTENT_TYPE)) {
			l := len(CONTENT_TYPE)
			out_header.ContentType = string(item[l:])
		} else if bytes.HasPrefix(item, []byte(CONTENT_LENGTH)) {
			l := len(CONTENT_LENGTH)
			s := string(item[l:])
			content_length, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				logger.ErrLogger.Sugar().Infof("content length error:%s", string(item))
				return out_header, false
			} else {
				out_header.ContentLength = content_length
			}
		} else {
			logger.ErrLogger.Sugar().Infof("unknown:%s\n", string(item))
		}
	}
	if len(out_header.FileName) == 0 {
		return out_header, false
	}
	return out_header, true
}

/// Read from the stream to the last bit of the file
/// @return []byte Data that is not written to a file and belongs to the next file
/// @return bool Has the last bit of the stream been read
/// @return error
func ReadToBoundary(boundary []byte, stream io.ReadCloser, target io.WriteCloser) ([]byte, bool, error) {
	read_data := make([]byte, 1024*8)
	read_data_len := 0
	buf := make([]byte, 1024*4)
	b_len := len(boundary)
	reach_end := false
	for !reach_end {
		read_len, err := stream.Read(buf)
		if err != nil {
			if err != io.EOF && read_len <= 0 {
				return nil, true, err
			}
			reach_end = true
		}
		copy(read_data[read_data_len:], buf[:read_len]) //It is added to another buffer just for search convenience
		read_data_len += read_len
		if read_data_len < b_len+4 {
			continue
		}
		loc := bytes.Index(read_data[:read_data_len], boundary)
		if loc >= 0 {
			//End position found
			target.Write(read_data[:loc-4])
			return read_data[loc:read_data_len], reach_end, nil
		}

		target.Write(read_data[:read_data_len-b_len-4])
		copy(read_data[0:], read_data[read_data_len-b_len-4:])
		read_data_len = b_len + 4
	}
	target.Write(read_data[:read_data_len])
	return nil, reach_end, nil
}

/// Parse the header of the form
/// @param read_data Data that has been read from the stream
/// @param read_total The length of data that has been read from the stream
/// @param boundary Split string for the form
/// @param stream Input stream
/// @return FileHeader File name and other information headers
///			[]byte The part that has been read from the stream
///			error
func ParseFromHead(read_data []byte, read_total int, boundary []byte, c *gin.Context) (FileHeader, []byte, error) {
	file, _, _ := c.Request.FormFile("file")
	buf := make([]byte, 1024*4)
	found_boundary := false
	boundary_loc := -1
	var file_header FileHeader
	for {
		read_len, err := file.Read(buf)
		if err != nil {
			if err != io.EOF {
				return file_header, nil, err
			}
			break
		}
		if read_total+read_len > cap(read_data) {
			return file_header, nil, fmt.Errorf("not found boundary")
		}
		copy(read_data[read_total:], buf[:read_len])
		read_total += read_len
		if !found_boundary {
			boundary_loc = bytes.Index(read_data[:read_total], boundary)
			if -1 == boundary_loc {
				continue
			}
			found_boundary = true
		}
		start_loc := boundary_loc + len(boundary)
		file_head_loc := bytes.Index(read_data[start_loc:read_total], []byte("\r\n\r\n"))
		if -1 == file_head_loc {
			continue
		}
		file_head_loc += start_loc
		ret := false
		file_header, ret = ParseFileHeader(read_data[start_loc:file_head_loc])
		if !ret {
			return file_header, nil, fmt.Errorf("ParseFileHeader fail:%s", string(read_data[start_loc:file_head_loc]))
		}
		return file_header, read_data[file_head_loc+4 : read_total], nil
	}
	return file_header, nil, fmt.Errorf("reach to sream EOF")
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
		if (in[i].Power.Uint64()-in[i].Space.Uint64())*configs.SegMentSize_1M >= size {
			rtn = append(rtn, in[i])
		}
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
