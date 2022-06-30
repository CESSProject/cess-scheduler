package configs

// type and version
const Version = "cess-scheduler v0.4.3.220630 dev"

// rpc service and method
const (
	RpcService_Scheduler         = "wservice"
	RpcService_Miner             = "mservice"
	RpcMethod_Miner_WriteFile    = "writefile"
	RpcMethod_Miner_ReadFile     = "readfile"
	RpcMethod_Miner_WriteFileTag = "writefiletag"
	RpcMethod_Miner_ReadFileTag  = "readfiletag"
	RpcFileBuffer                = 1024 * 1024 //1MB
	RpcSpaceBuffer               = 512 * 1024  //512KB
)

// return code
const (
	//success
	Code_200 = 200
	//bad request
	Code_400 = 400
	//forbidden
	Code_403 = 403
	//not found
	Code_404 = 404
	//server internal error
	Code_500 = 500
	//The block was produced but the event was not resolved
	Code_600 = 600
)

//
const (
	LengthOfALine      = 4096
	BYTE_SIZE_1GB      = 1024 * 1024 * 1024
	BlockSize          = 1024 * 1024 //1MB
	ScanBlockSize      = 512 * 1024  //512KB
	ByteSize_1Kb       = 1024        //1KB
	TimeToWaitEvents_S = 20
	BaseDir            = "scheduler"
)

var (
	//data dir
	LogFileDir    = "log"
	FileCacheDir  = "file"
	DbFileDir     = "db"
	SpaceCacheDir = "space"
)
