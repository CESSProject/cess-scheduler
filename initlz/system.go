package initlz

import (
	"scheduler-mining/internal/chain"
	"scheduler-mining/internal/fileauth"
	"scheduler-mining/internal/logger"
)

func SystemInit() {
	logger.LoggerInit()
	chain.Chain_Init()
	fileauth.FileAuth_Init()
}
