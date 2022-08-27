package chain

import (
	"sync"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

type Chainer interface {
	GetStorageMinerInfo() (MinerInfo, error)
	GetAllMinerDataOnChain() ([]types.AccountID, error)
	GetFileMetaInfo(fid types.Bytes) (FileMetaInfo, error)
	GetSchedulerInfo() ([]SchedulerInfo, error)
	GetProofs() ([]Proof, error)
	GetCessAccount() (types.Bytes, error)
	GetSpacePackageInfo() (SpacePackage, error)
}

type chainClient struct {
	l              *sync.Mutex
	c              *gsrpc.SubstrateAPI
	metadata       *types.Metadata
	keyEvents      types.StorageKey
	runtimeVersion *types.RuntimeVersion
	genesisHash    types.Hash
	keyring        signature.KeyringPair
	rpcAddr        string
}

func NewChainClient(rpcAddr, secret string) (Chainer, error) {
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
	if secret != "" {
		cli.keyring, err = signature.KeyringPairFromSecret(secret, 0)
		if err != nil {
			return nil, err
		}
	}
	cli.l = new(sync.Mutex)
	return cli, nil
}

func ReconnectChainClient(rpcAddr string, keyring signature.KeyringPair) (*chainClient, error) {
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
	return cli, nil
}

func (c *chainClient) IsChainClientOk() bool {
	id, err := healthchek(c.c)
	if id == 0 || err != nil {
		c, err = ReconnectChainClient(c.rpcAddr, c.keyring)
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
