package task

import (
	"cess-scheduler/configs"
	. "cess-scheduler/internal/logger"
	"cess-scheduler/internal/pattern"
	apiv1 "cess-scheduler/internal/proof/apiv1"
	"cess-scheduler/tools"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func task_GenerateFiller(ch chan bool) {
	defer func() {
		ch <- true
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
	}()

	Tgf.Info("-----> Start task_GenerateFiller")

	var (
		err        error
		uid        string
		fillerpath string
	)
	for {
		for len(pattern.C_Filler) < pattern.C_Filler_Maxlen {
			for {
				uid, _ = tools.GetGuid(int64(tools.RandomInRange(0, 1024)))
				if uid == "" {
					continue
				}
				fillerpath = filepath.Join(configs.SpaceCacheDir, fmt.Sprintf("%s", uid))
				_, err = os.Stat(fillerpath)
				if err != nil {
					break
				}
			}
			err = generateFiller(fillerpath)
			if err != nil {
				Tgf.Sugar().Errorf("%v", err)
				os.Remove(fillerpath)
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 30)))
				continue
			}

			fstat, _ := os.Stat(fillerpath)
			if fstat.Size() != 8386771 {
				Tgf.Sugar().Errorf("filler size err: %v", err)
				os.Remove(fillerpath)
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 30)))
				continue
			}

			// calculate file tag info
			var PoDR2commit apiv1.PoDR2Commit
			var commitResponse apiv1.PoDR2CommitResponse
			PoDR2commit.FilePath = fillerpath
			PoDR2commit.BlockSize = configs.BlockSize

			commitResponseCh, err := PoDR2commit.PoDR2ProofCommit(
				apiv1.Key_Ssk,
				string(apiv1.Key_SharedParams),
				int64(configs.ScanBlockSize),
			)
			if err != nil {
				os.Remove(fillerpath)
				Tgf.Sugar().Errorf("PoDR2ProofCommit false")
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 30)))
				continue
			}

			select {
			case commitResponse = <-commitResponseCh:
			}

			if commitResponse.StatueMsg.StatusCode != apiv1.Success {
				os.Remove(fillerpath)
				Tgf.Sugar().Errorf("PoDR2ProofCommit false")
				time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 30)))
				continue
			}

			var fillerEle pattern.Filler
			fillerEle.FillerId = uid
			fillerEle.Path = fillerpath
			fillerEle.T = commitResponse.T
			fillerEle.Sigmas = commitResponse.Sigmas
			pattern.Chan_Filler <- fillerEle
			Tgf.Sugar().Infof("Produced a filler: %v", uid)
		}
		time.Sleep(time.Second)
	}
}

func generateFiller(fpath string) error {
	f, err := os.OpenFile(fpath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	for i := 0; i < 2048; i++ {
		f.WriteString(tools.RandStr(4095) + "\n")
	}
	err = f.Sync()
	if err != nil {
		os.Remove(fpath)
		return err
	}
	return nil
}