package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"scheduler-mining/configs"
	"scheduler-mining/internal/logger"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

func UploadHandler(c *gin.Context) {
	var rsp = configs.RespMsg{
		Code: -1,
		Msg:  "",
		Data: nil,
	}
	tk := c.Query("token")
	tobj, ext, err := VerifyToken(tk)
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("%v", err)
		rsp.Msg = err.Error()
		c.JSON(http.StatusUnauthorized, rsp)
		return
	}

	fmt.Printf("extdData: %+v\n", ext)

	fname, hash, path, _, err := localSaveFile(c, tobj)
	if err != nil {
		rsp.Msg = "failed"
		c.JSON(http.StatusBadRequest, rsp)
		return
	}
	renameFile(path, configs.CacheFilePath, hash)

	fileSuffix := filepath.Ext(fname)
	fileSuffix = fileSuffix
	//TODO
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
		sameNameFileHash = calcFileHash(path)
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
		hash := calcFileHash(path)
		size := calcFileSize(path)
		if sameNameFileHash != "" || sameNameFileSize != 0 {
			if hash == sameNameFileHash && sameNameFileSize == size {
				os.Remove(path)
				rsp.Code = 0
				rsp.Msg = "success"
				c.JSON(http.StatusOK, rsp)
				oldpath := filepath.Join(configs.CacheFilePath, file.Filename)
				CallBack(to, fmt.Sprintf("%d", size), "", hash, "0")
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
		CallBack(to, fmt.Sprintf("%d", size), "", hash, "0")
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

func calcFileHash(fpath string) string {
	f, err := os.Open(fpath)
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("%v", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		logger.ErrLogger.Sugar().Errorf("%v", err)
	}

	return hex.EncodeToString(h.Sum(nil))
}

func renameFile(oldpath, sectorpath, hash string) {
	dstpath := filepath.Join(sectorpath, hash)
	os.Rename(oldpath, dstpath)
}
