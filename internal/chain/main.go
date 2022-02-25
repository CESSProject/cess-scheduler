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
	wlock *sync.Mutex
	r     *gsrpc.SubstrateAPI
}

type MinersInfo struct {
	f    chan bool
	Data []CessChain_AllMinerItems
}

var Ci *CessInfo
var Tk *CessInfo
var SP *CessInfo
var api = new(mySubstrateApi)
var Miners MinersInfo

func Chain_Init() {
	var err error
	Ci = &CessInfo{
		RpcAddr:                configs.Confile.CessChain.RpcAddr,
		IdentifyAccountPhrase:  configs.Confile.MinerData.IdAccountPhraseOrSeed,
		IncomeAccountPublicKey: "",
		TransactionName:        configs.ChainTransaction_Sminer_Register,
		ChainModule:            configs.ChainModule_EtcdSminer,
		ChainModuleMethod:      configs.ChainModule_Sminer_EtcdRegister,
	}
	Tk = &CessInfo{
		RpcAddr:                configs.Confile.CessChain.RpcAddr,
		IdentifyAccountPhrase:  configs.Confile.MinerData.IdAccountPhraseOrSeed,
		IncomeAccountPublicKey: "",
		TransactionName:        configs.ClusterToken_Sminer_Register,
		ChainModule:            configs.ClusterToken_Sminer,
		ChainModuleMethod:      configs.ClusterToken_Sminer_MinerItems,
	}
	SP = &CessInfo{
		RpcAddr:                configs.Confile.CessChain.RpcAddr,
		IdentifyAccountPhrase:  configs.Confile.MinerData.IdAccountPhraseOrSeed,
		IncomeAccountPublicKey: "",
		TransactionName:        configs.ServicePort_Sminer_Register,
		ChainModule:            configs.ServicePort_Sminer,
		ChainModuleMethod:      configs.ServicePort_Sminer_MinerItems,
	}
	api.wlock = new(sync.Mutex)
	api.r, err = gsrpc.NewSubstrateAPI(configs.Confile.CessChain.RpcAddr)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		logger.ErrLogger.Sugar().Errorf("%v", err)
		os.Exit(configs.Exit_Normal)
	}
	go substrateAPIKeepAlive()

	Miners.f = make(chan bool, 1)
	Miners.Data, err = GetAllMinerDataOnChain(
		configs.ChainModule_Sminer,
		configs.ChainModule_Sminer_AllMinerItems,
	)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m Get Miner Info Failed,%v\n", 41, err)
		logger.ErrLogger.Sugar().Errorf("%v", err)
		os.Exit(configs.Exit_Normal)
	}
	go getMinersInfo_wait()
	err = SP.RegisterEtcdOnChain(configs.Confile.MinerData.ServicePort)
	if err != nil {
		logger.ErrLogger.Sugar().Errorf("This account have not privilege to set service port on chain:%v", err.Error())
	}
	res, err := SP.GetDataOnChain()
	if len(res.Ip) != 0 {
		configs.Confile.MinerData.ServicePort = string(res.Ip)
	} else {
		logger.ErrLogger.Sugar().Errorf("lack of very important param,error:%v", err.Error())
		panic("No port information on chain or internet issue happened!")
	}
}

func getMinersInfo_wait() {
	for {
		Miners.f <- true
		time.Sleep(time.Minute * time.Duration(5))
	}
}

func GetMinersDate() []CessChain_AllMinerItems {
	var err error
	if len(Miners.f) == 1 {
		<-Miners.f
		Miners.Data, err = GetAllMinerDataOnChain(
			configs.ChainModule_Sminer,
			configs.ChainModule_Sminer_AllMinerItems,
		)
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("Get Miner Info Failed %v", err)
		}
	}
	return Miners.Data
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

func getSubstrateApi_safe() *gsrpc.SubstrateAPI {
	api.wlock.Lock()
	return api.r
}
func releaseSubstrateApi() {
	api.wlock.Unlock()
}
