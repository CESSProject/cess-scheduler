package initlz

import (
	"scheduler-mining/internal/chain"
	"scheduler-mining/internal/logger"
)

func SystemInit() {
	logger.LoggerInit()
	chain.Chain_Init()
}
