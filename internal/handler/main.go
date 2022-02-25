package handler

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"net/http"
	_ "net/http/pprof"
	"scheduler-mining/configs"
	"scheduler-mining/internal/rotation"
)

func Handler_main() {
	r := gin.Default()
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true                                                                                                 //允许所有域名
	config.AllowMethods = []string{"GET", "POST", "OPTIONS"}                                                                      //允许请求的方法
	config.AllowHeaders = []string{"tus-resumable", "upload-length", "upload-metadata", "cache-control", "x-requested-with", "*"} //允许的Header
	r.Use(cors.New(config))

	//TODO:
	r.POST("/file/upload", UploadHandler)
	r.POST("/file/download", DownloadHandler)
	r.POST("/checkclusterspaceenough", rotation.CheckClusterSpaceEnough)
	r.POST("/detectjoininfo", rotation.DetectJoinInfo)
	r.POST("/testpolling", rotation.TestPolling)

	//start go-pprof
	go http.ListenAndServe("0.0.0.0:6060", nil)
	r.Run(":" + configs.Confile.MinerData.ServicePort)
}
