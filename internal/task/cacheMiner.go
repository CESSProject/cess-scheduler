package task

import (
	"encoding/json"
	"time"

	"github.com/CESSProject/cess-scheduler/internal/pattern"
	"github.com/CESSProject/cess-scheduler/pkg/chain"
	"github.com/CESSProject/cess-scheduler/pkg/db"
	"github.com/CESSProject/cess-scheduler/pkg/rpc"
	"github.com/CESSProject/cess-scheduler/tools"
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

			var cm chain.Cache_MinerInfo

			mdata, err := chain.GetMinerInfo(allMinerAcc[i])
			if err != nil {
				Tsmi.Sugar().Errorf("[%v] GetMinerInfo: %v", addr, err)
				continue
			}

			if ok {
				err = rpc.Dial(string(mdata.Ip))
				if err != nil {
					Tsmi.Sugar().Errorf("[%v] %v", addr, err)
					db.Delete(b)
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
					Tsmi.Sugar().Errorf("[%v] Put: %v", addr, err)
				}
				Tsmi.Sugar().Infof("[%v] Cache updated", addr)
				continue
			}

			if string(mdata.State) == "exit" {
				continue
			}

			err = rpc.Dial(string(mdata.Ip))
			if err != nil {
				Tsmi.Sugar().Errorf("[%v] %v", addr, err)
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
				Tsmi.Sugar().Errorf("[%v] Put: %v", addr, err)
			}
			Tsmi.Sugar().Infof("[%v] Cache is stored", addr)
			pattern.DeleteBliacklist(string(b))
		}
	}
}
