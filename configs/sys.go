package configs

// type and version
const Version = "CESS-Scheduler-Mining_0.3.0_Alpha"

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

// cess chain module
const (
	ChainModule_Sminer      = "Sminer"
	ChainModule_SegmentBook = "SegmentBook"
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
)

// cess chain Transaction name
const (
	ChainTx_SegmentBook_VerifyInVpa  = "SegmentBook.verify_in_vpa"
	ChainTx_SegmentBook_VerifyInVpb  = "SegmentBook.verify_in_vpb"
	ChainTx_SegmentBook_VerifyInVpc  = "SegmentBook.verify_in_vpc"
	ChainTx_SegmentBook_VerifyInVpd  = "SegmentBook.verify_in_vpd"
	ChainTx_SegmentBook_IntentSubmit = "SegmentBook.intent_submit"
	ChainTx_FileBank_Update          = "FileBank.update"
)

const (
	MinerUpfileUrl   = "/upfile"
	MinerDownfileUrl = "/downfile"
)

const (
	SegMentSize_1M = 1048576
)

var (
	LogfilePathPrefix = "./log/"
	CacheFilePath     = "./cess_filecache"
	MinSegMentSize    = 8323072
	RduShards         = 2
	CurrentPath       = ""
	FilePostProof     = 6
)

const (
	ChainModule_EtcdSminer           = "Sminer"
	ChainTransaction_Sminer_Register = "Sminer.setetcd"
	ChainModule_Sminer_EtcdRegister  = "EtcdRegister"

	ClusterToken_Sminer_Register   = "Sminer.setetcdtoken"
	ClusterToken_Sminer            = "Sminer"
	ClusterToken_Sminer_MinerItems = "EtcdToken"

	ServicePort_Sminer_Register   = "Sminer.setserviceport"
	ServicePort_Sminer            = "Sminer"
	ServicePort_Sminer_MinerItems = "ServicePort"
)
