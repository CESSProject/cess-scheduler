package main

import (
	"scheduler-mining/initlz"
	"scheduler-mining/internal/handler"
	"scheduler-mining/internal/proof"
	"scheduler-mining/internal/rotation"
)

// program entry
func main() {
	// init
	initlz.SystemInit()

	// start-up
	proof.Chain_Main()
	//tools.CleanLocalRecord(".etcd")
	go rotation.EtcdClusterRegister()
	// web service
	handler.Handler_main()
}
