package chain

import (
	"reflect"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

// cess chain state
const (
	State_Sminer      = "Sminer"
	State_SegmentBook = "SegmentBook"
	State_FileBank    = "FileBank"
	State_FileMap     = "FileMap"
)

// cess chain module method
const (
	Sminer_AllMinerItems      = "AllMiner"
	Sminer_MinerItems         = "MinerItems"
	Sminer_SegInfo            = "SegInfo"
	FileMap_FileMetaInfo      = "File"
	FileMap_SchedulerInfo     = "SchedulerMap"
	FileBank_UserSpaceList    = "UserSpaceList"
	FileBank_PurchasedPackage = "PurchasedPackage"
	FileBank_UserFilelist     = "UserHoldFileList"
	Sminer_PurchasedSpace     = "PurchasedSpace"
	Sminer_TotalSpace         = "AvailableSpace"
	FileMap_SchedulerPuk      = "SchedulerPuk"
	SegmentBook_UnVerifyProof = "UnVerifyProof"
	FileBank_FileRecovery     = "FileRecovery"
)

// cess chain Transaction name
const (
	ChainTx_FileBank_Update       = "FileBank.update"
	ChainTx_FileMap_Add_schedule  = "FileMap.registration_scheduler"
	Tx_FileBank_Upload            = "FileBank.upload"
	ChainTx_FileBank_UploadFiller = "FileBank.upload_filler"
	SegmentBook_VerifyProof       = "SegmentBook.verify_proof"
	FileBank_ClearRecoveredFile   = "FileBank.recover_file"
	FileMap_UpdateScheduler       = "FileMap.update_scheduler"
)

// storage miner info
type MinerInfo struct {
	PeerId      types.U64
	IncomeAcc   types.AccountID
	Ip          types.Bytes
	Collaterals types.U128
	State       types.Bytes
	Power       types.U128
	Space       types.U128
	RewardInfo  RewardInfo
}

type RewardInfo struct {
	Total       types.U128
	Received    types.U128
	NotReceived types.U128
}

// cache storage miner
type Cache_MinerInfo struct {
	Peerid uint64 `json:"peerid"`
	Ip     string `json:"ip"`
	Pubkey []byte `json:"pubkey"`
}

// public user file meta info
type FileMetaInfo struct {
	FileSize  types.U64
	Index     types.U32
	FileState types.Bytes
	Users     []types.AccountID
	Names     []types.Bytes
	ChunkInfo []ChunkInfo
}

type ChunkInfo struct {
	MinerId   types.U64
	ChunkSize types.U64
	BlockNum  types.U32
	ChunkId   types.Bytes
	MinerIp   types.Bytes
	MinerAcc  types.AccountID
}

// scheduler info
type SchedulerInfo struct {
	Ip             types.Bytes
	StashUser      types.AccountID
	ControllerUser types.AccountID
}

// filler info
type SpaceFileInfo struct {
	FileSize  types.U64
	Index     types.U32
	BlockNum  types.U32
	BlockSize types.U32
	ScanSize  types.U32
	Acc       types.AccountID
	FileId    types.Bytes
	FileHash  types.Bytes
}

type Chain_SchedulerPuk struct {
	Spk           types.Bytes
	Shared_params types.Bytes
	Shared_g      types.Bytes
}

//
type Chain_Proofs struct {
	FileId         types.Bytes
	Miner_pubkey   types.AccountID
	Challenge_info ChallengeInfo
	Mu             []types.Bytes
	Sigma          types.Bytes
}

type ChallengeInfo struct {
	File_size  types.U64
	File_type  types.U8
	Block_list types.Bytes
	File_id    types.Bytes
	Random     []types.Bytes
}

// user space Info
type SpacePackage struct {
	Space           types.U128
	Used_space      types.U128
	Remaining_space types.U128
	Tenancy         types.U32
	Package_type    types.U8
	Start           types.U32
	Deadline        types.U32
	State           types.Bytes
}

//
type VerifyResult struct {
	Miner_pubkey types.AccountID
	FileId       types.Bytes
	Result       types.Bool
}

const (
	ERR_Failed  = "Failed"
	ERR_Timeout = "Timeout"
	ERR_Empty   = "Empty"
)

func (this ChunkInfo) IsEmpty() bool {
	return reflect.DeepEqual(this, ChunkInfo{})
}
