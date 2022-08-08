package chain

import (
	"cess-scheduler/configs"
	"log"
	"os"
	"reflect"
	"sync"
	"time"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

var (
	first          bool = true
	l              *sync.Mutex
	api            *gsrpc.SubstrateAPI
	metadata       *types.Metadata
	keyEvents      types.StorageKey
	runtimeVersion *types.RuntimeVersion
	genesisHash    types.Hash
	keyring        signature.KeyringPair
)

func ChainInit() {
	l = new(sync.Mutex)
	for {
		ok, err := SyncState()
		if err != nil {
			log.Printf("\x1b[%dm[err]\x1b[0m Network Error: %v\n", 41, err)
			os.Exit(1)
		}
		if !ok {
			break
		}
		log.Printf("\x1b[%dm[ok]\x1b[0m In sync block...\n", 42)
		time.Sleep(time.Second * 30)
	}
	log.Printf("\x1b[%dm[ok]\x1b[0m Sync complete\n", 42)
	GetMetadata(api)
	GetGenesisHash(api)
	GetRuntimeVersion(api)
	GetKeyEvents()
}

func SyncState() (bool, error) {
	_, err := GetRpcClient_Safe(configs.C.RpcAddr)
	defer Free()
	if err != nil {
		return false, err
	}
	h, err := api.RPC.System.Health()
	if err != nil {
		return false, err
	}
	return h.IsSyncing, nil
}

func GetRpcClient_Safe(rpcaddr string) (*gsrpc.SubstrateAPI, error) {
	var err error
	l.Lock()
	if api == nil {
		api, err = gsrpc.NewSubstrateAPI(rpcaddr)
		return api, err
	}
	id, err := healthchek(api)
	if id == 0 || err != nil {
		api, err = gsrpc.NewSubstrateAPI(rpcaddr)
		if err != nil {
			return nil, err
		}
	}
	return api, nil
}

func Free() {
	l.Unlock()
}

func healthchek(a *gsrpc.SubstrateAPI) (uint64, error) {
	defer func() {
		recover()
	}()
	h, err := a.RPC.System.Health()
	return uint64(h.Peers), err
}

func NewRpcClient(rpcaddr string) (*gsrpc.SubstrateAPI, error) {
	return gsrpc.NewSubstrateAPI(rpcaddr)
}

func GetMetadata(api *gsrpc.SubstrateAPI) (*types.Metadata, error) {
	var err error
	if metadata == nil {
		metadata, err = api.RPC.State.GetMetadataLatest()
		return metadata, err
	}
	return metadata, nil
}

func GetGenesisHash(api *gsrpc.SubstrateAPI) (types.Hash, error) {
	var err error
	if first {
		genesisHash, err = api.RPC.Chain.GetBlockHash(0)
		if err == nil {
			first = false
		}
		return genesisHash, err
	}
	return genesisHash, nil
}

func GetRuntimeVersion(api *gsrpc.SubstrateAPI) (*types.RuntimeVersion, error) {
	var err error
	if runtimeVersion == nil {
		runtimeVersion, err = api.RPC.State.GetRuntimeVersionLatest()
		return runtimeVersion, err
	}
	return runtimeVersion, nil
}

func GetKeyEvents() (types.StorageKey, error) {
	var err error
	if len(keyEvents) == 0 {
		keyEvents, err = types.CreateStorageKey(metadata, "System", "Events", nil)
		return keyEvents, err
	}
	return keyEvents, nil
}

func GetPublicKey(privatekey string) ([]byte, error) {
	kring, err := signature.KeyringPairFromSecret(privatekey, 0)
	if err != nil {
		return nil, err
	}
	return kring.PublicKey, nil
}

func GetKeyring() (signature.KeyringPair, error) {
	var err error
	if reflect.DeepEqual(keyring, signature.KeyringPair{}) {
		keyring, err = signature.KeyringPairFromSecret(configs.C.CtrlPrk, 0)
		return keyring, err
	}
	return keyring, nil
}

// func newSubApi(rpcaddr string) (*SubstrateApi, error) {
// 	var err error
// 	if SubApi == nil {
// 		SubApi = new(SubstrateApi)
// 		SubApi.l = new(sync.Mutex)
// 		SubApi.r, err = gsrpc.NewSubstrateAPI(rpcaddr)
// 		return SubApi, err
// 	}
// 	if SubApi.l == nil {
// 		SubApi.l = new(sync.Mutex)
// 	}
// 	if SubApi.r == nil {
// 		SubApi.r, err = gsrpc.NewSubstrateAPI(rpcaddr)
// 		return SubApi, err
// 	}
// 	return SubApi, nil
// }

// func (this *SubstrateApi) keepAlive() {
// 	var (
// 		err     error
// 		count_r uint8  = 0
// 		peer    uint64 = 0
// 	)

// 	for range time.Tick(time.Second * 25) {
// 		if count_r <= 1 {
// 			this.l.Lock()
// 			peer, err = healthchek(this.r)
// 			this.l.Unlock()
// 			if err != nil || peer == 0 {
// 				Com.Sugar().Errorf("[%v] %v", peer, err)
// 				count_r++
// 			}
// 		}
// 		if count_r > 1 {
// 			count_r = 2
// 			this.l.Lock()
// 			this.r, err = gsrpc.NewSubstrateAPI(configs.C.RpcAddr)
// 			this.l.Unlock()
// 			if err != nil {
// 				Com.Sugar().Errorf("%v", err)
// 				time.Sleep(time.Minute * time.Duration(tools.RandomInRange(2, 5)))
// 			} else {
// 				count_r = 0
// 			}
// 		}
// 	}
// }

// func (this *SubstrateApi) getApi() *gsrpc.SubstrateAPI {
// 	this.l.Lock()
// 	return this.r
// }
// func (this *SubstrateApi) free() {
// 	this.l.Unlock()
// }

// func healthchek(a *gsrpc.SubstrateAPI) (uint64, error) {
// 	defer func() {
// 		if err := recover(); err != nil {
// 			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
// 		}
// 	}()
// 	h, err := a.RPC.System.Health()
// 	return uint64(h.Peers), err
// }
