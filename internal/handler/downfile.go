package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin/binding"
	"io/ioutil"
	"net/http"
	"net/url"
	"scheduler-mining/configs"
	"scheduler-mining/internal/logger"
	"scheduler-mining/internal/rotation"

	"github.com/gin-gonic/gin"
)

type DownloadRq struct {
	Token string `json:"token"`
	Hash  string `json:"hash"`
}

func DownloadHandler(c *gin.Context) {
	var rsp = configs.RespMsg{
		Code: -1,
		Msg:  "",
		Data: nil,
	}
	var query DownloadRq
	if err := c.ShouldBindBodyWith(&query, binding.JSON); err != nil {
		rsp.Msg = "Parsing request failed"
		c.JSON(http.StatusBadRequest, rsp)
		return
	}
	fmt.Println(query.Token)
	_, ext, err := VerifyToken(query.Token)
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("%v", err)
		rsp.Msg = "Token validation failed.."
		c.JSON(http.StatusForbidden, rsp)
		return
	}
	// sendFile(c)
	// return
	//fmt.Printf("Start download [%v] .......\n", hash)
	logger.InfoLogger.Sugar().Infof("[downfile] [%v] [%v]", ext.FileName, query.Hash)
	getRes, err := rotation.BalanceCon.Client.Get(context.Background(), query.Hash)
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("Get [%v] of etcd failed, %v", query.Hash, err)
		rsp.Msg = "unexpected system error"
		c.JSON(http.StatusInternalServerError, rsp)
		return
	}
	if len(getRes.Kvs) == 0 {
		logger.ErrLogger.Sugar().Errorf("No miner was found to store the file[%v]", query.Hash)
		rsp.Msg = "Not found"
		c.JSON(http.StatusNotFound, rsp)
		return
	}
	for _, v := range getRes.Kvs {
		//fmt.Println("etcd: ", k, string(v.Value))
		peerurl := fmt.Sprintf("http://%v/downfile/%s?filename=%v", string(v.Value), query.Hash, url.QueryEscape(ext.FileName))
		//fmt.Println(peerurl)
		req, err := http.NewRequest(http.MethodGet, peerurl, nil)
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("%v", err)
			continue
		}
		var resp *http.Response
		transport := http.Transport{
			DisableKeepAlives: true,
		}
		cli := http.Client{
			Transport: &transport,
		}
		resp, err = cli.Do(req)
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("%v", err)
			continue
		}

		if resp != nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				content, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					logger.ErrLogger.Sugar().Errorf("%v", err)
					continue
				}
				var proveInfo configs.RespMsg
				err = json.Unmarshal(content, &proveInfo)
				if err != nil {
					logger.ErrLogger.Sugar().Errorf("%v", err)
					continue
				}
				//fmt.Println("---ok----", proveInfo.Data.(string))
				logger.InfoLogger.Sugar().Infof("[downfile addr][%v][%v]", query.Hash, proveInfo.Data.(string))
				rsp.Code = 0
				rsp.Msg = "sucess"
				rsp.Data = proveInfo.Data
				c.JSON(http.StatusOK, rsp)
				return
			} else {
				logger.ErrLogger.Sugar().Errorf("resp code: %v", resp.StatusCode)
			}
		} else {
			logger.ErrLogger.Error("resp is nil")
		}
	}
	rsp.Msg = "Unexpected error"
	c.JSON(http.StatusNotAcceptable, rsp)
}
