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
	"sync"
	"time"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type Chainer interface {
	// Getpublickey returns its own public key
	GetPublicKey() []byte
	// GetSyncStatus returns whether the block is being synchronized
	GetSyncStatus() (bool, error)
	// Getstorageminerinfo is used to get the details of the miner
	GetStorageMinerInfo(pkey []byte) (MinerInfo, error)
	// Getallstorageminer is used to obtain the AccountID of all miners
	GetAllStorageMiner() ([]types.AccountID, error)
	// GetFileMetaInfo is used to get the meta information of the file
	GetFileMetaInfo(fid types.Bytes) (FileMetaInfo, error)
	// GetAllSchedulerInfo is used to get information about all schedules
	GetAllSchedulerInfo() ([]SchedulerInfo, error)
	// GetProofs is used to get all the proofs to be verified
	GetProofs() ([]Proof, error)
	// GetCessAccount is used to get the account in cess chain format
	GetCessAccount() (string, error)
	// GetAccountInfo is used to get account information
	GetAccountInfo(pkey []byte) (types.AccountInfo, error)
	// GetSpacePackageInfo is used to get the space package information of the account
	GetSpacePackageInfo(pkey []byte) (SpacePackage, error)
	// Register is used by the scheduling service to register
	Register(stash, contact string) (string, error)
	// SubmitProofResults is used to submit proof verification results
	SubmitProofResults(data []ProofResult) (string, error)
	// SubmitFillerMeta is used to submit the meta information of the filler
	SubmitFillerMeta(miner_acc types.AccountID, info []FillerMetaInfo) (string, error)
	// SubmitFileMeta is used to submit the meta information of the file
	SubmitFileMeta(fid string, fsize uint64, block []BlockInfo) (string, error)
	// Update is used to update the communication address of the scheduling service
	Update(contact string) (string, error)
}

type chainClient struct {
	l               *sync.Mutex
	c               *gsrpc.SubstrateAPI
	metadata        *types.Metadata
	keyEvents       types.StorageKey
	runtimeVersion  *types.RuntimeVersion
	genesisHash     types.Hash
	keyring         signature.KeyringPair
	rpcAddr         string
	timeForBlockOut time.Duration
}

func NewChainClient(rpcAddr, secret string, t time.Duration) (Chainer, error) {
	var (
		err error
		cli = &chainClient{}
	)
	cli.c, err = gsrpc.NewSubstrateAPI(rpcAddr)
	if err != nil {
		return nil, err
	}
	cli.metadata, err = cli.c.RPC.State.GetMetadataLatest()
	if err != nil {
		return nil, err
	}
	cli.genesisHash, err = cli.c.RPC.Chain.GetBlockHash(0)
	if err != nil {
		return nil, err
	}
	cli.runtimeVersion, err = cli.c.RPC.State.GetRuntimeVersionLatest()
	if err != nil {
		return nil, err
	}
	cli.keyEvents, err = types.CreateStorageKey(
		cli.metadata,
		state_System,
		system_Events,
		nil,
	)
	if err != nil {
		return nil, err
	}
	if secret != "" {
		cli.keyring, err = signature.KeyringPairFromSecret(secret, 0)
		if err != nil {
			return nil, err
		}
	}
	cli.l = new(sync.Mutex)
	cli.timeForBlockOut = t
	cli.rpcAddr = rpcAddr
	return cli, nil
}

func (c *chainClient) IsChainClientOk() bool {
	err := healthchek(c.c)
	if err != nil {
		c.c = nil
		cli, err := reconnectChainClient(c.rpcAddr)
		if err != nil {
			return false
		}
		c.c = cli
		return true
	}
	return true
}

func reconnectChainClient(rpcAddr string) (*gsrpc.SubstrateAPI, error) {
	return gsrpc.NewSubstrateAPI(rpcAddr)
}

func healthchek(a *gsrpc.SubstrateAPI) error {
	defer recover()
	_, err := a.RPC.System.Health()
	return err
}
