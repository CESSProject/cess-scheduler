package chain

import (
	"reflect"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

const (
	ERR_Failed  = "Failed"
	ERR_Timeout = "Timeout"
	ERR_Empty   = "Empty"
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

// file meta info
type FileMetaInfo struct {
	Size      types.U64
	Index     types.U32
	State     types.Bytes
	Users     []types.AccountID
	Names     []types.Bytes
	BlockInfo []BlockInfo
}

type BlockInfo struct {
	MinerId   types.U64
	BlockSize types.U64
	BlockNum  types.U32
	BlockId   types.Bytes
	MinerIp   types.Bytes
	MinerAcc  types.AccountID
	Tag       TagInfo
}

type TagInfo struct {
	Name   types.Bytes
	N      types.U64
	U      []types.Bytes
	Sigmas []types.Bytes
	Pkey   types.Bytes
	Sign   types.Bytes
}

// filler meta info
type FillerMetaInfo struct {
	Size      types.U64
	Index     types.U32
	BlockNum  types.U32
	BlockSize types.U32
	Acc       types.AccountID
	Id        types.Bytes
	Hash      types.Bytes
	Tag       TagInfo
}

// scheduler info
type SchedulerInfo struct {
	Ip             types.Bytes
	StashUser      types.AccountID
	ControllerUser types.AccountID
}

type Chain_SchedulerPuk struct {
	Spk           types.Bytes
	Shared_params types.Bytes
	Shared_g      types.Bytes
}

//
type Proof struct {
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

func (this BlockInfo) IsEmpty() bool {
	return reflect.DeepEqual(this, BlockInfo{})
}
