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
	Sminer_MinerDetails       = "MinerDetails"
	FileMap_SchedulerPuk      = "SchedulerPuk"
	SegmentBook_UnVerifyProof = "UnVerifyProof"
	FileBank_FileRecovery     = "FileRecovery"
)

// cess chain Transaction name
const (
	ChainTx_FileBank_Update         = "FileBank.update"
	ChainTx_FileMap_Add_schedule    = "FileMap.registration_scheduler"
	ChainTx_FileBank_PutMetaInfo    = "FileBank.update_dupl"
	ChainTx_FileBank_Upload         = "FileBank.upload"
	ChainTx_FileBank_HttpDeleteFile = "FileBank.http_delete"
	ChainTx_FileBank_UploadFiller   = "FileBank.upload_filler"
	SegmentBook_VerifyProof         = "SegmentBook.verify_proof"
	FileBank_ClearRecoveredFile     = "FileBank.recover_file"
)

type Chain_MinerItems struct {
	Peerid      types.U64
	Beneficiary types.AccountID
	Ip          types.Bytes
	Collaterals types.U128
	Earnings    types.U128
	Locked      types.U128
	State       types.Bytes
	Power       types.U128
	Space       types.U128
	Publickey   types.Bytes
}

type CessChain_AllMinerInfo struct {
	Peerid types.U64   `json:"peerid"`
	Ip     types.Bytes `json:"ip"`
	Power  types.U128  `json:"power"`
	Space  types.U128  `json:"space"`
}

type FileMetaInfo struct {
	//FileId      types.Bytes         `json:"acc"`         //File id
	File_name   types.Bytes         `json:"file_name"`   //File name
	FileSize    types.U64           `json:"file_size"`   //File size
	FileHash    types.Bytes         `json:"file_hash"`   //File hash
	Public      types.Bool          `json:"public"`      //Public or not
	UserAddr    types.AccountID     `json:"user_addr"`   //Upload user's address
	FileState   types.Bytes         `json:"file_state"`  //File state
	Backups     types.U8            `json:"backups"`     //Number of backups
	Downloadfee types.U128          `json:"downloadfee"` //Download fee
	FileDupl    []FileDuplicateInfo `json:"file_dupl"`   //File backup information list
}

type FileDuplicateInfo struct {
	MinerId   types.U64
	BlockNum  types.U32
	ScanSize  types.U32
	Acc       types.AccountID
	MinerIp   types.Bytes
	DuplId    types.Bytes
	RandKey   types.Bytes
	BlockInfo []BlockInfo
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
	MinerId   types.U64
	FileSize  types.U64
	BlockNum  types.U32
	ScanSize  types.U32
	Acc       types.AccountID
	BlockInfo []BlockInfo
	FileId    types.Bytes
	FileHash  types.Bytes
}
type BlockInfo struct {
	BlockIndex types.Bytes
	BlockSize  types.U32
}

type Chain_MinerDetails struct {
	Address                           types.AccountID
	Beneficiary                       types.AccountID
	ServiceAddr                       types.Bytes
	Power                             types.U128
	Space                             types.U128
	Total_reward                      types.U128
	Total_rewards_currently_available types.U128
	Totald_not_receive                types.U128
}

type Chain_SchedulerPuk struct {
	Spk           types.Bytes
	Shared_params types.Bytes
	Shared_g      types.Bytes
}

type Chain_Proofs struct {
	Miner_id       types.U64
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
