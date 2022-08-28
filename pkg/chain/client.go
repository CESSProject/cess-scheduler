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
	GetPublicKey() []byte
	GetStorageMinerInfo(pkey []byte) (MinerInfo, error)
	GetAllStorageMiner() ([]types.AccountID, error)
	GetFileMetaInfo(fid types.Bytes) (FileMetaInfo, error)
	GetSchedulerInfo() ([]SchedulerInfo, error)
	GetProofs() ([]Proof, error)
	GetCessAccount() (string, error)
	GetAccountInfo() (types.AccountInfo, error)
	GetSpacePackageInfo() (SpacePackage, error)
	Register(stash, contact string) (string, error)
	SubmitProofResults(data []ProofResult) (string, error)
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
		State_System,
		System_Account,
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
	return cli, nil
}

func ReconnectChainClient(rpcAddr string, t time.Duration, keyring signature.KeyringPair) (*chainClient, error) {
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
	cli.keyEvents, err = types.CreateStorageKey(cli.metadata, "System", "Events", nil)
	if err != nil {
		return nil, err
	}

	cli.keyring = keyring
	cli.l = new(sync.Mutex)
	cli.timeForBlockOut = t
	return cli, nil
}

func (c *chainClient) IsChainClientOk() bool {
	id, err := healthchek(c.c)
	if id == 0 || err != nil {
		c, err = ReconnectChainClient(c.rpcAddr, c.timeForBlockOut, c.keyring)
		if err != nil {
			return false
		}
		return true
	}
	return true
}

func healthchek(a *gsrpc.SubstrateAPI) (uint64, error) {
	defer recover()
	h, err := a.RPC.System.Health()
	return uint64(h.Peers), err
}

// func SyncState() (bool, error) {
// 	_, err := GetRpcClient_Safe(configs.C.RpcAddr)
// 	defer Free()
// 	if err != nil {
// 		return false, err
// 	}
// 	h, err := api.RPC.System.Health()
// 	if err != nil {
// 		return false, err
// 	}
// 	return h.IsSyncing, nil
// }
