package task

import (
	"cess-scheduler/configs"
	"cess-scheduler/internal/chain"
	"cess-scheduler/internal/db"
	. "cess-scheduler/internal/logger"
	apiv1 "cess-scheduler/internal/proof/apiv1"
	"cess-scheduler/internal/rpc"
	"cess-scheduler/tools"
	"encoding/json"
	"fmt"
	"os"
	"time"

	. "cess-scheduler/internal/rpc/protobuf"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"google.golang.org/protobuf/proto"
)

//
func task_ValidateProof(ch chan bool) {
	var (
		err         error
		goeson      bool
		puk         chain.Chain_SchedulerPuk
		poDR2verify apiv1.PoDR2Verify
		reqtag      ReadTagReq
		proofs      = make([]chain.Chain_Proofs, 0)
	)
	defer func() {
		if err := recover(); err != nil {
			Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
		}
		ch <- true
	}()

	Tvp.Info("--> Start task_ValidateProof")

	reqtag.Acc, err = chain.GetPublicKeyByPrk(configs.C.CtrlPrk)
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}

	Tvp.Sugar().Infof("--> %v", reqtag.Acc)

	for {
		puk, err = chain.GetSchedulerPukFromChain()
		if err != nil {
			time.Sleep(time.Second * time.Duration(tools.RandomInRange(5, 30)))
			continue
		}
		Tvp.Info("--> Successfully found puk")
		Tvp.Sugar().Infof("--> %v", puk.Shared_g)
		Tvp.Sugar().Infof("--> %v", puk.Shared_params)
		Tvp.Sugar().Infof("--> %v", puk.Spk)
		break
	}

	for {
		var verifyResults = make([]chain.VerifyResult, 0)
		proofs, err = chain.GetProofsFromChain(configs.C.CtrlPrk)
		if err != nil {
			if err.Error() != chain.ERR_Empty {
				Tvp.Sugar().Errorf("%v", err)
			}
			time.Sleep(time.Minute * time.Duration(tools.RandomInRange(3, 10)))
			continue
		}
		if len(proofs) == 0 {
			time.Sleep(time.Minute * time.Duration(tools.RandomInRange(3, 10)))
			continue
		}

		Tvp.Sugar().Infof("--> Ready to verify %v proofs", len(proofs))

		var respData []byte
		var tag apiv1.TagInfo
		for i := 0; i < len(proofs); i++ {
			if len(verifyResults) > 45 {
				break
			}
			goeson = false
			addr, err := tools.EncodeToCESSAddr(proofs[i].Miner_pubkey[:])
			if err != nil {
				Tvp.Sugar().Errorf("%v EncodeToCESSAddr: %v", proofs[i].Miner_pubkey, err)
			}

			cacheData, err := db.Get(proofs[i].Miner_pubkey[:])
			if err != nil {
				resultTemp := chain.VerifyResult{}
				resultTemp.Miner_pubkey = proofs[i].Miner_pubkey
				resultTemp.FileId = proofs[i].Challenge_info.File_id
				if err.Error() == "leveldb: not found" {
					resultTemp.Result = false
				} else {
					resultTemp.Result = true
				}
				verifyResults = append(verifyResults, resultTemp)
				continue
			}

			var minerinfo chain.Cache_MinerInfo
			err = json.Unmarshal(cacheData, &minerinfo)
			if err != nil {
				Tvp.Sugar().Errorf("[%v] Unmarshal: %v", addr, err)
				resultTemp := chain.VerifyResult{}
				resultTemp.Miner_pubkey = proofs[i].Miner_pubkey
				resultTemp.FileId = proofs[i].Challenge_info.File_id
				resultTemp.Result = true
				verifyResults = append(verifyResults, resultTemp)
				continue
			}

			goeson = false
			reqtag.FileId = string(proofs[i].Challenge_info.File_id)
			req_proto, err := proto.Marshal(&reqtag)
			if err != nil {
				Tvp.Sugar().Errorf("[%v] Marshal: %v", addr, err)
			}
			for j := 0; j < 3; j++ {
				respData, err = rpc.WriteData(
					string(minerinfo.Ip),
					rpc.RpcService_Miner,
					rpc.RpcMethod_Miner_ReadFileTag,
					time.Duration(time.Second*30),
					req_proto,
				)
				if err != nil {
					Tvp.Sugar().Errorf("[%v] WriteData: %v", addr, err)
					time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 6)))
				} else {
					goeson = true
					break
				}
			}

			if !goeson {
				resultTemp := chain.VerifyResult{}
				resultTemp.Miner_pubkey = proofs[i].Miner_pubkey
				resultTemp.FileId = proofs[i].Challenge_info.File_id
				resultTemp.Result = false
				verifyResults = append(verifyResults, resultTemp)
				continue
			}

			err = json.Unmarshal(respData, &tag)
			if err != nil {
				Tvp.Sugar().Errorf("[%v] Unmarshal: %v", addr, err)
			}
			qSlice, err := apiv1.PoDR2ChallengeGenerateFromChain(proofs[i].Challenge_info.Block_list, proofs[i].Challenge_info.Random)
			if err != nil {
				Tvp.Sugar().Errorf("[%v] [%v] [%v] qslice: %v", addr, len(proofs[i].Challenge_info.Block_list), len(proofs[i].Challenge_info.Random), err)
			}

			poDR2verify.QSlice = qSlice
			poDR2verify.MU = make([][]byte, len(proofs[i].Mu))
			for j := 0; j < len(proofs[i].Mu); j++ {
				poDR2verify.MU[j] = append(poDR2verify.MU[j], proofs[i].Mu[j]...)
			}

			poDR2verify.Sigma = proofs[i].Sigma
			poDR2verify.T = tag.T

			gWait := make(chan bool)
			go func(ch chan bool) {
				defer func() {
					if err := recover(); err != nil {
						ch <- true
						Pnc.Sugar().Errorf("%v", tools.RecoverError(err))
					}
				}()
				ch <- poDR2verify.PoDR2ProofVerify(puk.Shared_g, puk.Spk, string(puk.Shared_params))
			}(gWait)
			result := <-gWait
			resultTemp := chain.VerifyResult{}
			resultTemp.Miner_pubkey = proofs[i].Miner_pubkey
			resultTemp.FileId = proofs[i].Challenge_info.File_id
			resultTemp.Result = types.Bool(result)
			verifyResults = append(verifyResults, resultTemp)
		}
		go processProofResult(verifyResults)
	}
}

func processProofResult(data []chain.VerifyResult) {
	var (
		err      error
		txhash   string
		tryCount uint8
	)
	for tryCount < 3 {
		txhash, err = chain.PutProofResult(configs.C.CtrlPrk, data)
		if txhash != "" {
			Tvp.Sugar().Infof("Proof result submitted: %v", txhash)
			if err == nil {
				return
			}
		}
		tryCount++
		time.Sleep(time.Second * time.Duration(tools.RandomInRange(3, 15)))
	}
}
