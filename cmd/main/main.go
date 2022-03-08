package main

import (
	"scheduler-mining/cmd"
	"scheduler-mining/initlz"
	"scheduler-mining/rpc"

	"scheduler-mining/internal/proof"
)

// program entry
func main() {
	cmd.Execute()

	// init
	initlz.SystemInit()

	// start-up
	proof.Chain_Main()

	// rpc service
	rpc.Rpc_Main()
}
