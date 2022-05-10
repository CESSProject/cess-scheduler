package configs

// type and version
const Version = "cess-scheduler_V0.4.0"

// rpc
const (
	RpcService_Scheduler         = "wservice"
	RpcService_Miner             = "mservice"
	RpcMethod_Miner_WriteFile    = "writefile"
	RpcMethod_Miner_ReadFile     = "readfile"
	RpcMethod_Miner_WriteFileTag = "writefiletag"
	RpcFileBuffer                = 2 * 1024 * 1024
)

// return code
const (
	Code_200 = 200
	Code_400 = 400
	Code_403 = 403
	Code_404 = 404
	Code_500 = 500
)

// space file
const (
	LengthOfALine = 128
	BlockSize     = 1024 * 1024 //1MB
	ScanBlockSize = 512 * 1024  //512KB
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

	//data dir
	LogFileDir    = "log"
	FileCacheDir  = "file"
	DbFileDir     = "db"
	SpaceCacheDir = "space"
)
