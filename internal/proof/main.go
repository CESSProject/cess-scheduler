package proof

import (
	"fmt"
	"scheduler-mining/configs"
	"scheduler-mining/internal/chain"
	"scheduler-mining/internal/logger"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v3/types"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/actors/runtime/proof"
)

func Chain_Main() {
	go verifyVpa()
	go verifyVpb()
	go verifyVpc()
	go verifyVpd()
}

func verifyVpa() {
	var (
		err     error
		i       int
		j       int
		ok      bool
		segtype uint8
		sealcid string
		proofs  string
	)
	for range time.Tick(time.Second) {
		var data []chain.UnVerifiedVpaVpb
		data, err = chain.GetUnverifiedVpaVpb(
			configs.ChainModule_SegmentBook,
			configs.ChainModule_SegmentBook_UnVerifiedA,
		)
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("%v", err)
			continue
		}
		//fmt.Println("Number of unverified vpa: ", len(data))

		if len(data) == 0 {
			time.Sleep(time.Minute)
		} else {
			logger.InfoLogger.Sugar().Infof("Number of unverified vpa: %v", len(data))
		}
		for i = 0; i < len(data); i++ {
			if len(data[i].Proof) == 0 {
				continue
			}
			proofs = ""
			sealcid = ""
			for j = 0; j < len(data[i].Proof); j++ {
				temp := fmt.Sprintf("%c", data[i].Proof[j])
				proofs += temp
			}
			p, err := types.HexDecodeString(proofs)
			if err != nil {
				logger.ErrLogger.Sugar().Errorf("%v", err)
			}
			//fmt.Println(p)
			for j = 0; j < len(data[i].Sealed_cid); j++ {
				temp := fmt.Sprintf("%c", data[i].Sealed_cid[j])
				sealcid += temp
			}
			//fmt.Println(sealcid)
			sizetypes := fmt.Sprintf("%v", data[i].Size_type)
			switch sizetypes {
			case "8":
				segtype = 1
			case "512":
				segtype = 2
			}
			if segtype == 0 {
				//fmt.Printf("\x1b[%dm[err]\x1b[0m segtype is invalid\n", 41)
				logger.ErrLogger.Sugar().Errorf("[C%v] segtype is invalid", data[i].Peer_id)
				continue
			}
			fmt.Println("Verify vpa :", uint64(data[i].Peer_id), uint64(data[i].Segment_id), uint32(data[i].Rand), segtype, sealcid, p)
			ok, err = verifyVpaProof(uint64(data[i].Peer_id), uint64(data[i].Segment_id), uint32(data[i].Rand), segtype, sealcid, p)
			if err != nil {
				//fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
				logger.ErrLogger.Sugar().Errorf("[C%v] %v", data[i].Peer_id, err)
				continue
			}

			err = chain.VerifyInVpaOrVpb(
				configs.Confile.SchedulerInfo.TransactionPrK,
				configs.ChainTx_SegmentBook_VerifyInVpa,
				data[i].Peer_id,
				data[i].Segment_id,
				ok,
			)
			if err != nil {
				//fmt.Printf("\x1b[%dm[err]\x1b[0m vpa submit failed: %v\n", 41, err)
				logger.ErrLogger.Sugar().Errorf("[C%v][%v][%v] vpa submit failed,err:%v", data[i].Peer_id, data[i].Segment_id, ok, err)
				continue
			}
			//fmt.Printf("\x1b[%dm[ok]\x1b[0m [C%v][%v] vpa verify suc\n", 42, data[i].Peer_id, data[i].Segment_id)
			logger.InfoLogger.Sugar().Infof("[C%v][%v][%v] vpa submit suc", data[i].Peer_id, data[i].Segment_id, ok)
		}
	}
}

func verifyVpb() {
	var (
		err       error
		i         int
		j         int
		ok        bool
		segtype   uint8
		prooftype uint8
		sealcid   string
		proofs    string
	)
	for range time.Tick(time.Minute) {
		var data []chain.UnVerifiedVpaVpb
		data, err = chain.GetUnverifiedVpaVpb(
			configs.ChainModule_SegmentBook,
			configs.ChainModule_SegmentBook_UnVerifiedB,
		)
		if err != nil {
			//fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
			logger.ErrLogger.Sugar().Errorf("%v", err)
			continue
		}
		//fmt.Println("Number of unverified vpb: ", len(data))

		if len(data) == 0 {
			time.Sleep(time.Minute)
		} else {
			logger.InfoLogger.Sugar().Infof("Number of unverified vpb: %v", len(data))
		}
		for i = 0; i < len(data); i++ {
			proofs = ""
			sealcid = ""
			for j = 0; j < len(data[i].Proof); j++ {
				temp := fmt.Sprintf("%c", data[i].Proof[j])
				proofs += temp
			}
			p, err := types.HexDecodeString(proofs)
			if err != nil {
				logger.ErrLogger.Sugar().Errorf("%v", err)
			}
			// fmt.Println(proofs)
			//fmt.Println(p)
			for j = 0; j < len(data[i].Sealed_cid); j++ {
				temp := fmt.Sprintf("%c", data[i].Sealed_cid[j])
				sealcid += temp
			}
			//fmt.Println(sealcid)
			sizetypes := fmt.Sprintf("%v", data[i].Size_type)
			switch sizetypes {
			case "8":
				segtype = 1
				prooftype = 6
			case "512":
				segtype = 2
				prooftype = 7
			}
			if segtype == 0 {
				//fmt.Printf("\x1b[%dm[err]\x1b[0m segtype is invalid\n", 41)
				logger.ErrLogger.Sugar().Errorf("[C%v] segtype is invalid", data[i].Peer_id)
				continue
			}
			pf := proof.PoStProof{
				PoStProof:  abi.RegisteredPoStProof(prooftype),
				ProofBytes: p,
			}
			ok, err = verifyVpbProof(uint64(data[i].Peer_id), uint64(data[i].Segment_id), uint32(data[i].Rand), segtype, sealcid, []proof.PoStProof{pf})
			if err != nil {
				//fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
				logger.ErrLogger.Sugar().Errorf("[C%v] %v", data[i].Peer_id, err)
				continue
			}

			err = chain.VerifyInVpaOrVpb(
				configs.Confile.SchedulerInfo.TransactionPrK,
				configs.ChainTx_SegmentBook_VerifyInVpb,
				data[i].Peer_id,
				data[i].Segment_id,
				ok,
			)
			if err != nil {
				//fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
				logger.ErrLogger.Sugar().Errorf("[C%v][%v][%v] vpb submit failed,err:%v", data[i].Peer_id, data[i].Segment_id, ok, err)
				continue
			}
			//fmt.Printf("\x1b[%dm[ok]\x1b[0m [C%v][%v] vpb verify suc\n", 42, data[i].Peer_id, data[i].Segment_id)
			logger.InfoLogger.Sugar().Infof("[C%v][%v][%v] vpb submit suc", data[i].Peer_id, data[i].Segment_id, ok)
		}
	}
}

func verifyVpc() {
	var (
		err     error
		i       int
		j       int
		ok      bool
		segtype uint8
	)
	segtype = 1
	for range time.Tick(time.Second) {
		var data []chain.UnVerifiedVpc
		data, err = chain.GetUnverifiedVpc(
			configs.ChainModule_SegmentBook,
			configs.ChainModule_SegmentBook_UnVerifiedC,
		)
		if err != nil {
			//fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
			logger.ErrLogger.Sugar().Errorf("%v", err)
			continue
		}
		//fmt.Println("Number of unverified vpc: ", len(data))
		if len(data) == 0 {
			time.Sleep(time.Minute)
		} else {
			logger.InfoLogger.Sugar().Infof("Number of unverified vpc: %v", len(data))
		}
		for i = 0; i < len(data); i++ {
			var proof = make([][]byte, len(data[i].Proof))
			var sealcid = make([]string, 0)
			var uncid = make([]string, 0)
			for j = 0; j < len(data[i].Proof); j++ {
				proof[j] = make([]byte, 0)
				proof[j] = append(proof[j], data[i].Proof[j]...)
			}

			cid := ""
			for j = 0; j < len(data[i].Sealed_cid); j++ {
				cid = ""
				for k := 0; k < len(data[i].Sealed_cid[j]); k++ {
					temp := fmt.Sprintf("%c", data[i].Sealed_cid[j][k])
					cid += temp
				}
				sealcid = append(sealcid, cid)
			}
			for j = 0; j < len(data[i].Unsealed_cid); j++ {
				cid = ""
				for k := 0; k < len(data[i].Unsealed_cid[j]); k++ {
					temp := fmt.Sprintf("%c", data[i].Unsealed_cid[j][k])
					cid += temp
				}
				uncid = append(uncid, cid)
			}
			// fmt.Println("sealcid:", sealcid)
			// fmt.Println("uncid:", uncid)
			// fmt.Println("proof:", proof)

			ok, err = verifyVpcProof(uint64(data[i].Peer_id), uint64(data[i].Segment_id), uint32(data[i].Rand), segtype, sealcid, uncid, proof)
			if err != nil {
				//fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
				logger.ErrLogger.Sugar().Errorf("[C%v] %v", data[i].Peer_id, err)
				continue
			}

			err = chain.VerifyInVpc(
				configs.Confile.SchedulerInfo.TransactionPrK,
				configs.ChainTx_SegmentBook_VerifyInVpc,
				data[i].Peer_id,
				data[i].Segment_id,
				data[i].Unsealed_cid,
				ok,
			)
			if err != nil {
				//fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
				logger.ErrLogger.Sugar().Errorf("[C%v][%v][%v] vpc submit failed,err:%v", data[i].Peer_id, data[i].Segment_id, ok, err)
				continue
			}
			//fmt.Printf("\x1b[%dm[ok]\x1b[0m [C%v][%v] vpc verify suc\n", 42, data[i].Peer_id, data[i].Segment_id)
			logger.InfoLogger.Sugar().Infof("[C%v][%v][%v] vpc submit suc", data[i].Peer_id, data[i].Segment_id, ok)
		}
	}
}

func verifyVpd() {
	var (
		err     error
		i       int
		j       int
		ok      bool
		segtype uint8
	)
	segtype = 1
	for range time.Tick(time.Minute) {
		var data []chain.UnVerifiedVpd
		data, err = chain.GetUnverifiedVpd(
			configs.ChainModule_SegmentBook,
			configs.ChainModule_SegmentBook_UnVerifiedD,
		)
		if err != nil {
			//fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
			logger.ErrLogger.Sugar().Errorf("%v", err)
			continue
		}
		//fmt.Println("Number of unverified vpd: ", len(data))
		if len(data) == 0 {
			time.Sleep(time.Minute)
		} else {
			logger.InfoLogger.Sugar().Infof("Number of unverified vpd: %v", len(data))
		}
		for i = 0; i < len(data); i++ {
			var postproof = make([]proof.PoStProof, len(data[i].Proof))
			for j = 0; j < len(data[i].Proof); j++ {
				postproof[j].ProofBytes = make([]byte, 0)
				postproof[j].PoStProof = abi.RegisteredPoStProof(configs.FilePostProof)
				postproof[j].ProofBytes = append(postproof[j].ProofBytes, data[i].Proof[j]...)
			}
			var sealcid = make([]string, 0)
			for j = 0; j < len(data[i].Sealed_cid); j++ {
				var cid string
				for k := 0; k < len(data[i].Sealed_cid[j]); k++ {
					temp := fmt.Sprintf("%c", data[i].Sealed_cid[j][k])
					cid += temp
				}
				sealcid = append(sealcid, cid)
			}
			ok, err = verifyVpdProof(uint64(data[i].Peer_id), uint64(data[i].Segment_id), uint32(data[i].Rand), segtype, sealcid, postproof)
			if err != nil {
				//fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
				logger.ErrLogger.Sugar().Errorf("[C%v] %v", data[i].Peer_id, err)
				continue
			}

			err = chain.VerifyInVpaOrVpb(
				configs.Confile.SchedulerInfo.TransactionPrK,
				configs.ChainTx_SegmentBook_VerifyInVpd,
				data[i].Peer_id,
				data[i].Segment_id,
				ok,
			)
			if err != nil {
				//fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
				logger.ErrLogger.Sugar().Errorf("[C%v][%v][%v] vpc submit failed,err:%v", data[i].Peer_id, data[i].Segment_id, ok, err)
				continue
			}
			//fmt.Printf("\x1b[%dm[ok]\x1b[0m [C%v][%v] vpd verify suc\n", 42, data[i].Peer_id, data[i].Segment_id)
			logger.InfoLogger.Sugar().Infof("[C%v][%v][%v] vpd submit suc", data[i].Peer_id, data[i].Segment_id, ok)
		}
	}
}
