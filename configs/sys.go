package configs

// type and version
const Version = "cess-scheduler v0.4.1"

// rpc
const (
	RpcService_Scheduler         = "wservice"
	RpcService_Miner             = "mservice"
	RpcMethod_Miner_WriteFile    = "writefile"
	RpcMethod_Miner_ReadFile     = "readfile"
	RpcMethod_Miner_WriteFileTag = "writefiletag"
	RpcMethod_Miner_ReadFileTag  = "readfiletag"
	RpcFileBuffer                = 64 * 1024 //64KB
	RpcSpaceBuffer               = 16 * 1024 //16KB
)

// return state code
const (
	Code_200 = 200
	Code_400 = 400
	Code_403 = 403
	Code_404 = 404
	Code_500 = 500
	Code_600 = 600
)

//
const (
	LengthOfALine            = 4096
	BlockSize                = 1024 * 1024 //1MB
	ScanBlockSize            = 512 * 1024  //512KB
	ByteSize_1Kb             = 1024        //1KB
	TimeToWaitEvents_S       = 20
	Backups_Min        uint8 = 3
	Backups_Max        uint8 = 6
	BaseDir                  = "scheduler"
	NewTestAddr              = true
)

var (
	//data dir
	LogFileDir    = "log"
	FileCacheDir  = "file"
	DbFileDir     = "db"
	SpaceCacheDir = "space"
)
