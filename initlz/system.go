package initlz

import (
	"scheduler-mining/internal/chain"
	"scheduler-mining/internal/logger"
	"scheduler-mining/rpc"
)

func SystemInit() {
	rpc.Rpc_Init()
	logger.LoggerInit()
	chain.Chain_Init()
}
