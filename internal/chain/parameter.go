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
	SegmentBook_ParamSet      = "ParamSet"
	SegmentBook_ConProofInfoA = "ConProofInfoA"
	SegmentBook_UnVerifiedA   = "UnVerifiedA"
	SegmentBook_UnVerifiedB   = "UnVerifiedB"
	SegmentBook_UnVerifiedC   = "UnVerifiedC"
	SegmentBook_UnVerifiedD   = "UnVerifiedD"
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
	ChainTx_FileBank_Upload          = "FileBank.upload"
	ChainTx_FileBank_HttpDeleteFile  = "FileBank.http_delete"
	ChainTx_FileBank_FillerMap       = "FileBank.FillerMap"
	SegmentBook_VerifyProof          = "SegmentBook.verify_proof"
)

type Chain_MinerItems struct {
	Peerid      types.U64       `json:"peerid"`
	Beneficiary types.AccountID `json:"beneficiary"`
	Ip          types.U32       `json:"ip"`
	Collaterals types.U128      `json:"collaterals"`
	Earnings    types.U128      `json:"earnings"`
	Locked      types.U128      `json:"locked"`
	Publickey   types.Bytes     `json:"publickey"`
}

type CessChain_AllMinerInfo struct {
	Peerid types.U64   `json:"peerid"`
	Ip     types.Bytes `json:"ip"`
	Power  types.U128  `json:"power"`
	Space  types.U128  `json:"space"`
}

type ParamInfo struct {
	Peer_id    types.U64 `json:"peer_id"`
	Segment_id types.U64 `json:"segment_id"`
	Rand       types.U32 `json:"rand"`
}

type IpostParaInfo struct {
	Peer_id    types.U64   `json:"peer_id"`
	Segment_id types.U64   `json:"segment_id"`
	Size_type  types.U128  `json:"size_type"`
	Sealed_cid types.Bytes `json:"sealed_cid"`
}

type UnVerifiedVpaVpb struct {
	Accountid  types.AccountID `json:"acc"`
	Peer_id    types.U64       `json:"peer_id"`
	Segment_id types.U64       `json:"segment_id"`
	Proof      types.Bytes     `json:"proof"`
	Sealed_cid types.Bytes     `json:"sealed_cid"`
	Rand       types.U32       `json:"rand"`
	Size_type  types.U128      `json:"size_type"`
}

type UnVerifiedVpc struct {
	Accountid    types.AccountID `json:"acc"`
	Peer_id      types.U64       `json:"peer_id"`
	Segment_id   types.U64       `json:"segment_id"`
	Proof        []types.Bytes   `json:"proof"`
	Sealed_cid   []types.Bytes   `json:"sealed_cid"`
	Unsealed_cid []types.Bytes   `json:"unsealed_cid"`
	Rand         types.U32       `json:"rand"`
	Size_type    types.U128      `json:"size_type"`
}

type UnVerifiedVpd struct {
	Accountid  types.AccountID `json:"acc"`
	Peer_id    types.U64       `json:"peer_id"`
	Segment_id types.U64       `json:"segment_id"`
	Proof      []types.Bytes   `json:"proof"`
	Sealed_cid []types.Bytes   `json:"sealed_cid"`
	Rand       types.U32       `json:"rand"`
	Size_type  types.U128      `json:"size_type"`
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

// type FileDuplicateInfo struct {
// 	DuplId    types.Bytes     `json:"dupl_id"`    //Backup id
// 	RandKey   types.Bytes     `json:"rand_key"`   //Random key
// 	SliceNum  types.U16       `json:"slice_num"`  //Number of slices
// 	FileSlice []FileSliceInfo `json:"file_slice"` //Slice information list
// }

// type FileSliceInfo struct {
// 	SliceId   types.Bytes   `json:"slice_id"`   //Slice id
// 	SliceSize types.U32     `json:"slice_size"` //Slice size
// 	SliceHash types.Bytes   `json:"slice_hash"` //Slice hash
// 	FileShard FileShardInfo `json:"file_shard"` //Shard information
// }

// type FileShardInfo struct {
// 	DataShardNum  types.U8      `json:"data_shard_num"`  //Number of data shard
// 	RedunShardNum types.U8      `json:"redun_shard_num"` //Number of redundant shard
// 	ShardHash     []types.Bytes `json:"shard_hash"`      //Shard hash list
// 	ShardAddr     []types.Bytes `json:"shard_addr"`      //Store miner service addr list
// 	Peerid        []types.U64   `json:"wallet_addr"`     //Store miner wallet addr list
// }

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
	BlockIndex types.U32
	BlockSize  types.U32
}

type Chain_MinerDetails struct {
	Address                           types.AccountID
	Beneficiary                       types.AccountID
	Temp_power                        types.U128
	Power                             types.U128
	Space                             types.U128
	Total_reward                      types.U128
	Total_rewards_currently_available types.U128
	Totald_not_receive                types.U128
	Collaterals                       types.U128
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
	Segment_size types.U64
	File_type    types.U8
	Block_list   []types.U32
	File_id      types.Bytes
	Random       []types.Bytes
}
