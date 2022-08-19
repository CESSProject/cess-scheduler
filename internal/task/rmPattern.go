package task

import (
	. "cess-scheduler/internal/logger"
	"cess-scheduler/internal/pattern"
	"cess-scheduler/tools"
	"time"
)

func task_ClearAuthMap(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()
	var count uint8
	for {
		count++
		if count >= 5 {
			count = 0
			Com.Info("Connected miners:")
			Com.Sugar().Info(pattern.GetConnectedSpacem())
			Com.Info("Black miners:")
			Com.Sugar().Info(pattern.GetBlacklist())
		}
		time.Sleep(time.Minute)
		pattern.DeleteExpiredAuth()
		pattern.DeleteExpiredSpacem()
	}
}
