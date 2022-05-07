package configs

// type and version
const Version = "cess-scheduler_V0.4.0"

const (
	// base dir
	BaseDir = "/usr/local/cess-scheduler"

	// log file dir
	LogFilePath = BaseDir + "/log"
	// file cache dir
	CacheFilePath = BaseDir + "/cache"
	// database dir
	DbFilePath = BaseDir + "/db"
)

const (
	RpcService_Scheduler      = "wservice"
	RpcService_Miner          = "mservice"
	RpcMethod_Miner_WriteFile = "writefile"
	RpcMethod_Miner_ReadFile  = "readfile"
)

const (
	SegMentSize_1M            = 1048576
	TimeToWaitEvents_S        = 20
	SegMentType_Idle    uint8 = 1
	SegMentType_Service uint8 = 2
	SegMentType_8M      uint8 = 1
	SegMentType_8M_S          = "1"
	SegMentType_512M    uint8 = 2
	SegMentType_512M_S        = "2"
)

var (
	MinSegMentSize = 8323072
	RduShards      = 2
	CurrentPath    = ""
	FilePostProof  = 6
)
