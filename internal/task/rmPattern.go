package task

import (
	"cess-scheduler/configs"
	"cess-scheduler/internal/chain"
	. "cess-scheduler/internal/logger"
	"cess-scheduler/internal/pattern"
	"cess-scheduler/internal/rpc"
	"cess-scheduler/tools"
	"log"
	"math/big"
	"os"
	"syscall"
	"time"
)

func task_ClearAuthMap(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()
	Com.Info("-----> Start task_ClearAuthMap")
	var count uint8
	for {
		count++
		time.Sleep(time.Minute)

		if count%5 == 0 {
			accountinfo, err := chain.GetAccountInfo(configs.PublicKey)
			if err == nil {
				if accountinfo.Data.Free.CmpAbs(new(big.Int).SetUint64(2000000000000)) == -1 {
					Com.Info("Insufficient balance, program exited.")
					log.Printf("Insufficient balance, program exited.\n")
					os.Exit(1)
				}
			}
			Com.Info("Connected miners:")
			Com.Sugar().Info(pattern.GetConnectedSpacem())
			Com.Info("Black miners:")
			Com.Sugar().Info(pattern.GetBlacklist())
		}

		if count%60 == 0 {
			count = 0
			files, _ := tools.GetAllFile(configs.SpaceCacheDir)
			if len(files) > 0 {
				for _, v := range files {
					fst, err := os.Stat(v)
					if err != nil {
						linuxFileAttr := fst.Sys().(*syscall.Stat_t)
						if time.Since(time.Unix(linuxFileAttr.Ctim.Sec, 0)).Hours() > 5 {
							os.Remove(v)
						}
					}
				}
			}
		}
		pattern.DeleteExpiredAuth()
		pattern.DeleteExpiredSpacem()
		rpc.DelExpired()
	}
}
