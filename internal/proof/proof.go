package proof

import (
	"fmt"
	"scheduler-mining/internal/logger"
	"scheduler-mining/tools"

	"github.com/filecoin-project/go-state-types/abi"
	prf "github.com/filecoin-project/specs-actors/actors/runtime/proof"
	cid "github.com/ipfs/go-cid"
	"github.com/pkg/errors"
)

func verifyVpaProof(id, segid uint64, rand uint32, segtype uint8, sealedcid string, proof []byte) (bool, error) {
	defer func() {
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	sectorId := SectorID{PeerID: abi.ActorID(id), SectorNum: abi.SectorNumber(segid)}
	seed, err := tools.IntegerToBytes(rand)
	if err != nil {
		return false, errors.Wrap(err, "tools.IntegerToBytes err")
	}
	scid, err := cid.Parse(sealedcid)
	if err != nil {
		return false, errors.Wrap(err, "cid.Parse err")
	}
	fmt.Println(sectorId, " - ", seed, " - ", abi.RegisteredSealProof(segtype), " - ", scid, " - ", proof)
	isValid := VerifyFileOnceForIdle(sectorId, seed, seed, abi.RegisteredSealProof(segtype), scid, proof)
	return isValid, nil
}

func verifyVpbProof(id, segid uint64, rand uint32, segtype uint8, sealedcid string, pf []prf.PoStProof) (bool, error) {
	defer func() {
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	sectorId := SectorID{PeerID: abi.ActorID(id), SectorNum: abi.SectorNumber(segid)}
	seed, err := tools.IntegerToBytes(rand)
	if err != nil {
		return false, errors.Wrap(err, "tools.IntegerToBytes err")
	}
	scid, err := cid.Parse(sealedcid)
	if err != nil {
		return false, errors.Wrap(err, "cid.Parse err")
	}

	isValid := VerifyFileInterval(sectorId, abi.RegisteredSealProof(segtype), seed, []cid.Cid{scid}, pf)
	return isValid, nil
}

func verifyVpcProof(id, segid uint64, rand uint32, segtype uint8, sealcid, uncid []string, proof [][]byte) (bool, error) {
	defer func() {
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	sectorId := SectorID{PeerID: abi.ActorID(id), SectorNum: abi.SectorNumber(segid)}
	seed, err := tools.IntegerToBytes(rand)
	if err != nil {
		return false, errors.Wrap(err, "tools.IntegerToBytes err")
	}

	var unsealedCid = make([]cid.Cid, 0)
	var sealedCid = make([]cid.Cid, 0)
	for i := 0; i < len(uncid); i++ {
		tmp, err := cid.Parse(uncid[i])
		if err != nil {
			return false, errors.Wrap(err, "cid.Parse err")
		}
		unsealedCid = append(unsealedCid, tmp)
	}
	for i := 0; i < len(sealcid); i++ {
		tmp, err := cid.Parse(sealcid[i])
		if err != nil {
			return false, errors.Wrap(err, "cid.Parse err")
		}
		sealedCid = append(sealedCid, tmp)
	}
	isValid := VerifyFileOnce(sectorId, seed, seed, abi.RegisteredSealProof(segtype), unsealedCid, sealedCid, proof)
	return isValid, nil
}

func verifyVpdProof(id, segid uint64, rand uint32, segtype uint8, sealcid []string, pf []prf.PoStProof) (bool, error) {
	defer func() {
		err := recover()
		if err != nil {
			logger.ErrLogger.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	sectorId := SectorID{PeerID: abi.ActorID(id), SectorNum: abi.SectorNumber(segid)}
	seed, err := tools.IntegerToBytes(rand)
	if err != nil {
		return false, errors.Wrap(err, "tools.IntegerToBytes err")
	}
	var sealedCid = make([]cid.Cid, 0)
	for i := 0; i < len(sealcid); i++ {
		tmp, err := cid.Parse(sealcid[i])
		if err != nil {
			return false, errors.Wrap(err, "cid.Parse err")
		}
		sealedCid = append(sealedCid, tmp)
	}
	isValid := VerifyFileInterval(sectorId, abi.RegisteredSealProof(segtype), seed, sealedCid, pf)
	return isValid, nil
}
