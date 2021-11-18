package configs

// type and version
const Version = "CESS-Scheduler-Mining_0.5.0_Alpha"

// system exit code
const (
	Exit_Normal                   = 0
	Exit_LoginFailed              = -1
	Exit_RunningSystemError       = -2
	Exit_ExecutionPermissionError = -3
	Exit_InvalidIP                = -4
	Exit_CreateFolder             = -5
	Exit_CreateEmptyFile          = -6
	Exit_ConfFileNotExist         = -7
	Exit_ConfFileFormatError      = -8
	Exit_ConfFileTypeError        = -9
	Exit_CmdLineParaErr           = -10
)

var (
	LogfilePathPrefix = "./log/"
)
