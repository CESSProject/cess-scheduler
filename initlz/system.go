package initlz

import (
	"log"
	"os"
	"runtime"
)

func init() {
	// Determine if the operating system is linux
	if runtime.GOOS != "linux" {
		log.Println("Please run on linux system.")
		os.Exit(1)
	}
	// Allocate all cores to the program
	runtime.GOMAXPROCS(runtime.NumCPU())
}
