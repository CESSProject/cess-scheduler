/*
   Copyright 2022 CESS scheduler authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

        http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package chain

import (
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

const (
	ERR_Failed = "Failed"
	//ERR_Timeout = "Timeout"
	//ERR_Empty = "Empty"
)

// error type
var (
	ERR_RPC_CONNECTION  = errors.New("rpc connection failed")
	ERR_RPC_TIMEOUT     = errors.New("timeout")
	ERR_RPC_EMPTY_VALUE = errors.New("empty")
)

type FileHash [64]types.U8
type FileBlockId [68]types.U8

// storage miner info
type MinerInfo struct {
	PeerId      types.U64
	IncomeAcc   types.AccountID
	Ip          Ipv4Type
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
	Free   uint64 `json:"free"`
}

// file meta info
type FileMetaInfo struct {
	Size  types.U64
	Index types.U32
	State types.Bytes
	// Users []types.AccountID
	// Names []types.Bytes
	UserBriefs []UserBrief
	BlockInfo  []BlockInfo
}

type UserBrief struct {
	User        types.AccountID
	File_name   types.Bytes
	Bucket_name types.Bytes
}

// file block info
type BlockInfo struct {
	MinerId   types.U64
	BlockSize types.U64
	BlockNum  types.U32
	BlockId   FileBlockId
	MinerIp   Ipv4Type
	MinerAcc  types.AccountID
}

// filler meta info
type FillerMetaInfo struct {
	Size      types.U64
	Index     types.U32
	BlockNum  types.U32
	BlockSize types.U32
	ScanSize  types.U32
	Acc       types.AccountID
	Hash      FileHash
}

// scheduler info
type SchedulerInfo struct {
	Ip             Ipv4Type
	StashUser      types.AccountID
	ControllerUser types.AccountID
}

type Ipv4Type_Query struct {
	Placeholder types.U8 //
	Index       types.U8
	Value       [4]types.U8
	Port        types.U16
}

type IpAddress struct {
	IPv4 Ipv4Type
	IPv6 Ipv6Type
}
type Ipv4Type struct {
	Index types.U8
	Value [4]types.U8
	Port  types.U16
}
type Ipv6Type struct {
	Index types.U8
	Value [8]types.U16
	Port  types.U16
}

// proof type
type Proof struct {
	FileId         FileHash
	Miner_pubkey   types.AccountID
	Challenge_info ChallengeInfo
	U              types.Bytes
	Mu             types.Bytes
	Sigma          types.Bytes
	Omega          types.Bytes
	SigRootHash    types.Bytes
	HashMi         []types.Bytes
}

// challenge info
type ChallengeInfo struct {
	File_size  types.U64
	File_type  types.U8
	Block_list types.Bytes
	File_id    FileHash
	Shard_id   FileBlockId
	Random     []types.Bytes
}

// user space package Info
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

// proof result
type ProofResult struct {
	PublicKey types.AccountID
	FileId    FileHash
	Shard_id  FileBlockId
	Result    types.Bool
}
