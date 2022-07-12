package chain

import (
	"cess-scheduler/configs"
	. "cess-scheduler/internal/logger"
	"cess-scheduler/tools"
	"fmt"
	"os"
	"sync"
	"time"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
)

type SubstrateApi struct {
	l *sync.Mutex
	r *gsrpc.SubstrateAPI
}

var SubApi *SubstrateApi

func ChainInit() {
	var err error
	SubApi, err = newSubApi(configs.C.RpcAddr)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
	go SubApi.keepAlive()
}

func SyncState() (bool, error) {
	h, err := SubApi.r.RPC.System.Health()
	if err != nil {
		return false, err
	}
	return h.IsSyncing, nil
}

func NewRpcClient(rpcaddr string) (*gsrpc.SubstrateAPI, error) {
	return gsrpc.NewSubstrateAPI(rpcaddr)
}

func newSubApi(rpcaddr string) (*SubstrateApi, error) {
	var err error
	if SubApi == nil {
		SubApi = new(SubstrateApi)
		SubApi.l = new(sync.Mutex)
		SubApi.r, err = gsrpc.NewSubstrateAPI(rpcaddr)
		return SubApi, err
	}
	if SubApi.l == nil {
		SubApi.l = new(sync.Mutex)
	}
	if SubApi.r == nil {
		SubApi.r, err = gsrpc.NewSubstrateAPI(rpcaddr)
		return SubApi, err
	}
	return SubApi, nil
}

func (this *SubstrateApi) keepAlive() {
	var (
		err     error
		count_r uint8  = 0
		peer    uint64 = 0
	)

	for range time.Tick(time.Second * 25) {
		if count_r <= 1 {
			this.l.Lock()
			peer, err = healthchek(this.r)
			if err != nil || peer == 0 {
				Com.Sugar().Errorf("[%v] %v", peer, err)
				count_r++
			}
			this.l.Unlock()
		}
		if count_r > 1 {
			count_r = 2
			this.l.Lock()
			this.r, err = gsrpc.NewSubstrateAPI(configs.C.RpcAddr)
			if err != nil {
				Com.Sugar().Errorf("%v", err)
			} else {
				count_r = 0
			}
			this.l.Unlock()
		}
	}
}

func (this *SubstrateApi) getApi() *gsrpc.SubstrateAPI {
	this.l.Lock()
	return this.r
}
func (this *SubstrateApi) free() {
	this.l.Unlock()
}

func healthchek(a *gsrpc.SubstrateAPI) (uint64, error) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()
	h, err := a.RPC.System.Health()
	return uint64(h.Peers), err
}
