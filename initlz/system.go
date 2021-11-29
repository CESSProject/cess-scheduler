package initlz

import (
	"scheduler-mining/internal/chain"
	"scheduler-mining/internal/cmdline"
	"scheduler-mining/internal/fileauth"
	"scheduler-mining/internal/logger"
)

func SystemInit() {
	cmdline.CmdlineInit()
	logger.LoggerInit()
	chain.Chain_Init()
	fileauth.FileAuth_Init()
}
