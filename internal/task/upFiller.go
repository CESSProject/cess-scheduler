package task

import (
	"cess-scheduler/configs"
	"cess-scheduler/internal/chain"
	. "cess-scheduler/internal/logger"
	"cess-scheduler/internal/pattern"
	"cess-scheduler/tools"
	"os"
	"path/filepath"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
)

func task_SubmitFillerMeta(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	Tsfm.Info("-----> Start task_SubmitFillerMeta")

	var (
		err    error
		txhash string
	)
	t_active := time.Now()
	for {
		time.Sleep(time.Second * 1)
		for len(pattern.Chan_FillerMeta) > 0 {
			var tmp = <-pattern.Chan_FillerMeta
			pattern.FillerMap.Add(string(tmp.Acc[:]), tmp)
		}
		if time.Since(t_active).Seconds() > 5 {
			t_active = time.Now()
			for k, v := range pattern.FillerMap.Fillermetas {
				addr, _ := tools.EncodeToCESSAddr([]byte(k))
				if len(v) >= 8 {
					txhash, err = chain.PutSpaceTagInfoToChain(configs.C.CtrlPrk, types.NewAccountID([]byte(k)), v[:8])
					if txhash == "" {
						Tsfm.Sugar().Errorf("%v", err)
						continue
					}
					pattern.FillerMap.Delete(k)
					pattern.DeleteSpacemap(k)
					fpath := filepath.Join(configs.SpaceCacheDir, addr)
					os.RemoveAll(fpath)
					Tsfm.Sugar().Infof("[%v] %v", addr, txhash)
				} else {
					ok := pattern.IsExitSpacem(k)
					if !ok && len(v) > 0 {
						txhash, err = chain.PutSpaceTagInfoToChain(configs.C.CtrlPrk, types.NewAccountID([]byte(k)), v[:])
						if txhash == "" {
							Tsfm.Sugar().Errorf("%v", err)
							continue
						}
						pattern.FillerMap.Delete(k)
						fpath := filepath.Join(configs.SpaceCacheDir, addr)
						os.RemoveAll(fpath)
						Tsfm.Sugar().Infof("[%v] %v", addr, txhash)
					}
				}
			}
		}
	}
}
