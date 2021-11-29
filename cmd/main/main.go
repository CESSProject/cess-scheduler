package main

import (
	"scheduler-mining/initlz"
	"scheduler-mining/internal/handler"
	"scheduler-mining/internal/proof"
)

// program entry
func main() {
	// init
	initlz.SystemInit()

	// start-up
	proof.Chain_Main()

	// web service
	handler.Handler_main()
}
