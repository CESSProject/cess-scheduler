package chain

import (
	"cess-scheduler/configs"
	. "cess-scheduler/internal/logger"
	"cess-scheduler/tools"
	"encoding/binary"

	"github.com/centrifuge/go-substrate-rpc-client/v4/signature"
	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

// Get miner information on the chain
func GetMinerDataOnChain(addr string) (Chain_MinerItems, int, error) {
	var (
		err   error
		mdata Chain_MinerItems
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic] %v", err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}
	var pre []byte
	if configs.NewTestAddr {
		pre = tools.ChainCessTestPrefix
	} else {
		pre = tools.SubstratePrefix
	}
	pub, err := tools.DecodeToPub(addr, pre)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[DecodeToPub]")
	}

	b, err := types.EncodeToBytes(types.NewAccountID(pub))
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[EncodeToBytes]")
	}
	key, err := types.CreateStorageKey(meta, State_Sminer, Sminer_MinerItems, b)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return mdata, configs.Code_404, errors.New("[value is empty]")
	}
	return mdata, configs.Code_200, nil
}

// Get all miner information on the cess chain
func GetAllMinerDataOnChain() ([]CessChain_AllMinerInfo, int, error) {
	var (
		err   error
		mdata []CessChain_AllMinerInfo
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic] [%v.%v] [err:%v]", State_Sminer, Sminer_AllMinerItems, err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	key, err := types.CreateStorageKey(meta, State_Sminer, Sminer_AllMinerItems)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return mdata, configs.Code_404, errors.New("[value is empty]")
	}
	return mdata, configs.Code_200, nil
}

// Get miner information on the cess chain
func GetMinerDetailsById(id uint64) (Chain_MinerDetails, int, error) {
	var (
		err   error
		mdata Chain_MinerDetails
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic] [%v.%v] [err:%v]", State_Sminer, Sminer_MinerDetails, err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	eraIndexSerialized := make([]byte, 8)
	binary.LittleEndian.PutUint64(eraIndexSerialized, id)
	key, err := types.CreateStorageKey(meta, State_Sminer, Sminer_MinerDetails, types.NewBytes(eraIndexSerialized))
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return mdata, configs.Code_404, errors.New("Not found miner")
	}
	return mdata, configs.Code_200, nil
}

// Query file meta info
func GetFileMetaInfoOnChain(fileid string) (FileMetaInfo, int, error) {
	var (
		err   error
		mdata FileMetaInfo
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic] [%v.%v] [err:%v]", State_FileBank, FileMap_FileMetaInfo, err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	b, err := types.EncodeToBytes(fileid)
	if err != nil {
		return mdata, configs.Code_400, errors.Wrap(err, "[EncodeToBytes]")
	}

	key, err := types.CreateStorageKey(meta, State_FileBank, FileMap_FileMetaInfo, types.NewBytes(b))
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return mdata, configs.Code_404, errors.Errorf("[%v not folund]", fileid)
	}
	return mdata, configs.Code_200, nil
}

// Query Scheduler info
func GetSchedulerInfoOnChain() ([]SchedulerInfo, int, error) {
	var (
		err   error
		mdata = make([]SchedulerInfo, 0)
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic] [%v.%v] [err:%v]", State_FileMap, FileMap_SchedulerInfo, err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return mdata, configs.Code_500, errors.Wrapf(err, "[%v.%v:GetMetadataLatest]", State_FileMap, FileMap_SchedulerInfo)
	}

	key, err := types.CreateStorageKey(meta, State_FileMap, FileMap_SchedulerInfo)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrapf(err, "[%v.%v:CreateStorageKey]", State_FileMap, FileMap_SchedulerInfo)
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &mdata)
	if err != nil {
		return mdata, configs.Code_500, errors.Wrapf(err, "[%v.%v:GetStorageLatest]", State_FileMap, FileMap_SchedulerInfo)
	}
	if !ok {
		return mdata, configs.Code_404, errors.Errorf("[%v.%v:value is empty]", State_FileMap, FileMap_SchedulerInfo)
	}
	return mdata, configs.Code_200, nil
}

//
func GetSchedulerPukFromChain() (Chain_SchedulerPuk, int, error) {
	var (
		err  error
		data Chain_SchedulerPuk
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()
	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	key, err := types.CreateStorageKey(meta, State_FileMap, FileMap_SchedulerPuk)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return data, configs.Code_404, errors.New("public key not found")
	}
	return data, configs.Code_200, nil
}

//
func GetProofsFromChain(prk string) ([]Chain_Proofs, int, error) {
	var (
		err  error
		data []Chain_Proofs
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()

	keyring, err := signature.KeyringPairFromSecret(prk, 0)
	if err != nil {
		return data, configs.Code_400, errors.Wrap(err, "[KeyringPairFromSecret]")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	key, err := types.CreateStorageKey(meta, State_SegmentBook, SegmentBook_UnVerifyProof, keyring.PublicKey)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return data, configs.Code_404, errors.New("public key not found")
	}
	return data, configs.Code_200, nil
}

//
func GetAddressByPrk(prk string) (string, error) {
	keyring, err := signature.KeyringPairFromSecret(prk, 0)
	if err != nil {
		return "", errors.Wrap(err, "[KeyringPairFromSecret]")
	}
	var pre []byte
	if configs.NewTestAddr {
		pre = tools.ChainCessTestPrefix
	} else {
		pre = tools.SubstratePrefix
	}
	acc, err := tools.Encode(keyring.PublicKey, pre)
	if err != nil {
		return "", errors.Wrap(err, "[Encode]")
	}
	return acc, nil
}

//
func GetFileRecoveryByAcc(prk string) ([]types.Bytes, int, error) {
	var (
		err  error
		data []types.Bytes
	)
	api := getSubstrateApi_safe()
	defer func() {
		releaseSubstrateApi()
		err := recover()
		if err != nil {
			Err.Sugar().Errorf("[panic]: %v", err)
		}
	}()

	keyring, err := signature.KeyringPairFromSecret(prk, 0)
	if err != nil {
		return data, configs.Code_400, errors.Wrap(err, "[KeyringPairFromSecret]")
	}

	meta, err := api.RPC.State.GetMetadataLatest()
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[GetMetadataLatest]")
	}

	key, err := types.CreateStorageKey(meta, State_FileBank, FileBank_FileRecovery, keyring.PublicKey)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[CreateStorageKey]")
	}

	ok, err := api.RPC.State.GetStorageLatest(key, &data)
	if err != nil {
		return data, configs.Code_500, errors.Wrap(err, "[GetStorageLatest]")
	}
	if !ok {
		return data, configs.Code_404, errors.New("public key not found")
	}
	return data, configs.Code_200, nil
}
