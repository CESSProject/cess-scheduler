package task

import (
	"cess-scheduler/internal/chain"
	"cess-scheduler/internal/db"
	. "cess-scheduler/internal/logger"
	"cess-scheduler/internal/pattern"
	"cess-scheduler/tools"
	"encoding/json"
	"time"
)

func task_SyncMinersInfo(ch chan bool) {
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
		ch <- true
	}()

	Tsmi.Info("-----> Start task_UpdateMinerInfo")

	for {
		allMinerAcc, _ := chain.GetAllMinerDataOnChain()
		if len(allMinerAcc) == 0 {
			time.Sleep(time.Second * 3)
			continue
		}
		for i := 0; i < len(allMinerAcc); i++ {
			b := allMinerAcc[i][:]
			addr, err := tools.EncodeToCESSAddr(b)
			if err != nil {
				Tsmi.Sugar().Errorf("[%v] EncodeToCESSAddr: %v", allMinerAcc[i], err)
				continue
			}
			ok, err := db.Has(b)
			if err != nil {
				Tsmi.Sugar().Errorf("[%v] c.Has: %v", addr, err)
				continue
			}

			if ok {
				continue
			}

			var cm chain.Cache_MinerInfo

			mdata, err := chain.GetMinerInfo(allMinerAcc[i])
			if err != nil {
				Tsmi.Sugar().Errorf("[%v] GetMinerInfo: %v", addr, err)
				continue
			}
			if string(mdata.State) == "exit" {
				continue
			}
			cm.Peerid = uint64(mdata.PeerId)
			cm.Ip = string(mdata.Ip)
			cm.Pubkey = b

			value, err := json.Marshal(&cm)
			if err != nil {
				Tsmi.Sugar().Errorf("[%v] json.Marshal: %v", addr, err)
				continue
			}
			err = db.Put(b, value)
			if err != nil {
				Tsmi.Sugar().Errorf("[%v] c.Put: %v", addr, err)
			}
			Tsmi.Sugar().Infof("[%v] Cache succeeded", addr)
			pattern.DeleteBliacklist(string(b))
			Com.Sugar().Infof("Del blacklist: %v", b)
		}
		time.Sleep(time.Minute * 2)
	}
}
