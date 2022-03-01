package main

import (
	"scheduler-mining/cmd"
	"scheduler-mining/initlz"

	"scheduler-mining/internal/proof"
)

// program entry
func main() {
	cmd.Execute()

	// init
	initlz.SystemInit()

	// start-up
	proof.Chain_Main()
	//tools.CleanLocalRecord(".etcd")

	select {}
	// web service
	//handler.Handler_main()
}
