package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"scheduler-mining/configs"
	"scheduler-mining/internal/logger"

	"github.com/gin-gonic/gin"
)

func DownloadHandler(c *gin.Context) {
	var rsp = configs.RespMsg{
		Code: -1,
		Msg:  "",
		Data: nil,
	}

	tk := c.Query("token")
	_, ext, err := VerifyToken(tk)
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("%v", err)
		rsp.Msg = "Token validation failed.."
		c.JSON(http.StatusForbidden, rsp)
		return
	}
	// sendFile(c)
	// return
	hash := c.Param("hash")
	fmt.Printf("Start download [%v] .......\n", hash)
	peerurl := fmt.Sprintf("http://172.16.2.191:15001/downfile/%s?filename=%v", hash, ext.FileName)
	req, err := http.NewRequest(http.MethodGet, peerurl, nil)
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("%v", err)
		rsp.Msg = "Request error"
		c.JSON(http.StatusInternalServerError, rsp)
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("%v", err)
		rsp.Msg = "Unexpected error"
		c.JSON(http.StatusNotAcceptable, rsp)
		return
	}

	if resp != nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			content, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				logger.ErrLogger.Sugar().Errorf("%v", err)
				rsp.Msg = "Miner net error"
				c.JSON(http.StatusNotAcceptable, rsp)
				return
			}
			var proveInfo configs.RespMsg
			err = json.Unmarshal(content, &proveInfo)
			if err != nil {
				logger.ErrLogger.Sugar().Errorf("%v", err)
				rsp.Msg = "Miner data error"
				c.JSON(http.StatusNotAcceptable, rsp)
				return
			}
			fmt.Println("---ok----", proveInfo.Data.(string))
			rsp.Code = 0
			rsp.Msg = "sucess"
			rsp.Data = proveInfo.Data
			c.JSON(http.StatusOK, rsp)
			return
		}
	}
	rsp.Msg = "Unexpected error"
	c.JSON(http.StatusNotAcceptable, rsp)
}

func sendFile(c *gin.Context) {
	filehash := c.Param("hash")
	path := filepath.Join(configs.CacheFilePath, filehash)
	if _, err := os.Stat(path); err == nil {
		c.Writer.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filehash))
		c.Writer.Header().Add("Content-Type", "application/octet-stream")
		c.File(path)
		logger.InfoLogger.Sugar().Infof("A file was downloaded:%v", filehash)
		return
	}

	logger.ErrLogger.Error("File not exists...")

	var rsp = configs.RespMsg{
		Code: -1,
		Msg:  "File not exists...",
		Data: nil,
	}
	c.JSON(http.StatusNotAcceptable, rsp)
}
