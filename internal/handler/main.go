package handler

import (
	"io/ioutil"
	"scheduler-mining/configs"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func Handler_main() {
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = ioutil.Discard
	r := gin.Default()
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true                                                                                                 //允许所有域名
	config.AllowMethods = []string{"GET", "POST", "OPTIONS"}                                                                      //允许请求的方法
	config.AllowHeaders = []string{"tus-resumable", "upload-length", "upload-metadata", "cache-control", "x-requested-with", "*"} //允许的Header
	r.Use(cors.New(config))

	//TODO:

	r.Run(":" + configs.Confile.MinerData.ServicePort)
}
