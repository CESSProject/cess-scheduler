package main

import (
	"scheduler-mining/initlz"
	"scheduler-mining/internal/handler"
)

// program entry
func main() {
	// init
	initlz.SystemInit()

	// start-up

	// web service
	handler.Handler_main()
}
