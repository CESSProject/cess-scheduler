package configs

// type and version
const Version = "cess-scheduler_V0.3.1"

// cess chain module
const (
	ChainModule_Sminer      = "Sminer"
	ChainModule_SegmentBook = "SegmentBook"
	ChainModule_FileBank    = "FileBank"
	ChainModule_FileMap     = "FileMap"
)

// cess chain module method
const (
	ChainModule_Sminer_AllMinerItems      = "AllMiner"
	ChainModule_Sminer_MinerItems         = "MinerItems"
	ChainModule_Sminer_SegInfo            = "SegInfo"
	ChainModule_SegmentBook_ParamSet      = "ParamSet"
	ChainModule_SegmentBook_ConProofInfoA = "ConProofInfoA"
	ChainModule_SegmentBook_UnVerifiedA   = "UnVerifiedA"
	ChainModule_SegmentBook_UnVerifiedB   = "UnVerifiedB"
	ChainModule_SegmentBook_UnVerifiedC   = "UnVerifiedC"
	ChainModule_SegmentBook_UnVerifiedD   = "UnVerifiedD"
	ChainModule_FileMap_FileMetaInfo      = "File"
	ChainModule_FileMap_SchedulerInfo     = "SchedulerMap"
)

// cess chain Transaction name
const (
	ChainTx_SegmentBook_VerifyInVpa  = "SegmentBook.verify_in_vpa"
	ChainTx_SegmentBook_VerifyInVpb  = "SegmentBook.verify_in_vpb"
	ChainTx_SegmentBook_VerifyInVpc  = "SegmentBook.verify_in_vpc"
	ChainTx_SegmentBook_VerifyInVpd  = "SegmentBook.verify_in_vpd"
	ChainTx_SegmentBook_IntentSubmit = "SegmentBook.intent_submit"
	ChainTx_FileBank_Update          = "FileBank.update"
	ChainTx_FileMap_Add_schedule     = "FileMap.registration_scheduler"
	ChainTx_FileBank_PutMetaInfo     = "FileBank.update_dupl"
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
	LogFilePath    = "/var/cesscache/log"
	CacheFilePath  = "/var/cesscache/cache"
	DbFilePath     = "/var/cesscache/db"
	MinSegMentSize = 8323072
	RduShards      = 2
	CurrentPath    = ""
	FilePostProof  = 6
)
