package proof

import (
	"cess-scheduler/configs"
	"cess-scheduler/internal/chain"
	. "cess-scheduler/internal/logger"
	api "cess-scheduler/internal/proof/apiv1"
	"cess-scheduler/internal/rpc"
	p "cess-scheduler/internal/rpc/protobuf"
	"cess-scheduler/tools"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"time"

	"google.golang.org/protobuf/proto"
)

type TagInfo struct {
	T      api.FileTagT `json:"file_tag_t"`
	Sigmas [][]byte     `json:"sigmas"`
}

// Enable the verification proof module
func Chain_Main() {
	go processingProof()
	// go verifyVpa()
	// go verifyVpb()
	// go verifyVpc()
	// go verifyVpd()
}

// verifyVpa is used to verify the porep of idle data segments.
// normally it will run forever.
// func verifyVpa() {
// 	var (
// 		err     error
// 		ok      bool
// 		segtype uint8
// 		sealcid string
// 		proofs  string
// 		data    []chain.UnVerifiedVpaVpb
// 	)
// 	for {
// 		time.Sleep(time.Second * time.Duration(tools.RandomInRange(20, 80)))
// 		data, err = chain.GetUnverifiedVpaVpb(
// 			chain.State_SegmentBook,
// 			chain.SegmentBook_UnVerifiedA,
// 		)
// 		if err != nil {
// 			Err.Sugar().Errorf("%v", err)
// 			continue
// 		}
// 		if len(data) == 0 {
// 			continue
// 		}
// 		Out.Sugar().Infof("Number of unVerifiedVpa: %v", len(data))
// 		for i := 0; i < len(data); i++ {
// 			proofs = ""
// 			sealcid = ""
// 			for j := 0; j < len(data[i].Proof); j++ {
// 				temp := fmt.Sprintf("%c", data[i].Proof[j])
// 				proofs += temp
// 			}
// 			p, err := types.HexDecodeString(proofs)
// 			if err != nil {
// 				Err.Sugar().Errorf("%v", err)
// 				continue
// 			}
// 			for j := 0; j < len(data[i].Sealed_cid); j++ {
// 				temp := fmt.Sprintf("%c", data[i].Sealed_cid[j])
// 				sealcid += temp
// 			}
// 			sizetypes := fmt.Sprintf("%v", data[i].Size_type)
// 			switch sizetypes {
// 			case "8":
// 				segtype = 1
// 			case "512":
// 				segtype = 2
// 			}
// 			if segtype == 0 {
// 				Err.Sugar().Errorf("[C%v][%v] segtype is invalid", data[i].Peer_id, sizetypes)
// 				continue
// 			}
// 			ok, err = verifyVpaProof(uint64(data[i].Peer_id), uint64(data[i].Segment_id), uint32(data[i].Rand), segtype, sealcid, p)
// 			if err != nil {
// 				Err.Sugar().Errorf("[C%v] %v", data[i].Peer_id, err)
// 				continue
// 			}
// 			err = chain.VerifyInVpaOrVpbOrVpd(
// 				configs.Confile.SchedulerInfo.ControllerAccountPhrase,
// 				chain.ChainTx_SegmentBook_VerifyInVpa,
// 				data[i].Peer_id,
// 				data[i].Segment_id,
// 				ok,
// 			)
// 			if err != nil {
// 				Err.Sugar().Errorf("[C%v][%v][%v] vpa submit failed,err:%v", data[i].Peer_id, data[i].Segment_id, ok, err)
// 				continue
// 			}
// 			Out.Sugar().Infof("[C%v][%v][%v] vpa submit suc", data[i].Peer_id, data[i].Segment_id, ok)
// 		}
// 	}
// }

// verifyVpb is used to verify the post of idle data segments.
// normally it will run forever.
// func verifyVpb() {
// 	var (
// 		err       error
// 		ok        bool
// 		segtype   uint8
// 		prooftype uint8
// 		sealcid   string
// 		proofs    string
// 		data      []chain.UnVerifiedVpaVpb
// 	)
// 	for {
// 		time.Sleep(time.Second * time.Duration(tools.RandomInRange(20, 80)))
// 		data, err = chain.GetUnverifiedVpaVpb(
// 			chain.State_SegmentBook,
// 			chain.SegmentBook_UnVerifiedB,
// 		)
// 		if err != nil {
// 			Err.Sugar().Errorf("%v", err)
// 			continue
// 		}
// 		if len(data) == 0 {
// 			continue
// 		}
// 		Out.Sugar().Infof("Number of unVerifiedVpb:%v", len(data))
// 		for i := 0; i < len(data); i++ {
// 			proofs = ""
// 			sealcid = ""
// 			for j := 0; j < len(data[i].Proof); j++ {
// 				temp := fmt.Sprintf("%c", data[i].Proof[j])
// 				proofs += temp
// 			}
// 			p, err := types.HexDecodeString(proofs)
// 			if err != nil {
// 				Err.Sugar().Errorf("%v", err)
// 				continue
// 			}
// 			for j := 0; j < len(data[i].Sealed_cid); j++ {
// 				temp := fmt.Sprintf("%c", data[i].Sealed_cid[j])
// 				sealcid += temp
// 			}
// 			sizetypes := fmt.Sprintf("%v", data[i].Size_type)
// 			switch sizetypes {
// 			case "8":
// 				segtype = 1
// 				prooftype = 6
// 			case "512":
// 				segtype = 2
// 				prooftype = 7
// 			}
// 			if segtype == 0 {
// 				Err.Sugar().Errorf("[C%v] segtype is invalid", data[i].Peer_id)
// 				continue
// 			}
// 			pf := proof.PoStProof{
// 				PoStProof:  abi.RegisteredPoStProof(prooftype),
// 				ProofBytes: p,
// 			}
// 			ok, err = verifyVpbProof(uint64(data[i].Peer_id), uint64(data[i].Segment_id), uint32(data[i].Rand), segtype, sealcid, []proof.PoStProof{pf})
// 			if err != nil {
// 				Err.Sugar().Errorf("[C%v] %v", data[i].Peer_id, err)
// 				continue
// 			}

// 			err = chain.VerifyInVpaOrVpbOrVpd(
// 				configs.Confile.SchedulerInfo.ControllerAccountPhrase,
// 				chain.ChainTx_SegmentBook_VerifyInVpb,
// 				data[i].Peer_id,
// 				data[i].Segment_id,
// 				ok,
// 			)
// 			if err != nil {
// 				Err.Sugar().Errorf("[C%v][%v][%v] vpb submit failed,err:%v", data[i].Peer_id, data[i].Segment_id, ok, err)
// 				continue
// 			}
// 			Out.Sugar().Infof("[C%v][%v][%v] vpb submit suc", data[i].Peer_id, data[i].Segment_id, ok)
// 		}
// 	}
// }

// verifyVpc is used to verify the porep of service data segments.
// normally it will run forever.
// func verifyVpc() {
// 	var (
// 		err  error
// 		ok   bool
// 		data []chain.UnVerifiedVpc
// 	)
// 	for {
// 		time.Sleep(time.Second * time.Duration(tools.RandomInRange(20, 80)))
// 		data, err = chain.GetUnverifiedVpc(
// 			chain.State_SegmentBook,
// 			chain.SegmentBook_UnVerifiedC,
// 		)
// 		if err != nil {
// 			Err.Sugar().Errorf("%v", err)
// 			continue
// 		}
// 		if len(data) == 0 {
// 			continue
// 		}
// 		Out.Sugar().Infof("Number of unVerifiedVpc:%v", len(data))
// 		for i := 0; i < len(data); i++ {
// 			var proof = make([][]byte, len(data[i].Proof))
// 			var sealcid = make([]string, 0)
// 			var uncid = make([]string, 0)
// 			for j := 0; j < len(data[i].Proof); j++ {
// 				proof[j] = make([]byte, 0)
// 				proof[j] = append(proof[j], data[i].Proof[j]...)
// 			}

// 			cid := ""
// 			for j := 0; j < len(data[i].Sealed_cid); j++ {
// 				cid = ""
// 				for k := 0; k < len(data[i].Sealed_cid[j]); k++ {
// 					temp := fmt.Sprintf("%c", data[i].Sealed_cid[j][k])
// 					cid += temp
// 				}
// 				sealcid = append(sealcid, cid)
// 			}
// 			for j := 0; j < len(data[i].Unsealed_cid); j++ {
// 				cid = ""
// 				for k := 0; k < len(data[i].Unsealed_cid[j]); k++ {
// 					temp := fmt.Sprintf("%c", data[i].Unsealed_cid[j][k])
// 					cid += temp
// 				}
// 				uncid = append(uncid, cid)
// 			}

// 			ok, err = verifyVpcProof(uint64(data[i].Peer_id), uint64(data[i].Segment_id), uint32(data[i].Rand), configs.SegMentType_8M, sealcid, uncid, proof)
// 			if err != nil {
// 				Err.Sugar().Errorf("[C%v] %v", data[i].Peer_id, err)
// 				continue
// 			}

// 			err = chain.VerifyInVpc(
// 				configs.Confile.SchedulerInfo.ControllerAccountPhrase,
// 				chain.ChainTx_SegmentBook_VerifyInVpc,
// 				data[i].Peer_id,
// 				data[i].Segment_id,
// 				data[i].Unsealed_cid,
// 				ok,
// 			)
// 			if err != nil {
// 				Err.Sugar().Errorf("[C%v][%v][%v] vpc submit failed,err:%v", data[i].Peer_id, data[i].Segment_id, ok, err)
// 				continue
// 			}
// 			Out.Sugar().Infof("[C%v][%v][%v] vpc submit suc", data[i].Peer_id, data[i].Segment_id, ok)
// 		}
// 	}
// }

// verifyVpd is used to verify the post of service data segments.
// normally it will run forever.
// func verifyVpd() {
// 	var (
// 		err          error
// 		ok           bool
// 		data         []chain.UnVerifiedVpd
// 		vpdfailcount = make(map[uint64]uint8, 0)
// 	)
// 	for {
// 		time.Sleep(time.Second * time.Duration(tools.RandomInRange(20, 80)))
// 		data, err = chain.GetUnverifiedVpd(
// 			chain.State_SegmentBook,
// 			chain.SegmentBook_UnVerifiedD,
// 		)
// 		if err != nil {
// 			Err.Sugar().Errorf("%v", err)
// 			continue
// 		}
// 		if len(data) == 0 {
// 			continue
// 		}
// 		Out.Sugar().Infof("Number of unVerifiedVpd:%v", len(data))
// 		for i := 0; i < len(data); i++ {
// 			var postproof = make([]proof.PoStProof, len(data[i].Proof))
// 			for j := 0; j < len(data[i].Proof); j++ {
// 				postproof[j].ProofBytes = make([]byte, 0)
// 				postproof[j].PoStProof = abi.RegisteredPoStProof(configs.FilePostProof)
// 				postproof[j].ProofBytes = append(postproof[j].ProofBytes, data[i].Proof[j]...)
// 			}
// 			var sealcid = make([]string, 0)
// 			for j := 0; j < len(data[i].Sealed_cid); j++ {
// 				var cid string
// 				for k := 0; k < len(data[i].Sealed_cid[j]); k++ {
// 					temp := fmt.Sprintf("%c", data[i].Sealed_cid[j][k])
// 					cid += temp
// 				}
// 				sealcid = append(sealcid, cid)
// 			}
// 			ok, err = verifyVpdProof(uint64(data[i].Peer_id), uint64(data[i].Segment_id), uint32(data[i].Rand), configs.SegMentType_8M, sealcid, postproof)
// 			if err != nil {
// 				Err.Sugar().Errorf("[C%v][%v] %v", data[i].Peer_id, data[i].Segment_id, err)
// 				continue
// 			}
// 			if ok {
// 				vpdfailcount[uint64(data[i].Segment_id)] = 0
// 			} else {
// 				vpdfailcount[uint64(data[i].Segment_id)]++
// 			}
// 			if ok || vpdfailcount[uint64(data[i].Segment_id)] >= uint8(3) {
// 				err = chain.VerifyInVpaOrVpbOrVpd(
// 					configs.Confile.SchedulerInfo.ControllerAccountPhrase,
// 					chain.ChainTx_SegmentBook_VerifyInVpd,
// 					data[i].Peer_id,
// 					data[i].Segment_id,
// 					ok,
// 				)
// 				if err != nil {
// 					Err.Sugar().Errorf("[C%v][%v][%v] vpd submit failed,err:%v", data[i].Peer_id, data[i].Segment_id, ok, err)
// 					continue
// 				}
// 				Out.Sugar().Infof("[C%v][%v][%v][%v] vpd submit suc", data[i].Peer_id, data[i].Segment_id, ok, vpdfailcount[uint64(data[i].Segment_id)])
// 				if !ok {
// 					delete(vpdfailcount, uint64(data[i].Segment_id))
// 				}
// 			}
// 		}
// 	}
// }

func processingProof() {
	var (
		err         error
		code        int
		puk         chain.Chain_SchedulerPuk
		poDR2verify api.PoDR2Verify
		proofs      []chain.Chain_Proofs
	)
	puk, _, err = chain.GetSchedulerPukFromChain()
	if err != nil {
		fmt.Printf("\x1b[%dm[err]\x1b[0m %v\n", 41, err)
		os.Exit(1)
	}
	for {
		time.Sleep(time.Second * time.Duration(tools.RandomInRange(10, 30)))
		proofs, code, err = chain.GetProofsFromChain(configs.Confile.SchedulerInfo.ControllerAccountPhrase)
		if err != nil {
			if code != configs.Code_404 {
				Err.Sugar().Errorf("[%v] %v", err)
			}
			continue
		}
		for i := 0; i < len(proofs); i++ {
			tmp := make(map[int]*big.Int, len(proofs[i].Challenge_info.Block_list))
			for j := 0; j < len(proofs[i].Challenge_info.Block_list); j++ {
				tmp[int(proofs[i].Challenge_info.Block_list[j])] = new(big.Int).SetBytes(proofs[i].Challenge_info.Random[j])
			}

			var reqtag p.ReadTagReq
			reqtag.FileId = string(proofs[i].Challenge_info.File_id)
			reqtag.Acc, err = chain.GetAddressByPrk(configs.Confile.SchedulerInfo.ControllerAccountPhrase)
			if err != nil {
				Out.Sugar().Infof("%v", err)
				continue
			}
			req_proto, err := proto.Marshal(&reqtag)
			if err != nil {
				Out.Sugar().Infof("%v", err)
				continue
			}
			minerDetails, _, err := chain.GetMinerDetailsById(uint64(proofs[i].Miner_id))
			if err != nil {
				Out.Sugar().Infof("%v", err)
				continue
			}
			respData, err := rpc.WriteData(string(minerDetails.Ip), configs.RpcService_Miner, configs.RpcMethod_Miner_ReadFileTag, req_proto)
			if err != nil {
				Out.Sugar().Infof("%v", err)
				continue
			}
			var tag TagInfo
			err = json.Unmarshal(respData, &tag)
			if err != nil {
				Out.Sugar().Infof("%v", err)
				continue
			}
			qSlice, err := api.PoDR2ChallengeGenerateFromChain(tmp, string(puk.Shared_params))
			if err != nil {
				Err.Sugar().Errorf("%v", err)
				continue
			}
			poDR2verify.QSlice = qSlice
			for j := 0; j < len(proofs[i].Mu); j++ {
				poDR2verify.MU[i] = append(poDR2verify.MU[i], proofs[i].Mu[i]...)
			}
			poDR2verify.Sigma = proofs[i].Sigma
			poDR2verify.T = tag.T
			result := poDR2verify.PoDR2ProofVerify(puk.Shared_g, puk.Spk, string(puk.Shared_params))
			chain.PutProofResult(configs.Confile.SchedulerInfo.ControllerAccountPhrase, proofs[i].Miner_id, proofs[i].Challenge_info.File_id, result)
		}
	}
}
