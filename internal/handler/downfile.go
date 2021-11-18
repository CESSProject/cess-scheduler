package handler

import (
	"fmt"
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
	_, _, err := VerifyToken(tk)
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("%v", err)
	}
	if err != nil {
		rsp.Msg = "Token validation failed.."
		c.JSON(http.StatusForbidden, rsp)
		return
	}
	sendFile(c)
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
