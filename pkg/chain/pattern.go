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
	"reflect"

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
}

// filler meta info
type FillerMetaInfo struct {
	Size      types.U64
	Index     types.U32
	BlockNum  types.U32
	BlockSize types.U32
	ScanSize  types.U32
	Acc       types.AccountID
	Hash      [68]types.U8
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

// proof result
type ProofResult struct {
	PublicKey types.AccountID
	FileId    types.Bytes
	Result    types.Bool
}

func (this BlockInfo) IsEmpty() bool {
	return reflect.DeepEqual(this, BlockInfo{})
}
