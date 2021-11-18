package initlz

import (
	"scheduler-mining/internal/cmdline"
	"scheduler-mining/internal/logger"
)

func SystemInit() {
	cmdline.CmdlineInit()
	logger.LoggerInit()
}
