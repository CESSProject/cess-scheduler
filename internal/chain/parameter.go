package chain

import (
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
	FileBank_UserSpaceInfo    = "UserHoldSpaceDetails"
	FileBank_UserFilelistInfo = "UserHoldFileList"
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
)

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

type Cache_MinerInfo struct {
	Peerid uint64 `json:"peerid"`
	Ip     string `json:"ip"`
	Pubkey []byte `json:"pubkey"`
}

type FileMetaInfo struct {
	FileSize    types.U64         `json:"File_size"`
	BlockNum    types.U32         `json:"Block_num"`
	ScanSize    types.U32         `json:"Scan_size"`
	SegmentSize types.U32         `json:"Segment_size"`
	MinerAcc    types.AccountID   `json:"Miner_acc"`
	MinerIp     types.Bytes       `json:"Miner_ip"`
	Users       []types.AccountID `json:"Users"`
	Names       []types.Bytes     `json:"Names"`
	FileState   types.Bytes       `json:"File_state"`
}

type SchedulerInfo struct {
	Ip             types.Bytes
	StashUser      types.AccountID
	ControllerUser types.AccountID
}

type CessChain_EtcdItems struct {
	Ip types.Bytes `json:"ip"`
}

type SpaceFileInfo struct {
	FileSize  types.U64
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

type Chain_Proofs struct {
	Miner_pubkey   types.AccountID
	Challenge_info ChallengeInfo
	Mu             []types.Bytes
	Sigma          types.Bytes
}

type ChallengeInfo struct {
	File_size    types.U64
	Segment_size types.U32
	File_type    types.U8
	Block_list   []types.Bytes
	File_id      types.Bytes
	Random       []types.Bytes
}

//---user space Info
type UserSpaceInfo struct {
	PurchasedSpace types.U128
	UsedSpace      types.U128
	RemainingSpace types.U128
}

//
type VerifyResult struct {
	Miner_pubkey types.AccountID
	FileId       types.Bytes
	Result       types.Bool
}
