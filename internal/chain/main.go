package chain

import (
	"fmt"
	"os"
	"scheduler-mining/configs"
	"scheduler-mining/internal/logger"
	"sync"
	"time"

	gsrpc "github.com/centrifuge/go-substrate-rpc-client/v4"
)

type mySubstrateApi struct {
	wlock sync.Mutex
	c     chan bool
	r     *gsrpc.SubstrateAPI
}

var api mySubstrateApi

func Chain_Init() {
	var err error
	api.r, err = gsrpc.NewSubstrateAPI(configs.Confile.CessChain.RpcAddr)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		logger.ErrLogger.Sugar().Errorf("%v", err)
		os.Exit(configs.Exit_Normal)
	}
	api.c = make(chan bool, 1)
	api.c <- true
	go waitBlock(api.c)
	go substrateAPIKeepAlive()
}

func substrateAPIKeepAlive() {
	var (
		err     error
		count_r uint8  = 0
		peer    uint64 = 0
	)

	for range time.Tick(time.Second * 25) {
		if count_r <= 1 {
			peer, err = healthchek(api.r)
			if err != nil || peer == 0 {
				count_r++
			}
		}
		if count_r > 1 {
			count_r = 2
			api.r, err = gsrpc.NewSubstrateAPI(configs.Confile.CessChain.RpcAddr)
			if err != nil {
				logger.ErrLogger.Sugar().Errorf("%v", err)
			} else {
				count_r = 0
			}
		}
	}
}

func healthchek(a *gsrpc.SubstrateAPI) (uint64, error) {
	defer func() {
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	h, err := a.RPC.System.Health()
	return uint64(h.Peers), err
}

func waitBlock(ch chan bool) {
	for {
		ch <- true
		time.Sleep(time.Second * 8)
	}
}

func SubstrateAPI_Read() *gsrpc.SubstrateAPI {
	return api.r
}

func SubstrateAPI_Write() *gsrpc.SubstrateAPI {
	api.wlock.Lock()
	defer api.wlock.Unlock()
	for len(api.c) == 0 {
		time.Sleep(time.Millisecond * 200)
	}
	<-api.c
	return api.r
}
