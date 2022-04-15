package proof

import (
	"cess-scheduler/configs"
	"cess-scheduler/internal/chain"
	. "cess-scheduler/internal/logger"
	"cess-scheduler/tools"
	"fmt"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/actors/runtime/proof"
)

// Enable the verification proof module
func Chain_Main() {
	go verifyVpa()
	go verifyVpb()
	go verifyVpc()
	go verifyVpd()
}

// verifyVpa is used to verify the porep of idle data segments.
// normally it will run forever.
func verifyVpa() {
	var (
		err     error
		ok      bool
		segtype uint8
		sealcid string
		proofs  string
		data    []chain.UnVerifiedVpaVpb
	)
	for {
		time.Sleep(time.Second * time.Duration(tools.RandomInRange(20, 80)))
		data, err = chain.GetUnverifiedVpaVpb(
			configs.ChainModule_SegmentBook,
			configs.ChainModule_SegmentBook_UnVerifiedA,
		)
		if err != nil {
			Err.Sugar().Errorf("%v", err)
			continue
		}
		if len(data) == 0 {
			continue
		}
		Out.Sugar().Infof("Number of unVerifiedVpa: %v", len(data))
		for i := 0; i < len(data); i++ {
			proofs = ""
			sealcid = ""
			for j := 0; j < len(data[i].Proof); j++ {
				temp := fmt.Sprintf("%c", data[i].Proof[j])
				proofs += temp
			}
			p, err := types.HexDecodeString(proofs)
			if err != nil {
				Err.Sugar().Errorf("%v", err)
				continue
			}
			for j := 0; j < len(data[i].Sealed_cid); j++ {
				temp := fmt.Sprintf("%c", data[i].Sealed_cid[j])
				sealcid += temp
			}
			sizetypes := fmt.Sprintf("%v", data[i].Size_type)
			switch sizetypes {
			case "8":
				segtype = 1
			case "512":
				segtype = 2
			}
			if segtype == 0 {
				Err.Sugar().Errorf("[C%v][%v] segtype is invalid", data[i].Peer_id, sizetypes)
				continue
			}
			ok, err = verifyVpaProof(uint64(data[i].Peer_id), uint64(data[i].Segment_id), uint32(data[i].Rand), segtype, sealcid, p)
			if err != nil {
				Err.Sugar().Errorf("[C%v] %v", data[i].Peer_id, err)
				continue
			}
			err = chain.VerifyInVpaOrVpbOrVpd(
				configs.Confile.SchedulerInfo.ControllerAccountPhrase,
				configs.ChainTx_SegmentBook_VerifyInVpa,
				data[i].Peer_id,
				data[i].Segment_id,
				ok,
			)
			if err != nil {
				Err.Sugar().Errorf("[C%v][%v][%v] vpa submit failed,err:%v", data[i].Peer_id, data[i].Segment_id, ok, err)
				continue
			}
			Out.Sugar().Infof("[C%v][%v][%v] vpa submit suc", data[i].Peer_id, data[i].Segment_id, ok)
		}
	}
}

// verifyVpb is used to verify the post of idle data segments.
// normally it will run forever.
func verifyVpb() {
	var (
		err       error
		ok        bool
		segtype   uint8
		prooftype uint8
		sealcid   string
		proofs    string
		data      []chain.UnVerifiedVpaVpb
	)
	for {
		time.Sleep(time.Second * time.Duration(tools.RandomInRange(20, 80)))
		data, err = chain.GetUnverifiedVpaVpb(
			configs.ChainModule_SegmentBook,
			configs.ChainModule_SegmentBook_UnVerifiedB,
		)
		if err != nil {
			Err.Sugar().Errorf("%v", err)
			continue
		}
		if len(data) == 0 {
			continue
		}
		Out.Sugar().Infof("Number of unVerifiedVpb:%v", len(data))
		for i := 0; i < len(data); i++ {
			proofs = ""
			sealcid = ""
			for j := 0; j < len(data[i].Proof); j++ {
				temp := fmt.Sprintf("%c", data[i].Proof[j])
				proofs += temp
			}
			p, err := types.HexDecodeString(proofs)
			if err != nil {
				Err.Sugar().Errorf("%v", err)
				continue
			}
			for j := 0; j < len(data[i].Sealed_cid); j++ {
				temp := fmt.Sprintf("%c", data[i].Sealed_cid[j])
				sealcid += temp
			}
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
				Err.Sugar().Errorf("[C%v] segtype is invalid", data[i].Peer_id)
				continue
			}
			pf := proof.PoStProof{
				PoStProof:  abi.RegisteredPoStProof(prooftype),
				ProofBytes: p,
			}
			ok, err = verifyVpbProof(uint64(data[i].Peer_id), uint64(data[i].Segment_id), uint32(data[i].Rand), segtype, sealcid, []proof.PoStProof{pf})
			if err != nil {
				Err.Sugar().Errorf("[C%v] %v", data[i].Peer_id, err)
				continue
			}

			err = chain.VerifyInVpaOrVpbOrVpd(
				configs.Confile.SchedulerInfo.ControllerAccountPhrase,
				configs.ChainTx_SegmentBook_VerifyInVpb,
				data[i].Peer_id,
				data[i].Segment_id,
				ok,
			)
			if err != nil {
				Err.Sugar().Errorf("[C%v][%v][%v] vpb submit failed,err:%v", data[i].Peer_id, data[i].Segment_id, ok, err)
				continue
			}
			Out.Sugar().Infof("[C%v][%v][%v] vpb submit suc", data[i].Peer_id, data[i].Segment_id, ok)
		}
	}
}

// verifyVpc is used to verify the porep of service data segments.
// normally it will run forever.
func verifyVpc() {
	var (
		err  error
		ok   bool
		data []chain.UnVerifiedVpc
	)
	for {
		time.Sleep(time.Second * time.Duration(tools.RandomInRange(20, 80)))
		data, err = chain.GetUnverifiedVpc(
			configs.ChainModule_SegmentBook,
			configs.ChainModule_SegmentBook_UnVerifiedC,
		)
		if err != nil {
			Err.Sugar().Errorf("%v", err)
			continue
		}
		if len(data) == 0 {
			continue
		}
		Out.Sugar().Infof("Number of unVerifiedVpc:%v", len(data))
		for i := 0; i < len(data); i++ {
			var proof = make([][]byte, len(data[i].Proof))
			var sealcid = make([]string, 0)
			var uncid = make([]string, 0)
			for j := 0; j < len(data[i].Proof); j++ {
				proof[j] = make([]byte, 0)
				proof[j] = append(proof[j], data[i].Proof[j]...)
			}

			cid := ""
			for j := 0; j < len(data[i].Sealed_cid); j++ {
				cid = ""
				for k := 0; k < len(data[i].Sealed_cid[j]); k++ {
					temp := fmt.Sprintf("%c", data[i].Sealed_cid[j][k])
					cid += temp
				}
				sealcid = append(sealcid, cid)
			}
			for j := 0; j < len(data[i].Unsealed_cid); j++ {
				cid = ""
				for k := 0; k < len(data[i].Unsealed_cid[j]); k++ {
					temp := fmt.Sprintf("%c", data[i].Unsealed_cid[j][k])
					cid += temp
				}
				uncid = append(uncid, cid)
			}

			ok, err = verifyVpcProof(uint64(data[i].Peer_id), uint64(data[i].Segment_id), uint32(data[i].Rand), configs.SegMentType_8M, sealcid, uncid, proof)
			if err != nil {
				Err.Sugar().Errorf("[C%v] %v", data[i].Peer_id, err)
				continue
			}

			err = chain.VerifyInVpc(
				configs.Confile.SchedulerInfo.ControllerAccountPhrase,
				configs.ChainTx_SegmentBook_VerifyInVpc,
				data[i].Peer_id,
				data[i].Segment_id,
				data[i].Unsealed_cid,
				ok,
			)
			if err != nil {
				Err.Sugar().Errorf("[C%v][%v][%v] vpc submit failed,err:%v", data[i].Peer_id, data[i].Segment_id, ok, err)
				continue
			}
			Out.Sugar().Infof("[C%v][%v][%v] vpc submit suc", data[i].Peer_id, data[i].Segment_id, ok)
		}
	}
}

// verifyVpd is used to verify the post of service data segments.
// normally it will run forever.
func verifyVpd() {
	var (
		err          error
		ok           bool
		data         []chain.UnVerifiedVpd
		vpdfailcount = make(map[uint64]uint8, 0)
	)
	for {
		time.Sleep(time.Second * time.Duration(tools.RandomInRange(20, 80)))
		data, err = chain.GetUnverifiedVpd(
			configs.ChainModule_SegmentBook,
			configs.ChainModule_SegmentBook_UnVerifiedD,
		)
		if err != nil {
			Err.Sugar().Errorf("%v", err)
			continue
		}
		if len(data) == 0 {
			continue
		}
		Out.Sugar().Infof("Number of unVerifiedVpd:%v", len(data))
		for i := 0; i < len(data); i++ {
			var postproof = make([]proof.PoStProof, len(data[i].Proof))
			for j := 0; j < len(data[i].Proof); j++ {
				postproof[j].ProofBytes = make([]byte, 0)
				postproof[j].PoStProof = abi.RegisteredPoStProof(configs.FilePostProof)
				postproof[j].ProofBytes = append(postproof[j].ProofBytes, data[i].Proof[j]...)
			}
			var sealcid = make([]string, 0)
			for j := 0; j < len(data[i].Sealed_cid); j++ {
				var cid string
				for k := 0; k < len(data[i].Sealed_cid[j]); k++ {
					temp := fmt.Sprintf("%c", data[i].Sealed_cid[j][k])
					cid += temp
				}
				sealcid = append(sealcid, cid)
			}
			ok, err = verifyVpdProof(uint64(data[i].Peer_id), uint64(data[i].Segment_id), uint32(data[i].Rand), configs.SegMentType_8M, sealcid, postproof)
			if err != nil {
				Err.Sugar().Errorf("[C%v][%v] %v", data[i].Peer_id, data[i].Segment_id, err)
				continue
			}
			if ok {
				vpdfailcount[uint64(data[i].Segment_id)] = 0
			} else {
				vpdfailcount[uint64(data[i].Segment_id)]++
			}
			if ok || vpdfailcount[uint64(data[i].Segment_id)] >= uint8(3) {
				err = chain.VerifyInVpaOrVpbOrVpd(
					configs.Confile.SchedulerInfo.ControllerAccountPhrase,
					configs.ChainTx_SegmentBook_VerifyInVpd,
					data[i].Peer_id,
					data[i].Segment_id,
					ok,
				)
				if err != nil {
					Err.Sugar().Errorf("[C%v][%v][%v] vpd submit failed,err:%v", data[i].Peer_id, data[i].Segment_id, ok, err)
					continue
				}
				Out.Sugar().Infof("[C%v][%v][%v][%v] vpd submit suc", data[i].Peer_id, data[i].Segment_id, ok, vpdfailcount[uint64(data[i].Segment_id)])
				if !ok {
					delete(vpdfailcount, uint64(data[i].Segment_id))
				}
			}
		}
	}
}
