package initlz

import (
	"cess-scheduler/internal/chain"
	"cess-scheduler/internal/logger"
	"cess-scheduler/internal/rpc"
)

func SystemInit() {
	rpc.Rpc_Init()
	logger.LoggerInit()
	chain.Chain_Init()
}
